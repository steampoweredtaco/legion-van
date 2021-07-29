package bananoutils_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/steampoweredtaco/legion-van/bananoutils"
)

func TestKeypairFromSeed(t *testing.T) {
	privateKey, err := hex.DecodeString("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil {
		t.Error(err)
	}
	expectedAddress := "ban_3wtsduys8b7jkbfwwfzx3jgpgpsi9b8zurfe9bp1p5cdxkqiz7a5wxcoo7ba"
	pub, _, err := bananoutils.KeypairFromSeed(bytes.NewReader(privateKey), 0)
	if err != nil {
		t.Error(err)
	}
	address := bananoutils.PubKeyToAddress(pub)
	if address != bananoutils.Account(expectedAddress) {
		t.Errorf("expected %s to be equal to %s", address, expectedAddress)
	}

}
