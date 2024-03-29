package wallet

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
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

func NewInternalWallet(l *glightning.Lightning, net *chaincfg.Params, dir string) *InternalWallet {
	f, err := os.Open(dir + "/hsm_secret")
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
		dir:       dir,
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
	defer db.Close()
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

	// TODO: the coin selection should be refactored out to DRY it out
	//       result sets are of different data types, need to convert both wallet types to use the same standard
	//       then refactor to a common coin selection
	for rows.Next() {
		u := Outs{}
		err = rows.Scan(&u.PrevOutTx, &u.PrevOutIndex, &u.Value, &u.Scriptpubkey)
		if err != nil {
			log.Printf("cannot read database row: %s", err.Error())
		}
		unspent = append(unspent, u)
		sats := uint64(u.Value)
		if sats >= amt+fee && sats <= amt+fee+DUST_LIMIT {
			txid := u.PrevOutTx
			h, _ := chainhash.NewHash(txid)
			o := wire.NewOutPoint(h, uint32(u.PrevOutIndex))

			// this is hacky, converting to address so we can convert back to scriptpubkey later
			// bitcoin core uses the address to get the keys for signing, so maybe keep address and add scriptpubkey
			// maybe best is not save address, and attach a func to UTXO to get address from scriptpubkey
			_, addr, _, _ := txscript.ExtractPkScriptAddrs(u.Scriptpubkey, i.net)

			utxos := []UTXO{UTXO{uint64(u.Value), addr[0].String(), *o}}
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
		h, err := chainhash.NewHash(txid)
		if err != nil {
			log.Printf("unable to create hash from txid %s\n", err)
			return nil, err
		}

		o := wire.NewOutPoint(h, uint32(c.PrevOutIndex))
		_, addr, _, _ := txscript.ExtractPkScriptAddrs(c.Scriptpubkey, i.net)
		utxos = append(utxos, UTXO{uint64(c.Value), addr[0].String(), *o})
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

func (i *InternalWallet) Sign(tx *Transaction, utxos []UTXO) {
	partial := tx.Unsigned

	dbpath := i.dir + "/lightningd.sqlite3"
	db, err := sql.Open("sqlite3", dbpath)
	defer db.Close()
	if err != nil {
		log.Printf("cannot open database: %s", err.Error())
	}

	for _, u := range utxos {
		t, err := btcutil.NewTxFromBytes(partial)
		txToSign := t.MsgTx()

		txhash := fmt.Sprintf("%x", u.OutPoint.Hash.CloneBytes())

		keyindex := uint32(0)
		scriptpubkey := make([]byte, 0)
		err = db.QueryRow("SELECT keyindex, scriptpubkey FROM outputs WHERE HEX(prev_out_tx)=? COLLATE NOCASE and prev_out_index=?",
			txhash, u.OutPoint.Index).Scan(&keyindex, &scriptpubkey)

		if err != nil {
			log.Printf("cannot read database row: %s", err.Error())
		}
		key, err := i.master.Child(keyindex)
		if err != nil {
			log.Printf("cannot derive key for signing: %s", err.Error())
		}
		pk, _ := key.ECPrivKey()

		// need to find input index, not in sequence if created elsewhere
		vin := -1
	FindVin:
		for o, in := range txToSign.TxIn {
			for _, u := range utxos {
				if u.OutPoint.String() == in.PreviousOutPoint.String() {
					vin = o
					break FindVin
				}
			}
		}
		if vin == -1 {
			log.Printf("cannot create find input to sign: %s", err.Error())
			return
		}
		if txscript.IsPayToScriptHash(scriptpubkey) {
			h160 := btcutil.Hash160(pk.PubKey().SerializeCompressed())
			scriptpubkey = append([]byte{0x00, 0x14}, h160...)
			txToSign.TxIn[vin].SignatureScript = append([]byte{0x16}, scriptpubkey...)
		}

		witSig, err := txscript.WitnessSignature(txToSign, txscript.NewTxSigHashes(txToSign), vin, int64(u.Amount), scriptpubkey, txscript.SigHashAll, pk, true)
		if err != nil {
			log.Printf("cannot create sig script: %s", err.Error())
		}

		txToSign.TxIn[vin].Witness = witSig

		var txsig bytes.Buffer
		if err != nil {
			log.Printf("cannot sign: %s", err.Error())
		}
		err = txToSign.Serialize(&txsig)

		partial = txsig.Bytes()
	}
	tx.Signed = partial

}
