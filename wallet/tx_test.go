package wallet

import (
	"encoding/hex"
	// "crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"

	// "github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	// "hash"
	// "math"
	// "os"
	// "os/user"
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

	/*
		test data from:
		https://github.com/hkjn/lnhw/tree/master/doc/hsmd#populate_secretstuff

		secretstuff.hsm_secret: 1e14cd384691a92120f6702742ca0e06951aeee57e91b5e137526c0a6c0867f4
		bip32 seed: 5a9bed3df01abd7aa0f260120530aaf1eea3ac2744648975dc23cfb25a71045d
		bip32 master key: xprv9s21ZrQH143K4Swn4rdeRhPLPfN1qJtA6yFR5RBTpU2s614zG7ELFMN6YAW4AGH3jZRJUUQBuPt9pJ5D5jzq65PKWCBy6xNarQAcgofD3Xr
		secretstuff.bip32: xprv9wYsM6fW2kCzYkSeu3AFZrJ7bk4Ny3w3L5UaLDKLxLizJcacRNGCVwouqJSNNqoi4DGdA6cf3kFEUDvmSdpCyQu8sYg4x44cpVbUFVpSXkc

		or in more details:
		chain code: 262f1245ae343a30b93da83adb0de95fe2c0e2751ed285ecfce030ebe35ed261
		priv_key[33]: 00 e558f771f5b6dcdd5073a876dbf3b8363377de0db33808ecdf54a76571f9db7d
		hash160: 5333f23b0664b8405de8eaf95e2c4417f930d544
		pubkey: 0242625c2cf7f546b9786efddd7c33cc1ec2e7cc2ba28e3838426e6415fd10ba09
		fingerprint: 6896baf1
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

	// tx, err := CreateTransaction(
	// 	[]*TxRecipient{&TxRecipient{"bcrt1q52g6zdr7la83fl3scx7an3znuu4dzy4paf2w2xx6u7j4af83pwzsa0ynrt", 91234}},
	// 	"6cb7c43cf84a4f7f88748b5abbe20fcc0d351c1331801fc51c3d41023beac47c", 0, net)

	// sig, err := pk.Sign([]byte(tx.UnsignedTx))
	// fmt.Printf("%s\n%x\n03a9b2177d30fc4df49abb0016e2dbb1fd95466ff93c9945dfd5fb5de5591239bb\n%x\n", wif.String(), pk.PubKey().SerializeCompressed(), sig.Serialize())
}
