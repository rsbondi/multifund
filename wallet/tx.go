package wallet

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

type Transaction struct {
	TxId     string
	Signed   []byte
	Unsigned []byte
}

type Outputs struct {
	Vout   int    `json:"vout"`
	Amount int64  `json:"amount"`
	Script []byte `json:"script"`
}

func (t *Transaction) String() string {
	if t.Signed != nil {
		return hex.EncodeToString(t.Signed)
	} else {
		return hex.EncodeToString(t.Unsigned)
	}
}

type TxRecipient struct {
	Address string
	Amount  int64
}

func CreateTransaction(destinations []*TxRecipient, utxos []UTXO, network *chaincfg.Params) (Transaction, error) {
	var transaction Transaction
	tx := wire.NewMsgTx(2)

	for _, utxo := range utxos {
		txIn := wire.NewTxIn(&utxo.OutPoint, nil, nil)
		tx.AddTxIn(txIn)
	}

	for _, destination := range destinations {
		destinationAddress, err := btcutil.DecodeAddress(destination.Address, network)
		if err != nil {
			log.Printf("unable to decode address: %s\n", err.Error())
			return Transaction{}, err
		}
		destinationPkScript, _ := txscript.PayToAddrScript(destinationAddress)
		tx.AddTxOut(wire.NewTxOut(destination.Amount, destinationPkScript))
	}

	var unsignedTx bytes.Buffer
	tx.Serialize(&unsignedTx)
	transaction.Unsigned = unsignedTx.Bytes()
	return transaction, nil
}

// Outputs:
// A P2PKH (1... address) output is 34 vbytes.
// A P2SH (3... address) output is 32 vbytes.
// A P2WPKH (bc1q... address of length 42) output is 31 vbytes.
// A P2WSH (bc1q... address of length 62) output is 43 vbytes.
// Inputs:
// A P2PKH spend with a compressed public key is 149 vbytes.
// A P2WPKH spend is 68 vbytes.
// A P2SH-P2WPKH spend is 93 vbytes.
// https://bitcoin.stackexchange.com/questions/87275/how-to-calculate-segwit-transaction-fee-in-bytes
// Pieter Wuille

func OutputFeeSats(destinations []*TxRecipient, network *chaincfg.Params) uint64 {
	total := 0
	for _, d := range destinations {
		addr, err := btcutil.DecodeAddress(d.Address, network)
		if err != nil {
			log.Printf("unable to decode address: %s\n", err.Error())
			return uint64(0)
		}
		pks, _ := txscript.PayToAddrScript(addr)

		if txscript.IsPayToScriptHash(pks) {
			total += 32
		} else if txscript.IsPayToWitnessPubKeyHash(pks) {
			total += 31
		} else if txscript.IsPayToWitnessScriptHash(pks) {
			total += 43
		} else {
			total += 34
		}
	}
	return uint64(total)
}

func InputFeeSats(utxos []UTXO, network *chaincfg.Params) uint64 {
	total := 0
	for _, u := range utxos {
		addr, err := btcutil.DecodeAddress(u.Address, network)
		if err != nil {
			log.Printf("unable to decode address: %s\n", err.Error())
			return uint64(0)
		}
		pks, _ := txscript.PayToAddrScript(addr)

		if txscript.IsPayToScriptHash(pks) {
			total += 93
		} else if txscript.IsPayToWitnessPubKeyHash(pks) {
			total += 68
		} else if txscript.IsPayToWitnessScriptHash(pks) {
			total += 93
		} else {
			total += 149
		}

	}
	return uint64(total)
}
