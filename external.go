package main

import (
	"bytes"
	"encoding/hex"
	"log"

	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/niftynei/glightning/glightning"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/rsbondi/multifund/wallet"
)

const FundExternalDescription = `Use external wallet funding feature to provide addresses for external device for creating channels to fund multiple channels
{channels} is an array of object{"id" string, "satoshi" int, "announce" bool}`

type MultiChannelExternal struct {
	Channels []glightning.FundChannelStart `json:"channels"`
}

func (m *MultiChannelExternal) Call() (jrpc2.Result, error) {
	return createMultiExt(&m.Channels)
}

func (m *MultiChannelExternal) Name() string {
	return "fund_multi_start"
}

func (m *MultiChannelExternal) New() interface{} {
	return &MultiChannelExternal{}
}

const FundExternalCompleteDescription = `Complete a request started with fund_multi_start by providing an externally created transaction`

type MultiChannelExternalComplete struct {
	Tx string `json:"tx"`
}

func (m *MultiChannelExternalComplete) Call() (jrpc2.Result, error) {
	return completeMultiExt(m.Tx)
}

func (m *MultiChannelExternalComplete) Name() string {
	return "fund_multi_complete"
}

func (m *MultiChannelExternalComplete) New() interface{} {
	return &MultiChannelExternalComplete{}
}

var outputs map[string]*wallet.Outputs

func completeMultiExt(raw string) (jrpc2.Result, error) {
	b, err := hex.DecodeString(raw)
	tx := wallet.Transaction{
		Signed: b,
	}
	wtx := wire.NewMsgTx(2)
	r := bytes.NewReader(tx.Signed)
	wtx.Deserialize(r)
	tx.TxId = wtx.TxHash().String()

	channels, err := fundr.CompleteChannels(tx, outputs)
	if err != nil {
		closeMulti(outputs)
		return nil, err
	}

	txid, err := fundr.Bitcoin.SendTx(tx.String())
	if err != nil {
		closeMulti(outputs)
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

func createMultiExt(chans *[]glightning.FundChannelStart) (jrpc2.Result, error) {
	outputs = make(map[string]*wallet.Outputs, 0)
	addresses := make([]string, 0)

	for i, c := range *chans {
		result, err := fundr.Lightning.StartFundChannel(c.Id, c.Amount, c.Announce, nil)
		if err != nil {
			log.Printf("fund start error: %s", err.Error())
			return nil, err
		}
		addr, err := btcutil.DecodeAddress(result, fundr.BitcoinNet)
		addr.ScriptAddress()

		amt := int64(c.Amount) // difference in wire and glightning
		outputs[c.Id] = &wallet.Outputs{Vout: uint16(i), Amount: amt, Script: addr.ScriptAddress()}
		addresses = append(addresses, result)
	}

	return addresses, nil
}
