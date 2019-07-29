package main

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/rsbondi/multifund/rpc"
	"github.com/rsbondi/multifund/wallet"
)

const FundMultiDescription = `Use external wallet funding feature to build a transaction to fund multiple channels
provide {utxos} as an array of {"txid":vout} and {ids} in the form of {"id":sats, "id2":sats, ..."idn":sats}!`

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
	if rate.Error != nil {
		return nil, errors.New(rate.Error.Message)
	}
	fee := wallet.Satoshis(rate.Result.(*wallet.EstimateSmartFeeResult).Feerate/1000.0) * uint64(150+50*len(*chans)) // crude size calc

	recipamt := int64(0)
	for i, c := range *chans {
		result, err := rpc.FundChannelStart(c.Id, c.Amount)
		if err != nil {
			return nil, err
		}
		amt := int64(c.Amount) // difference in wire and glightning
		inamt += uint64(c.Amount)
		outputs[c.Id] = &wallet.Outputs{Vout: i, Amount: amt, Address: result.FundingAddress}
		recipamt += amt
		recipients = append(recipients, &wallet.TxRecipient{Address: result.FundingAddress, Amount: amt})
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
	recipients = append(recipients, &wallet.TxRecipient{Address: change, Amount: int64(utxoamt-fee) - recipamt})
	tx, err := wallet.CreateTransaction(recipients, utxos, &chaincfg.RegressionNetParams)
	if err != nil {
		return nil, err
	}

	wally.Sign(&tx, utxos)

	return tx.String(), nil

}
