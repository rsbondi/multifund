package wallet

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"os/user"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/niftynei/glightning/glightning"
	"golang.org/x/crypto/hkdf"
)

type InternalWallet struct {
	lightning *glightning.Lightning
	master    *hdkeychain.ExtendedKey
	net       *chaincfg.Params
	dir       string
}

func NewInternalWallet(l *glightning.Lightning, net *chaincfg.Params) *InternalWallet {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	// f, err := os.Open(usr.HomeDir + "/.local/lib/python3.7/site-packages/lnet/run/lightning-4/hsm_secret")
	f, err := os.Open(usr.HomeDir + "/.lightning/hsm_secret") // TODO: listconfigs
	if err != nil {
		panic(err)
	}
	hsm_secret := make([]byte, 32)
	_, err = f.Read(hsm_secret)
	if err != nil {
		panic(err)
	}

	salt := []byte{0x0}
	bip32_seed := hkdf.New(sha256.New, hsm_secret, salt, []byte("bip32 seed"))
	b := make([]byte, 32)
	bip32_seed.Read(b)
	key, err := hdkeychain.NewMaster(b, net)
	if err != nil {
		panic(err)
	}
	base1, err := key.Child(0)
	master, err := base1.Child(0)

	return &InternalWallet{
		lightning: l,
		master:    master,
		net:       net,
		dir:       usr.HomeDir + "/.lightning", // TODO
	}
}

type Outs struct {
	PrevOutTx    []byte `db:"prev_out_tx"`
	PrevOutIndex int    `db:"prev_out_index"`
	Value        uint64 `db:"value"`
	Scriptpubkey []byte `db:"scriptpubkey"`
}

func (i *InternalWallet) Utxos(amt uint64, fee uint64) ([]UTXO, error) {
	dbpath := i.dir + "/lightningd.sqlite3"
	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Printf("cannot open database: %s", err.Error())
	}

	q := "SELECT prev_out_tx, prev_out_index, value, scriptpubkey FROM outputs WHERE spend_height IS NULL ORDER BY value"
	rows, err := db.Query(fmt.Sprintf(q))
	if err != nil {
		log.Printf("cannot execute query: %s", err.Error())
	}

	defer rows.Close()
	unspent := make([]Outs, 0)
	candidates := make([]*Outs, 0)

	cols, _ := rows.Columns()
	log.Printf("query columns: %s\n", cols)

	for rows.Next() {
		u := Outs{}
		err = rows.Scan(&u.PrevOutTx, &u.PrevOutIndex, &u.Value, &u.Scriptpubkey)
		log.Printf("row: %v\n", u)
		if err != nil {
			log.Printf("cannot read database row: %s", err.Error())
		}
		unspent = append(unspent, u)
		sats := uint64(u.Value)
		if sats == amt+fee {
			txid := u.PrevOutTx
			if err != nil {
				log.Printf("unable to decode txid %s\n", err)

			}
			h, _ := chainhash.NewHash(reverseBytes(txid))
			o := wire.NewOutPoint(h, uint32(u.PrevOutIndex))
			utxos := []UTXO{UTXO{uint64(u.Value), "", *o}}
			return utxos, nil
		}
		if sats > amt+fee+DUST_LIMIT {
			candidates = append(candidates, &u)
			break
		}
	}

	if len(candidates) == 0 {
		// unspent is sorted so grabbing the largest first should give us the least input count to tx
		sats := uint64(0)
		for i := len(unspent) - 1; i >= 0; i-- {
			u := unspent[i]
			sats += uint64(u.Value)
			candidates = append(candidates, &unspent[i])
			if sats > amt+fee+DUST_LIMIT {
				break
			}
		}
	}

	if len(candidates) == 0 {
		return nil, errors.New("no utxo candidates available")
	}

	utxos := make([]UTXO, 0)
	for _, c := range candidates {
		txid := c.PrevOutTx
		if err != nil {
			log.Printf("unable to decode txid %s\n", err)

		}
		h, err := chainhash.NewHash(reverseBytes(txid))
		if err != nil {
			log.Printf("unable to create hash from txid %s\n", err)
			return nil, err
		}

		o := wire.NewOutPoint(h, uint32(c.PrevOutIndex))

		utxos = append(utxos, UTXO{uint64(c.Value), "", *o})
	}

	return utxos, nil

}

func (i *InternalWallet) ChangeAddress() string {
	addr, err := i.lightning.NewAddr()
	if err != nil {
		return ""
	}
	return addr
}

func (b *InternalWallet) Sign(tx *Transaction, utxos []UTXO) {
	// sig, err := pk.Sign([]byte(tx.Unsigned))

}
