package main

import (
	"bytes"
	"errors"
	"log"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/rsbondi/multifund/rpc"
	"github.com/rsbondi/multifund/wallet"
)

const FundMultiDescription = `Use external wallet funding feature to build a transaction to fund multiple channels
{channels} is an array of object{"id" string, "satoshi" int, "announce" bool}`

type MultiChannel struct {
	Channels []rpc.FundChannelStartRequest
}

func (m *MultiChannel) Call() (jrpc2.Result, error) {
	return createMulti(&m.Channels)
}

func (f *MultiChannel) Name() string {
	return "fund_multi"
}

func (f *MultiChannel) New() interface{} {
	return &MultiChannel{}
}

func createMulti(chans *[]rpc.FundChannelStartRequest) (jrpc2.Result, error) {
	var recipients = make([]*wallet.TxRecipient, 0)
	outputs = make(map[string]*wallet.Outputs, 0)
	inamt := uint64(0)
	rate := bitcoin.EstimateSmartFee(100)
	kb := uint64(160 + 70*len(*chans)) // crude size calc
	feerate := rate.Result.(*wallet.EstimateSmartFeeResult).Feerate / 1000.0

	var fee uint64
	if feerate == 0.0 {
		log.Println("unable to estimate fee rate, using default")
		fee = 2 * kb
	} else {
		fee = wallet.Satoshis(feerate) * kb
	}

	// TODO: get utxos and loop channels to get total to make sure we have enough funds

	recipamt := int64(0)
	for _, c := range *chans {
		inamt += uint64(c.Amount)
	}

	var wally wallet.Wallet
	switch wallettype {
	case wallet.WALLET_BITCOIN:
		wally = bitcoin

	case wallet.WALLET_INTERNAL:
	case wallet.WALLET_EXTERNAL:
		resp := &outputs
		return resp, nil

	}
	wally = bitcoin // TODO
	change := wally.ChangeAddress()
	utxos, err := wally.Utxos(inamt, fee)
	utxoamt := uint64(0)
	for _, u := range utxos {
		utxoamt += u.Amount
	}

	if inamt > utxoamt+fee {
		return nil, errors.New("Insufficient funds, Need more coin")
	}

	for i, c := range *chans {
		result, err := rpc.FundChannelStart(c.Id, c.Amount)
		if err != nil {
			return nil, err
		}
		amt := int64(c.Amount) // difference in wire and glightning
		outputs[c.Id] = &wallet.Outputs{Vout: i, Amount: amt, Address: result.FundingAddress}
		recipamt += amt
		recipients = append(recipients, &wallet.TxRecipient{Address: result.FundingAddress, Amount: amt})
	}

	log.Printf("adding change %d %d %d\n", utxoamt, fee, recipamt)
	recipients = append(recipients, &wallet.TxRecipient{Address: change, Amount: int64(utxoamt-fee) - recipamt})
	tx, err := wallet.CreateTransaction(recipients, utxos, &chaincfg.RegressionNetParams)
	if err != nil {
		return nil, err
	}

	wally.Sign(&tx, utxos)
	wtx := wire.NewMsgTx(2)
	r := bytes.NewReader(tx.Signed)
	wtx.Deserialize(r)
	tx.TxId = wtx.TxHash().String()

	for k, o := range outputs {
		log.Printf("calling fundchannel_complete %s %s %d", k, tx.TxId, o.Vout)
		_, err := rpc.FundChannelComplete(k, tx.TxId, o.Vout)
		if err != nil {
			return nil, err
		}

	}

	txid, err := bitcoin.SendTx(tx.String())
	if err != nil {
		return nil, err
	}

	return txid, nil

}
