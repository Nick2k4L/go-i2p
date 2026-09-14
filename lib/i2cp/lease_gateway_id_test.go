package i2cp

import (
	"encoding/binary"
	common "github.com/go-i2p/common/data"
	"github.com/go-i2p/go-i2p/lib/tunnel"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeaseEntryUsesRemoteGatewayID(t *testing.T) {
	var gateway common.Hash
	gateway[0] = 1
	state := &tunnel.TunnelState{ID: 12, GatewayTunnelID: 34, Hops: []common.Hash{gateway}, CreatedAt: time.Now(), State: tunnel.TunnelReady, IsInbound: true}
	payload := make([]byte, 44)
	require.Equal(t, 44, encodeLeaseEntry(payload, 0, state, time.Now()))
	require.Equal(t, uint32(34), binary.BigEndian.Uint32(payload[32:36]))
	pool := tunnel.NewTunnelPool(nil)
	defer pool.Stop()
	pool.AddTunnel(state)
	id, hash, ok := makeReplyTunnelProvider(pool)()
	require.True(t, ok)
	require.Equal(t, tunnel.TunnelID(34), id)
	require.Equal(t, gateway, hash)
}
