package main

import (
	// "crypto/hmac"
	"crypto/sha256"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"

	// "github.com/btcsuite/btcutil"
	// "github.com/btcsuite/btcutil/hdkeychain"
	// "hash"
	// "math"
	// "os"
	// "os/user"
	"golang.org/x/crypto/hkdf"
	"testing"
)

func TestCreateTransaction(t *testing.T) {
	utxoHash, _ := chainhash.NewHashFromStr("6cb7c43cf84a4f7f88748b5abbe20fcc0d351c1331801fc51c3d41023beac47c")
	o := []*wire.OutPoint{wire.NewOutPoint(utxoHash, 0)}
	transaction, err := CreateTransaction(
		[]*TxRecipient{&TxRecipient{"bcrt1q52g6zdr7la83fl3scx7an3znuu4dzy4paf2w2xx6u7j4af83pwzsa0ynrt", 91234}},
		o,
		&chaincfg.RegressionNetParams)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(transaction.UnsignedTx)
}

// func hmac_sha256(data []byte, secret []byte) []byte {
// 	h := hmac.New(sha256.New, secret)
// 	h.Write([]byte(data))
// 	return h.Sum(nil)
// }

// const hash_len = 32

// func hkdf(length int, ikm []byte, salt []byte, info []byte) []byte {
// 	prk := hmac_sha256(salt, ikm)
// 	t := make([]byte, 0)
// 	okm := make([]byte, 0)

// 	for i := 0; i < int(math.Ceil(float64(length)/float64(hash_len))); i++ {
// 		t = hmac_sha256(prk, append(append(t, info...), byte(i+1)))
// 		okm = append(okm, t...)
// 	}
// 	return okm

// }

func TestDeriveKey(t *testing.T) {
	// net := &chaincfg.RegressionNetParams
	// usr, err := user.Current()
	// if err != nil {
	// 	panic(err)
	// }

	// f, err := os.Open(usr.HomeDir + "/.local/lib/python3.7/site-packages/lnet/run/lightning-4/hsm_secret")
	// f, err := os.Open(usr.HomeDir + "/.lightning/hsm_secret")
	// if err != nil {
	// 	panic(err)
	// }
	// b := make([]byte, 32)
	// _, err = f.Read(b)
	salt := []byte{0x0}

	hsm_secret := []byte("1e14cd384691a92120f6702742ca0e06951aeee57e91b5e137526c0a6c0867f4")
	bip32_seed := hkdf.New(sha256.New, hsm_secret, salt, []byte("bip32 seed"))
	b := make([]byte, 32)
	bip32_seed.Read(b)

	fmt.Printf("5a9bed3df01abd7aa0f260120530aaf1eea3ac2744648975dc23cfb25a71045d\n%x\n", b)

	// key, err := hdkeychain.NewMaster(bip32_seed, net)
	// if err != nil {
	// 	panic(err)
	// }
	// base1, err := key.Child(hdkeychain.HardenedKeyStart)
	// base, err := base1.Child(0)
	// child, err := base.Child(3)
	// pk, _ := child.ECPrivKey()
	// wif, err := btcutil.NewWIF(pk, net, true)

	// tx, err := CreateTransaction(
	// 	[]*TxRecipient{&TxRecipient{"bcrt1q52g6zdr7la83fl3scx7an3znuu4dzy4paf2w2xx6u7j4af83pwzsa0ynrt", 91234}},
	// 	"6cb7c43cf84a4f7f88748b5abbe20fcc0d351c1331801fc51c3d41023beac47c", 0, net)

	// sig, err := pk.Sign([]byte(tx.UnsignedTx))
	// fmt.Printf("%s\n%x\n03a9b2177d30fc4df49abb0016e2dbb1fd95466ff93c9945dfd5fb5de5591239bb\n%x\n", wif.String(), pk.PubKey().SerializeCompressed(), sig.Serialize())
}
