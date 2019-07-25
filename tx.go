package main

import (
	"bytes"
	"encoding/hex"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

type Transaction struct {
	TxId       string `json:"txid"`
	UnsignedTx string `json:"unsignedtx"`
}

type TxRecipient struct {
	Address string
	Amount  int64
}

func CreateTransaction(destinations []*TxRecipient, txHash string, vout uint32, network *chaincfg.Params) (Transaction, error) {
	var transaction Transaction
	tx := wire.NewMsgTx(2)

	utxoHash, _ := chainhash.NewHashFromStr(txHash)
	utxo := wire.NewOutPoint(utxoHash, vout)
	txIn := wire.NewTxIn(utxo, nil, nil)
	tx.AddTxIn(txIn)

	for _, destination := range destinations {
		destinationAddress, err := btcutil.DecodeAddress(destination.Address, network)
		if err != nil {
			return Transaction{}, err
		}
		destinationPkScript, _ := txscript.PayToAddrScript(destinationAddress)
		tx.AddTxOut(wire.NewTxOut(destination.Amount, destinationPkScript))
	}

	var unsignedTx bytes.Buffer
	tx.Serialize(&unsignedTx)
	transactionHash := tx.TxHash()
	transaction.TxId = transactionHash.String()
	transaction.UnsignedTx = hex.EncodeToString(unsignedTx.Bytes())
	return transaction, nil
}
