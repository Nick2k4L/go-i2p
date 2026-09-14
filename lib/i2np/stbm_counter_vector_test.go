package i2np

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Fixture generated with OpenSSL ChaCha20, key 00..1f, counter 1 and
// nonce 000000000200000000000000, as in i2pd Crypto.cpp.
func TestSTBMChaCha20CounterCompatibility(t *testing.T) {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}
	var record [ShortBuildRecordSize]byte
	if err := chacha20XORRecord(&record, key, 2); err != nil {
		t.Fatal(err)
	}
	expected, err := hex.DecodeString("a9a3fe55e7eaac6ecb4f3f70ccce61b5c490ac2b84b34fe1bda721d1cda42cf590195ab77c7de85b21e99188e1c4862541d96da2c49321b9eb12cc53bd4928d810414db3947937b17d2e7df8b3e836cefc0fa2cfb767e40d2dc333740989362d71dedcccd3ffc3714ceebfce26a614732ebf7f31a8ef883fd3140287464232bc201944fd4ce1afcb1bd85e4a11d1a239adf64132434cc0d0a72cd499b478ffee387584168a5d84ec805a0398e0a33ad51560867d3dfed8f38c3f0863fe30d310fafd4de70438858b7825c89e4398af2a517654d3bf1cfe405707")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(record[:], expected) {
		t.Fatal("STBM stream differs from counter-1 OpenSSL fixture")
	}
}
