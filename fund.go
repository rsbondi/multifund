package main

import (
	"bytes"
	"log"

	"github.com/btcsuite/btcd/wire"
	"github.com/niftynei/glightning/glightning"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/rsbondi/multifund/wallet"
)

const FundMultiDescription = `Use external wallet funding feature to build a transaction to fund multiple channels
{channels} is an array of object{"id" string, "satoshi" int, "announce" bool}`

type MultiChannel struct {
	Channels []glightning.FundChannelStart `json:"channels"`
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

type ConnectAndFundChannelRequest struct {
	Id       string  `json:"id"`
	Host     string  `json:"host,omitempty"`
	Port     float64 `json:"port,omitempty"`
	Amount   uint64  `json:"satoshi"`
	FeeRate  string  `json:"feerate,omitempty"`
	Announce bool    `json:"announce"`
}

type MultiChannelWithConnect struct {
	Channels []ConnectAndFundChannelRequest `json:"channels"`
}

func (m *MultiChannelWithConnect) Call() (jrpc2.Result, error) {
	return connectAndCreateMulti(&m.Channels)
}

func (f *MultiChannelWithConnect) Name() string {
	return "connect_fund_multi"
}

func (f *MultiChannelWithConnect) New() interface{} {
	return &MultiChannelWithConnect{}
}

func connectAndCreateMulti(chans *[]ConnectAndFundChannelRequest) (jrpc2.Result, error) {
	createChans := make([]glightning.FundChannelStart, 0)
	for _, c := range *chans {
		_, err := fundr.Lightning.Connect(c.Id, c.Host, uint(c.Port))
		if err != nil {
			return nil, err
		}
		newone := glightning.FundChannelStart{
			Id:       c.Id,
			Amount:   c.Amount,
			FeeRate:  c.FeeRate,
			Announce: c.Announce,
		}
		createChans = append(createChans, newone)
	}

	return createMulti(&createChans)
}

func createMulti(chans *[]glightning.FundChannelStart) (jrpc2.Result, error) {
	info, err := fundr.GetChannelAddresses(chans)
	if err != nil {
		cancelMulti(chans)
		return nil, err
	}

	tx, err := wallet.CreateTransaction(info.Recipients, info.Utxos, fundr.BitcoinNet)
	if err != nil {
		return nil, err
	}

	fundr.Wally.Sign(&tx, info.Utxos)
	wtx := wire.NewMsgTx(2)
	r := bytes.NewReader(tx.Signed)
	wtx.Deserialize(r)
	tx.TxId = wtx.TxHash().String()

	channels, err := fundr.CompleteChannels(tx, info.Outputs)
	if err != nil {
		cancelMultiExt(info.Outputs)
		return nil, err
	}

	txid, err := fundr.Bitcoin.SendTx(tx.String())
	if err != nil {
		cancelMultiExt(info.Outputs)
		return nil, err
	}

	return struct {
		Tx       string   `json:"tx"`
		Txid     string   `json:"txid"`
		Channels []string `json:"channels"`
	}{
		tx.String(),
		txid,
		channels,
	}, nil
}

func cancelMulti(chans *[]glightning.FundChannelStart) {
	for _, ch := range *chans {
		_, err := fundr.Lightning.CancelFundChannel(ch.Id)
		if err != nil {
			log.Printf("fundchannel_cancel error: %s", err.Error())
		}
	}
}

func cancelMultiExt(outputs map[string]*wallet.Outputs) {
	for k, _ := range outputs {
		_, err := fundr.Lightning.CancelFundChannel(k)
		if err != nil {
			log.Printf("channel cancel error: %s", err.Error())
		}
	}
}
