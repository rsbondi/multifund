package rpc

// this is where the missing glightning rpc calls are implemented, eventually to be moved to glightning

import (
	"github.com/niftynei/glightning/glightning"
)

var lightning *glightning.Lightning

func Init(g *glightning.Lightning) {
	lightning = g
}

type FundChannelStartRequest struct {
	Id       string  `json:"id"`
	Amount   float64 `json:"satoshi"`
	FeeRate  string  `json:"feerate,omitempty"`
	Announce bool    `json:"announce"`
}

func (f *FundChannelStartRequest) Name() string {
	return "fundchannel_start"
}

type FundChannelStartResponse struct {
	FundingAddress string `json:"funding_address"`
}

func FundChannelStart(id string, amt float64) (*FundChannelStartResponse, error) {
	var result FundChannelStartResponse
	req := &FundChannelStartRequest{}
	req.Id = id
	req.Amount = amt
	req.Announce = true
	err := lightning.Request(req, &result)
	return &result, err
}

type FundChannelCompleteRequest struct {
	Id    string `json:"id"`
	Txid  string `json:"txid"`
	Txout int    `json:"txout"`
}

func (f *FundChannelCompleteRequest) Name() string {
	return "fundchannel_complete"
}

type FundChannelCompleteResponse struct {
	ChannelId string `json:"channel_id"`
	Secured   bool   `json:"commitments_secured"`
}

func FundChannelComplete(id string, txid string, vout int) (*FundChannelCompleteResponse, error) {
	result := &FundChannelCompleteResponse{}
	req := &FundChannelCompleteRequest{}
	req.Id = id
	req.Txid = txid
	req.Txout = vout
	err := lightning.Request(req, result)
	return result, err
}

type ListConfigsRequest struct{}

func (c *ListConfigsRequest) Name() string {
	return "listconfigs"
}

type ListConfigsResponse struct {
	BitcoinRpcUser     string `json:"bitcoin-rpcuser"`
	BitcoinRpcPassword string `json:"bitcoin-rpcpassword"`
	BitcoinRpcConnect  string `json:"bitcoin-rpcconnect"` // host:port
	BitcoinRpcPort     string `json:"bitcoin-rpcport"`    // can this be used if connect is just the host? it works with localhost
}

func ListConfigs() (*ListConfigsResponse, error) {
	result := &ListConfigsResponse{}
	req := &ListConfigsRequest{}
	err := lightning.Request(req, result)
	return result, err
}
