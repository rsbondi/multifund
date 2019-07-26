package wallet

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/rsbondi/multifund/rpc"
)

type BitcoinWallet struct {
	rpchost     string
	rpcport     int
	rpcuser     string
	rpcpassword string
}

func NewBitcoinWallet() *BitcoinWallet {
	cfg, err := rpc.ListConfigs()
	if err != nil {
		log.Fatal(err)
	}
	var host, user, pass string
	var port int
	if cfg.BitcoinRpcConnect != "" && cfg.BitcoinRpcUser != "" && cfg.BitcoinRpcPassword != "" {
		user = cfg.BitcoinRpcUser
		pass = cfg.BitcoinRpcPassword
		connect := strings.Split(cfg.BitcoinRpcConnect, ":")
		host = connect[0]
		port, err = strconv.Atoi(connect[1])
		if err != nil {
			log.Fatal(err)
		}
	} else if cfg.BitcoinRpcPort != 0 && cfg.BitcoinRpcUser != "" && cfg.BitcoinRpcPassword != "" {
		user = cfg.BitcoinRpcUser
		pass = cfg.BitcoinRpcPassword
		host = "127.0.0.1"
		port = cfg.BitcoinRpcPort
	} else {
		// TODO: get from ~/.bitcoin/bitcoin.conf
	}

	return &BitcoinWallet{
		rpchost:     host,
		rpcport:     port,
		rpcuser:     user,
		rpcpassword: pass,
	}
}

func (b *BitcoinWallet) Utxos(amt uint64) []wire.OutPoint {
	return nil
}

func (b *BitcoinWallet) ChangeAddress() string {
	return ""
}

func (b *BitcoinWallet) RpcPost(method string, params []interface{}, result interface{}) (interface{}, error) {
	url := fmt.Sprintf("%s:%d", b.rpchost, b.rpcport)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", b.rpcuser, b.rpcpassword)))))
	client := &http.Client{Timeout: time.Second * 10}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	js := json.NewDecoder(res.Body).Decode(result)
	return js, err
}

func SendTx(rawtx string) {

}
