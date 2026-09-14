package i2cp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	common "github.com/go-i2p/common/data"
	"github.com/go-i2p/go-i2p/lib/i2np"
	"github.com/go-i2p/go-i2p/lib/tunnel"
	"github.com/stretchr/testify/require"
)

type outboundDeliveryCapture struct {
	id       uint32
	gateway  [32]byte
	messages [][]byte
}

func (c *outboundDeliveryCapture) ForwardToTunnel(id uint32, gateway [32]byte, b []byte) error {
	c.id = id
	c.gateway = gateway
	c.messages = append(c.messages, append([]byte(nil), b...))
	return nil
}
func (c *outboundDeliveryCapture) ForwardToRouter(_ [32]byte, _ []byte) error {
	return fmt.Errorf("unexpected router delivery")
}

func TestSendThroughGatewayTunnelDelivery(t *testing.T) {
	for _, size := range []int{92, 4096} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			keys := make([]tunnel.LayerKeys, 3)
			for h := range keys {
				for j := 0; j < 32; j++ {
					keys[h].Layer[j] = byte(h*64 + j)
					keys[h].IV[j] = byte(h*64 + 32 + j)
				}
			}
			state := &tunnel.TunnelState{ID: 123, Hops: []common.Hash{{1}, {2}, {3}}}
			state.SetLayerKeys(keys)
			remote, err := tunnel.NewLayeredCrypto(keys)
			require.NoError(t, err)
			identity, err := tunnel.NewLayeredCrypto(nil)
			require.NoError(t, err)
			endpoint, err := tunnel.NewEndpoint(123, identity, func([]byte) error { return fmt.Errorf("unexpected local delivery") })
			require.NoError(t, err)
			defer endpoint.Stop()
			capture := &outboundDeliveryCapture{}
			endpoint.SetForwarder(capture)
			frames := 0
			router := NewMessageRouter(nil, func(peer common.Hash, msg i2np.Message) error {
				require.Equal(t, state.Hops[0], peer)
				data, ok := msg.(*i2np.TunnelDataMessage)
				require.True(t, ok, "must send TunnelData, not raw garlic")
				require.Equal(t, state.GatewayID(), data.GetTunnelID())
				frame := make([]byte, 1028)
				binary.BigEndian.PutUint32(frame, uint32(data.GetTunnelID()))
				copy(frame[4:], data.GetTunnelData())
				plain, err := remote.Encrypt(frame)
				require.NoError(t, err)
				frames++
				return endpoint.Receive(plain)
			})
			garlic, err := i2np.WrapInGarlicMessage(bytes.Repeat([]byte{0x5a}, size))
			require.NoError(t, err)
			expected, err := garlic.MarshalBinary()
			require.NoError(t, err)
			gateway := common.Hash{9}
			require.NoError(t, router.sendThroughGateway(&Session{}, state, common.Hash{8}, garlic, tunnel.TunnelDelivery(456, gateway)))
			require.Equal(t, uint32(456), capture.id)
			require.Equal(t, [32]byte(gateway), capture.gateway)
			require.Equal(t, [][]byte{expected}, capture.messages)
			if size > 1000 {
				require.Greater(t, frames, 1)
			}
		})
	}
}
