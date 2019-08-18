package wallet

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

type BitcoinWallet struct {
	rpchost     string
	rpcport     string
	rpcuser     string
	rpcpassword string
}

func NewBitcoinWallet(cfg map[string]interface{}) *BitcoinWallet {
	var host, user, pass string
	var port string
	if cfg["bitcoin-rpcconnect"] != nil && cfg["bitcoin-rpcuser"] != nil && cfg["bitcoin-rpcpassword"] != nil {
		user = cfg["bitcoin-rpcuser"].(string)
		pass = cfg["bitcoin-rpcpassword"].(string)
		connect := strings.Split(cfg["bitcoin-rpcconnect"].(string), ":")
		host = connect[0]
		port = connect[1]
	} else if cfg["bitcoin-rpcport"] != nil && cfg["bitcoin-rpcuser"] != nil && cfg["bitcoin-rpcpassword"] != nil {
		user = cfg["bitcoin-rpcuser"].(string)
		pass = cfg["bitcoin-rpcpassword"].(string)
		host = "127.0.0.1"
		port = cfg["bitcoin-rpcport"].(string)
	} else {
		userdir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		f, err := os.Open(userdir + "/.bitcoin/bitcoin.conf")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		lines := strings.Split(string(b), "\n")
		for _, line := range lines {
			if len(line) > 0 {
				valid := strings.TrimSpace(strings.Split(line, "#")[0]) // ignore comments
				configmatch(valid, "^rpcuser=(.+)$", &user)
				configmatch(valid, "^rpcpassword=(.+)$", &pass)
				configmatch(valid, "^rpcbind=(.+)$", &host)
				configmatch(valid, "^rpcport=(.+)$", &port)
			}
		}
		if host == "" {
			host = "127.0.0.1"
		}
		if port == "" {
			for _, line := range lines {
				if len(line) > 0 {
					valid := strings.TrimSpace(strings.Split(line, "#")[0])
					netmatch(valid, "^([^=]+)=.+$", &port)
				}
			}
			if port == "" {
				port = "8332"
			}
		}
		if user == "" {
			// TODO: cookie auth
			log.Fatal(errors.New("Can not access bitcoin wallet"))
		}
	}

	if host == "" || port == "" || user == "" || pass == "" {
		panic("can not initialize bitcoin wallet, configuration information not found, try adding to your lightning config")
	}

	return &BitcoinWallet{
		rpchost:     host,
		rpcport:     port,
		rpcuser:     user,
		rpcpassword: pass,
	}
}

func configmatch(line string, exp string, setme *string) {
	r, _ := regexp.Compile(exp)
	u := r.FindStringSubmatch(line)
	if len(u) > 0 {
		*setme = u[1]
	}
}

func netmatch(line string, exp string, setme *string) {
	r, _ := regexp.Compile(exp)
	u := r.FindStringSubmatch(line)
	if len(u) > 0 {
		if u[1] == "regtest" {
			*setme = "18443"
		} else if u[1] == "testnet" {
			*setme = "18332"
		}
	}
}

type bitcoinUtxo struct {
	Txid          string  `json:"txid"`
	Vout          uint32  `json:"vout"`
	Amount        float64 `json:"amount"`
	Address       string  `json:"address"`
	ScriptPubKey  string  `json:"scriptPubKey"`
	RedeemScript  string  `json:"redeemScript"`
	Confirmations uint    `json:"confirmations"`
}

type empty struct{}

func makeResult(r interface{}) RpcResult {
	e := &RpcError{}
	result := RpcResult{
		Result: r,
		Error:  e,
	}
	return result
}

type ByAmount []bitcoinUtxo

