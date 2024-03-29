package main

import (
	"bytes"
	"errors"
	"log"

	"github.com/btcsuite/btcd/wire"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/rsbondi/multifund/wallet"
)

const WithdrawMultiDescription = `Withdraw funds to multiple addresses
{channels} is an array of object{"id" string, "satoshi" int, "announce" bool}`

type MultiWithdrawRequest struct {
	Destination string  `json:"destination"`
	Satoshi     float64 `json:"satoshi"`
	FeeRate     string  `json:"feerate,omitempty"`
}

type MultiWithdraw struct {
	Targets []MultiWithdrawRequest `json:"destinations"`
}

func (m *MultiWithdraw) Call() (jrpc2.Result, error) {
	return withdrawMulti(&m.Targets)
}

func (f *MultiWithdraw) Name() string {
	return "withdraw_multi"
}

func (f *MultiWithdraw) New() interface{} {
	return &MultiWithdraw{}
}

func withdrawMulti(targets *[]MultiWithdrawRequest) (jrpc2.Result, error) {
	var recipients = make([]*wallet.TxRecipient, 0)
	outamt := uint64(0)
	rate := fundr.Bitcoin.EstimateSmartFee(100)
	bytesEstimate := uint64(160 + 70*len(*targets)) // crude size calc
	feerate := rate.Result.(*wallet.EstimateSmartFeeResult).Feerate / 1000.0

	var fee uint64
	if feerate == 0.0 {
		log.Println("unable to estimate fee rate, using default")
		fee = 2 * bytesEstimate
	} else {
		fee = wallet.Satoshis(feerate) * bytesEstimate
	}

	recipamt := int64(0)
	for _, c := range *targets {
		outamt += uint64(c.Satoshi)
	}

	internal := fundr.InternalWallet()
	change := internal.ChangeAddress()
	utxos, err := internal.Utxos(outamt, fee)
	utxoamt := uint64(0)
	for _, u := range utxos {
		utxoamt += u.Amount
	}

	if outamt > utxoamt+fee {
		return nil, errors.New("Insufficient funds, Need more coin")
	}

	for _, c := range *targets {
		amt := int64(c.Satoshi) // difference in wire and glightning
		recipamt += amt
		recipients = append(recipients, &wallet.TxRecipient{Address: c.Destination, Amount: amt})
	}

	if utxoamt-fee > wallet.DUST_LIMIT { // no change if dust, save on tx fee
		recipients = append(recipients, &wallet.TxRecipient{Address: change, Amount: int64(utxoamt-fee) - recipamt})
		vsize := wallet.InputFeeSats(utxos, fundr.BitcoinNet) + wallet.OutputFeeSats(recipients, fundr.BitcoinNet) + 11
		if feerate == 0.0 {
			fee = 2 * vsize
		} else {
			fee = wallet.Satoshis(feerate) * vsize
		}
		recipients[len(recipients)-1].Amount = int64(utxoamt-fee) - recipamt
	}
	tx, err := wallet.CreateTransaction(recipients, utxos, fundr.BitcoinNet)
	if err != nil {
		return nil, err
	}

	internal.Sign(&tx, utxos)
	wtx := wire.NewMsgTx(2)
	r := bytes.NewReader(tx.Signed)
	wtx.Deserialize(r)
	tx.TxId = wtx.TxHash().String()

	txid, err := fundr.Bitcoin.SendTx(tx.String())
	if err != nil {
		return nil, err
	}

	return struct {
		Tx   string `json:"tx"`
		Txid string `json:"txid"`
	}{
		tx.String(),
		txid,
	}, nil
}
