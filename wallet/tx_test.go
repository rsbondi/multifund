package wallet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil/hdkeychain"
	"golang.org/x/crypto/hkdf"
	"testing"
)

func TestCreateTransaction(t *testing.T) {
	utxoHash, _ := chainhash.NewHashFromStr("6cb7c43cf84a4f7f88748b5abbe20fcc0d351c1331801fc51c3d41023beac47c")
	o := []UTXO{UTXO{Amount: 91235, OutPoint: *wire.NewOutPoint(utxoHash, 0)}}
	transaction, err := CreateTransaction(
		[]*TxRecipient{&TxRecipient{"bcrt1q52g6zdr7la83fl3scx7an3znuu4dzy4paf2w2xx6u7j4af83pwzsa0ynrt", 91234}},
		o,
		&chaincfg.RegressionNetParams)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(transaction.Unsigned)
}

func TestDeriveKey(t *testing.T) {
	net := &chaincfg.RegressionNetParams
	salt := []byte{0x0}

	/*
		test data from:
		https://github.com/hkjn/lnhw/tree/master/doc/hsmd#populate_secretstuff
	*/
	hsm_secret, _ := hex.DecodeString("1e14cd384691a92120f6702742ca0e06951aeee57e91b5e137526c0a6c0867f4")
	bip32_seed := hkdf.New(sha256.New, hsm_secret, salt, []byte("bip32 seed"))
	b := make([]byte, 32)
	bip32_seed.Read(b)

	want := "5a9bed3df01abd7aa0f260120530aaf1eea3ac2744648975dc23cfb25a71045d"
	have := fmt.Sprintf("%x", b)
	if have != want {
		t.Errorf("unable to derive seed, want %s, have %s", want, have)
	}

	key, err := hdkeychain.NewMaster(b, net)
	if err != nil {
		t.Errorf("key creation error: %s", err.Error())
	}
	base1, err := key.Child(0)
	base, err := base1.Child(0)
	want = "e558f771f5b6dcdd5073a876dbf3b8363377de0db33808ecdf54a76571f9db7d"
	priv, _ := base.ECPrivKey()
	have = fmt.Sprintf("%s", priv.Serialize())
	fmt.Printf("%s\n%x\n", want, have)
	if have != want {
		t.Errorf("unable to derive master key, want %s, have %s", want, have)
	}

}
