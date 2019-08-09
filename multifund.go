package main

import (
	"fmt"
	"log"
	"os"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/niftynei/glightning/glightning"
	"github.com/rsbondi/multifund/rpc"
	"github.com/rsbondi/multifund/wallet"
)

const VERSION = "0.0.1-WIP"

var plugin *glightning.Plugin
var lightning *glightning.Lightning
var wallettype int
var bitcoin *wallet.BitcoinWallet // we always use this at least for broadcasting the tx
var internalWallet wallet.Wallet
var bitcoinNet *chaincfg.Params
var lightningdir string

func main() {
	plugin = glightning.NewPlugin(onInit)
	lightning = glightning.NewLightning()
	rpc.Init(lightning)

	registerOptions(plugin)
	registerMethods(plugin)

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func InternalWallet() wallet.Wallet {
	if internalWallet == nil {
		internalWallet = wallet.NewInternalWallet(lightning, bitcoinNet, lightningdir)
	}
	return internalWallet
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("versiion: %s initialized for wallet type %s", VERSION, options["multi-wallet"])
	lightningdir = config.LightningDir
	options["rpc-file"] = fmt.Sprintf("%s/%s", config.LightningDir, config.RpcFile)
	switch options["multi-wallet"] {
	case "bitcoin":
		wallettype = wallet.WALLET_BITCOIN
	default:
		wallettype = wallet.WALLET_INTERNAL
	}
	lightning.StartUp(config.RpcFile, config.LightningDir)

	bitcoin = wallet.NewBitcoinWallet()

	cfg, err := rpc.ListConfigs()
	if err != nil {
		log.Fatal(err)
	}

	switch cfg.Network {
	case "bitcoin":
		bitcoinNet = &chaincfg.MainNetParams
	case "regtest":
		bitcoinNet = &chaincfg.RegressionNetParams
	case "signet":
		panic("unsupported network")
	default:
		bitcoinNet = &chaincfg.TestNet3Params
	}

}

func registerOptions(p *glightning.Plugin) {
	p.RegisterOption(glightning.NewOption("multi-wallet", "Wallet to use for multi-channel open - internal or bitcoin", "internal"))
}

// fund_multi [{"id":"0265b6...", "satoshi": 20000, "announce":true}, {id, satoshi, announce}...]
func registerMethods(p *glightning.Plugin) {
	multi := glightning.NewRpcMethod(&MultiChannel{}, `Open multiple channels in single transaction`)
	multi.LongDesc = FundMultiDescription
	multi.Usage = "channels"
	p.RegisterMethod(multi)

	multic := glightning.NewRpcMethod(&MultiChannelWithConnect{}, `Connects peers and opens multiple channels in single transaction`)
	multic.LongDesc = "{peers} consist of {id, host, port, satoshi, announce}"
	multic.Usage = "peers"
	p.RegisterMethod(multic)

	multiw := glightning.NewRpcMethod(&MultiWithdraw{}, `Batch withdraw funds to multiple destinations`)
	multiw.LongDesc = `{destinations} consist of an array of{"destination": ADDRESS, "satoshi": n}`
	multiw.Usage = "destinations"
	p.RegisterMethod(multiw)

}
