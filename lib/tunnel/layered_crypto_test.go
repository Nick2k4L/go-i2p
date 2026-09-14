package tunnel

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestLayeredCryptoIndependentThreeHopVector(t *testing.T) {
	// Fixture generated with Python cryptography AES ECB/CBC primitives, using
	// IV encrypt -> CBC encrypt -> IV encrypt for each of three remote hops.
	// Header and plaintext are byte(i); keys are byte(hop*64+i) and +32.
	encrypted, err := os.ReadFile("testdata/three_hop_aes.bin")
	require.NoError(t, err)
	keys := make([]LayerKeys, 3)
	for h := range keys {
		for i := 0; i < 32; i++ {
			keys[h].Layer[i] = byte(h*64 + i)
			keys[h].IV[i] = byte(h*64 + 32 + i)
		}
	}
	c, err := NewLayeredCrypto(keys)
	require.NoError(t, err)
	original := append([]byte(nil), encrypted...)
	plain, err := c.Decrypt(encrypted)
	require.NoError(t, err)
	expected := make([]byte, 1028)
	for i := range expected {
		expected[i] = byte(i)
	}
	require.Equal(t, expected, plain)
	require.Equal(t, original, encrypted, "decrypt must not mutate received bytes")
	output, err := c.Encrypt(expected)
	require.NoError(t, err)
	require.Equal(t, encrypted, output)
	_, err = c.Decrypt(encrypted[:1024])
	require.Error(t, err)
	zero, err := NewLayeredCrypto(nil)
	require.NoError(t, err)
	output, err = zero.Decrypt(expected)
	require.NoError(t, err)
	require.Equal(t, expected, output)
}
