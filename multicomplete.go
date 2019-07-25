package main

import (
	"fmt"

	"github.com/niftynei/glightning/jrpc2"
)

const FundMultiCompleteDescription = `Use external wallet funding feature to build a transaction to fund multiple channels
{transactions} is an array of object{"id":"nodeid", "txid":"txid", "txout":vout}`

type MultiChannelComplete struct {
	Txs []FundChannelCompleteRequest
}

func (m *MultiChannelComplete) Call() (jrpc2.Result, error) {
	return createMultiComplete(m.Txs)
}

func (f *MultiChannelComplete) Name() string {
	return "fund_multi_complete"
}

func (f *MultiChannelComplete) New() interface{} {
	return &MultiChannelComplete{}
}

func createMultiComplete(txs []FundChannelCompleteRequest) (interface{}, error) {
	for _, t := range txs {
		result, err := FundChannelComplete(t.Id, t.Txid, t.Txout)
		if err != nil {
			return nil, err
		}
		fmt.Printf("%v", result) // not sure I really need result
	}
	return nil, nil
}
