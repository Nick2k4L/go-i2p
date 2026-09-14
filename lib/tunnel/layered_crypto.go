package tunnel

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	cryptotunnel "github.com/go-i2p/crypto/tunnel"
)

// LayerKeys are the negotiated AES keys for one remote tunnel hop.
type LayerKeys struct{ Layer, IV [32]byte }

type aesTunnelLayer struct{ layer, iv cipher.Block }

// LayeredCrypto removes the layers added by remote inbound hops. Keys are
// ordered from gateway to the last remote hop. The four-byte tunnel ID is
// outside the encrypted 1024-byte tunnel message and is preserved.
type LayeredCrypto struct{ layers []aesTunnelLayer }

func NewLayeredCrypto(keys []LayerKeys) (*LayeredCrypto, error) {
	c := &LayeredCrypto{}
	for _, k := range keys {
		layer, err := aes.NewCipher(k.Layer[:])
		if err != nil {
			return nil, err
		}
		iv, err := aes.NewCipher(k.IV[:])
		if err != nil {
			return nil, err
		}
		c.layers = append(c.layers, aesTunnelLayer{layer, iv})
	}
	return c, nil
}

func (c *LayeredCrypto) Type() cryptotunnel.TunnelEncryptionType {
	return cryptotunnel.TunnelEncryptionAES
}

// Decrypt reverses the double IV encryption and CBC layer at each hop.
func (c *LayeredCrypto) Decrypt(data []byte) ([]byte, error) {
	if len(data) != 1028 {
		return nil, fmt.Errorf("tunnel message: got %d bytes, want 1028", len(data))
	}
	out := append([]byte(nil), data...)
	for i := len(c.layers) - 1; i >= 0; i-- {
		hop := c.layers[i]
		hop.iv.Decrypt(out[4:20], out[4:20])
		cipher.NewCBCDecrypter(hop.layer, out[4:20]).CryptBlocks(out[20:], out[20:])
		hop.iv.Decrypt(out[4:20], out[4:20])
	}
	return out, nil
}

// Encrypt applies the hop layers in gateway-to-endpoint order.
func (c *LayeredCrypto) Encrypt(data []byte) ([]byte, error) {
	if len(data) != 1028 {
		return nil, fmt.Errorf("tunnel message: got %d bytes, want 1028", len(data))
	}
	out := append([]byte(nil), data...)
	for _, hop := range c.layers {
		hop.iv.Encrypt(out[4:20], out[4:20])
		cipher.NewCBCEncrypter(hop.layer, out[4:20]).CryptBlocks(out[20:], out[20:])
		hop.iv.Encrypt(out[4:20], out[4:20])
	}
	return out, nil
}

// OutboundCrypto pre-decrypts a tunnel frame so successive remote AES
// encryptions recover the original frame at the outbound endpoint.
type OutboundCrypto struct{ *LayeredCrypto }

func (c *OutboundCrypto) Encrypt(data []byte) ([]byte, error) { return c.LayeredCrypto.Decrypt(data) }
func (c *OutboundCrypto) Decrypt(data []byte) ([]byte, error) { return c.LayeredCrypto.Encrypt(data) }
