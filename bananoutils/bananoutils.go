package bananoutils

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/blake2b"

	// We've forked golang's ed25519 implementation
	// to use blake2b instead of sha3
	"github.com/bbedward/crypto/ed25519"
	"github.com/golang/glog"
)

// nano uses a non-standard base32 character set.
const EncodeNano = "13456789abcdefghijkmnopqrstuwxyz"

var NanoEncoding = base32.NewEncoding(EncodeNano)

func Reversed(str []byte) (result []byte) {
	for i := len(str) - 1; i >= 0; i-- {
		result = append(result, str[i])
	}
	return result
}

func ValidateAddress(account Account) bool {
	_, err := AddressToPub(account)

	return err == nil
}

func AddressToPub(account Account) (public_key []byte, err error) {
	address := string(account)

	if address[:4] == "xrb_" || address[:4] == "ban_" {
		address = address[4:]
	} else if address[:5] == "nano_" {
		address = address[5:]
	} else {
		return nil, errors.New("invalid address format")
	}
	// A valid nano address is 64 bytes long
	// First 5 are simply a hard-coded string nano_ for ease of use
	// The following 52 characters form the address, and the final
	// 8 are a checksum.
	// They are base 32 encoded with a custom encoding.
	if len(address) == 60 {
		// The nano address string is 260bits which doesn't fall on a
		// byte boundary. pad with zeros to 280bits.
		// (zeros are encoded as 1 in nano's 32bit alphabet)
		key_b32nano := "1111" + address[0:52]
		input_checksum := address[52:]

		key_bytes, err := NanoEncoding.DecodeString(key_b32nano)
		if err != nil {
			return nil, err
		}
		// strip off upper 24 bits (3 bytes). 20 padding was added by us,
		// 4 is unused as account is 256 bits.
		key_bytes = key_bytes[3:]

		// nano checksum is calculated by hashing the key and reversing the bytes
		valid := NanoEncoding.EncodeToString(GetAddressChecksum(key_bytes)) == input_checksum
		if valid {
			return key_bytes, nil
		} else {
			return nil, errors.New("invalid address checksum")
		}
	}

	return nil, errors.New("invalid address format")
}

func GetAddressChecksum(pub ed25519.PublicKey) []byte {
	hash, err := blake2b.New(5, nil)
	if err != nil {
		panic("Unable to create hash")
	}

	hash.Write(pub)
	return Reversed(hash.Sum(nil))
}

func PubKeyToAddress(pub ed25519.PublicKey) Account {
	// Pubkey is 256bits, base32 must be multiple of 5 bits
	// to encode properly.
	// Pad the start with 0's and strip them off after base32 encoding
	padded := append([]byte{0, 0, 0}, pub...)
	address := NanoEncoding.EncodeToString(padded)[4:]
	checksum := NanoEncoding.EncodeToString(GetAddressChecksum(pub))

	return Account("ban_" + address + checksum)
}

func KeypairFromPrivateKey(private_key string) (ed25519.PublicKey, ed25519.PrivateKey) {
	private_bytes, _ := hex.DecodeString(private_key)
	pub, priv, _ := ed25519.GenerateKey(bytes.NewReader(private_bytes))

	return pub, priv
}

func KeypairFromSeed(seed io.Reader, index uint32) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	// This seems to be the standard way of producing wallets.

	// We hash together the seed with an address index and use
	// that as the private key. Whenever you "add" an address
	// to your wallet the wallet software increases the index
	// and generates a new address.
	hash, err := blake2b.New(32, nil)
	if err != nil {
		panic("Unable to create hash")
	}

	if err != nil {
		panic("Invalid seed")
	}

	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, index)
	seed_data, err := io.ReadAll(seed)
	if err != nil {
		return nil, nil, err
	}

	hash.Write(seed_data)
	hash.Write(bs)

	seed_bytes := hash.Sum(nil)
	pub, priv, err := ed25519.GenerateKey(bytes.NewReader(seed_bytes))

	if err != nil {
		panic("Unable to generate ed25519 key")
	}

	return pub, priv, nil
}

// Generate a private key and the first public account key
func GeneratePrivateKeyAndFirstPublicAddress() (string, Account, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		glog.Fatal("Could not get a crypto random value.")
		return "", "", err
	}

	pubKey, _, err := KeypairFromSeed(bytes.NewReader(key), 0)
	if err != nil {
		return "", "", err
	}
	account := PubKeyToAddress(pubKey)
	return hex.EncodeToString(key), account, nil
}

func GenerateKey() (ed25519.PublicKey, ed25519.PrivateKey) {
	pubkey, privkey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic("Unable to generate ed25519 key")
	}

	return pubkey, privkey
}
