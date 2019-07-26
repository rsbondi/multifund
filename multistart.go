package main

import (
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
	var recipients = make([]*TxRecipient, 0)
	outputs = make(map[string]*Outputs, 0)
	for i, c := range *chans {
		result, err := rpc.FundChannelStart(c.Id, c.Amount)
		if err != nil {
			return nil, err
		}
		amt := int64(c.Amount) // difference in wire and glightning
		outputs[c.Id] = &Outputs{i, amt, result.FundingAddress}
		recipients = append(recipients, &TxRecipient{result.FundingAddress, amt})
	}

	switch wallettype {
	case wallet.WALLET_BITCOIN:
		// use bitcoin rpc for utxos and change address
	case wallet.WALLET_INTERNAL:
		// use internall for utxos and change address

	}
	// TODO: get utxos and change address from wallet

	// TODO: create tx

	// TODO: call fundchannel_complete, if all is well broadcast

	resp := &outputs
	return resp, nil
}
