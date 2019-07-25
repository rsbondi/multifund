package main

import (
	"github.com/niftynei/glightning/jrpc2"
	"github.com/rsbondi/multifund/rpc"
)

const FundMultiDescription = `Use external wallet funding feature to build a transaction to fund multiple channels
provide {utxos} as an array of {"txid":vout} and {ids} in the form of {"id":sats, "id2":sats, ..."idn":sats}!`

type MultiChannel struct {
	Channels []rpc.FundChannelStartRequest
}

func (m *MultiChannel) Call() (jrpc2.Result, error) {
	return createMulti(m.Channels)
}

func (f *MultiChannel) Name() string {
	return "fund_multi"
}

func (f *MultiChannel) New() interface{} {
	return &MultiChannel{}
}

func createMulti(chans []rpc.FundChannelStartRequest) (interface{}, error) {
	for i, c := range chans {
		result, err := rpc.FundChannelStart(c.Id, c.Amount)
		if err != nil {
			return nil, err
		}
		outputs[c.Id] = &Outputs{i, c.Amount, result.FundingAddress}
	}

	return nil, nil
}
