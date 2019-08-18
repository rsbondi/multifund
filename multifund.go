package main

import (
	"fmt"
	"log"
	"os"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/niftynei/glightning/glightning"
	"github.com/rsbondi/multifund/funder"
	"github.com/rsbondi/multifund/rpc"
	"github.com/rsbondi/multifund/wallet"
)

const VERSION = "0.0.1-WIP"

var plugin *glightning.Plugin

var fundr *funder.Funder

func main() {
	plugin = glightning.NewPlugin(onInit)
	fundr = &funder.Funder{}
	fundr.Lightning = glightning.NewLightning()
	rpc.Init(fundr.Lightning)

	registerOptions(plugin)
	registerMethods(plugin)

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("versiion: %s initialized for wallet type %s", VERSION, options["multi-wallet"])
	fundr.Lightningdir = config.LightningDir
	options["rpc-file"] = fmt.Sprintf("%s/%s", config.LightningDir, config.RpcFile)
	switch options["multi-wallet"] {
	case "bitcoin":
		fundr.Wallettype = wallet.WALLET_BITCOIN
	default:
		fundr.Wallettype = wallet.WALLET_INTERNAL
	}
	fundr.Lightning.StartUp(config.RpcFile, config.LightningDir)

	fundr.Bitcoin = wallet.NewBitcoinWallet()

	cfg, err := rpc.ListConfigs()
	if err != nil {
		log.Fatal(err)
	}

	switch cfg.Network {
	case "bitcoin":
		fundr.BitcoinNet = &chaincfg.MainNetParams
	case "regtest":
		fundr.BitcoinNet = &chaincfg.RegressionNetParams
	case "signet":
		panic("unsupported network")
	default:
		fundr.BitcoinNet = &chaincfg.TestNet3Params
	}

}

func registerOptions(p *glightning.Plugin) {
	p.RegisterOption(glightning.NewOption("multi-wallet", "Wallet to use for multi-channel open - internal or bitcoin", "internal"))
}

// fund_multi [{"id":"0265b6...", "satoshi": 20000, "announce":true}, {id, satoshi, announce}...]
func registerMethods(p *glightning.Plugin) {
	multi := glightning.NewRpcMethod(&MultiChannel{}, `Open multiple channels in single transaction`)
	multi.LongDesc = FundMultiDescription
	p.RegisterMethod(multi)

	multic := glightning.NewRpcMethod(&MultiChannelWithConnect{}, `Connects peers and opens multiple channels in single transaction`)
	multic.LongDesc = "{peers} consist of {id, host, port, satoshi, announce}"
	p.RegisterMethod(multic)

	multiw := glightning.NewRpcMethod(&MultiWithdraw{}, `Batch withdraw funds to multiple destinations`)
	multiw.LongDesc = `{destinations} consist of an array of{"destination": ADDRESS, "satoshi": n}`
	p.RegisterMethod(multiw)

}