func (a ByAmount) Len() int           { return len(a) }
func (a ByAmount) Less(i, j int) bool { return a[i].Amount < a[j].Amount }
func (a ByAmount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (b *BitcoinWallet) Utxos(amt uint64, fee uint64) ([]UTXO, error) {
	minconf := uint(3)
	unspent := make([]bitcoinUtxo, 0)
	result := makeResult(&unspent)
	b.RpcPost("listunspent", []empty{}, &result)
	sort.Sort(ByAmount(unspent))
	candidates := make([]*bitcoinUtxo, 0)
	for _, u := range unspent {
		sats := Satoshis(u.Amount)

		// best case, but least likely, no change needed
		if sats >= amt+fee && sats <= amt+fee+DUST_LIMIT && u.Confirmations > minconf {
			txid, err := hex.DecodeString(u.Txid)
			if err != nil {
				log.Printf("unable to decode txid %s\n", err)

			}
			h, _ := chainhash.NewHash(reverseBytes(txid))
			o := wire.NewOutPoint(h, u.Vout)
			utxos := []UTXO{UTXO{Satoshis(u.Amount), u.Address, *o}}
			return utxos, nil
		}
		if sats > amt+fee+DUST_LIMIT && u.Confirmations > minconf {
			candidates = append(candidates, &u)
			break
		}
	}
	if len(candidates) == 0 {
		// unspent is sorted so grabbing the largest first should give us the least input count to tx
		sats := uint64(0)
		for i := len(unspent) - 1; i >= 0; i-- {
			u := unspent[i]
			sats += Satoshis(u.Amount)
			candidates = append(candidates, &unspent[i])
			if sats > amt+fee+DUST_LIMIT && u.Confirmations > minconf {
				break
			}
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no utxo candidates available")
	}

	utxos := make([]UTXO, 0)
	for _, c := range candidates {
		txid, err := hex.DecodeString(c.Txid)
		if err != nil {
			log.Printf("unable to decode txid %s\n", err)

		}
		h, err := chainhash.NewHash(reverseBytes(txid))
		if err != nil {
			log.Printf("unable to create hash from txid %s\n", err)
			return nil, err
		}

		o := wire.NewOutPoint(h, c.Vout)

		utxos = append(utxos, UTXO{Satoshis(c.Amount), c.Address, *o})
	}
	return utxos, nil
}

type EstimateSmartFeeResult struct {
	Feerate float64 `json:"feerate"`
}

func (b *BitcoinWallet) EstimateSmartFee(target uint) RpcResult {
	fee := EstimateSmartFeeResult{}
	result := makeResult(&fee)
	b.RpcPost("estimatesmartfee", []uint{target}, &result)

	return result
}

func (b *BitcoinWallet) ChangeAddress() string {
	addr := ""
	result := makeResult(&addr)
	b.RpcPost("getrawchangeaddress", []string{"bech32"}, &result)
	return addr
}

type RpcError struct {
	Message string `json:"message"`
}

type RpcResult struct {
	Result interface{} `json:"result"`
	Error  *RpcError   `json:"error"`
}

type RpcCall struct {
	Id      int64       `json:"id"`
	Method  string      `json:"method"`
	JsonRpc string      `json:"jsonrpc"`
	Params  interface{} `json:"params"`
}

func (b *BitcoinWallet) RpcPost(method string, params interface{}, result interface{}) error {
	url := fmt.Sprintf("http://%s:%s", b.rpchost, b.rpcport)
	rpcCall := &RpcCall{
		Id:      time.Now().Unix(),
		Method:  method,
		JsonRpc: "2.0",
		Params:  params,
	}
	jsoncall, err := json.Marshal(rpcCall)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsoncall))
	basic := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", b.rpcuser, b.rpcpassword))))
	req.Header.Set("Authorization", basic)
	client := &http.Client{Timeout: time.Second * 10}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(result)

	return err
}

type BitcoinSignResult struct {
	Hex      string `json:"hex"`
	Complete bool   `json:"complete"`
}

func (b *BitcoinWallet) Sign(tx *Transaction, utxos []UTXO) {
	pks := make([]string, 0)
	for _, u := range utxos {
		key := ""
		result := makeResult(&key)
		b.RpcPost("dumpprivkey", []string{u.Address}, &result)
		pks = append(pks, key)

	}

	raw := BitcoinSignResult{}
	rawresult := makeResult(&raw)
	b.RpcPost("signrawtransactionwithkey", []interface{}{tx.String(), pks}, &rawresult)

	signed, err := hex.DecodeString(raw.Hex)
	if err != nil {
		log.Printf("error signing tx: %s", err.Error())

		return
	}
	tx.Signed = signed

}

type BitcoinSendResult struct {
	Hex string `json:"hex"`
}

func (b *BitcoinWallet) SendTx(rawtx string) (string, error) {
	bs := ""
	result := makeResult(&bs)
	b.RpcPost("sendrawtransaction", []string{rawtx}, &result)
	if result.Error != nil {
		log.Printf("Transaction Send Error: %s", result.Error.Message)
		return "", errors.New(result.Error.Message)
	}

	return bs, nil

}
