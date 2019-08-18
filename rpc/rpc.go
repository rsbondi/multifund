package rpc

// this is where the missing glightning rpc calls are implemented, eventually to be moved to glightning

import (
	"github.com/niftynei/glightning/glightning"
)

var lightning *glightning.Lightning

func Init(g *glightning.Lightning) {
	lightning = g
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
	Network            string `json:"network"`
}

func ListConfigs() (*ListConfigsResponse, error) {
	result := &ListConfigsResponse{}
	req := &ListConfigsRequest{}
	err := lightning.Request(req, result)
	return result, err
}
