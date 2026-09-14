package router

import (
	"testing"
	"time"

	common "github.com/go-i2p/common/data"
	"github.com/go-i2p/go-i2p/lib/tunnel"
	"github.com/stretchr/testify/require"
)

type replyGatewayBuildRecorder struct {
	requests []tunnel.BuildTunnelRequest
}

func (b *replyGatewayBuildRecorder) BuildTunnel(req tunnel.BuildTunnelRequest) (*tunnel.BuildTunnelResult, error) {
	b.requests = append(b.requests, req)
	return &tunnel.BuildTunnelResult{TunnelID: 9000}, nil
}

func TestWireReplyTunnelProvidersGatewayID(t *testing.T) {
	const localTunnelID tunnel.TunnelID = 1001
	for _, tc := range []struct {
		name            string
		gatewayTunnelID tunnel.TunnelID
		wantReplyID     tunnel.TunnelID
	}{
		{
			name:            "remote gateway receive ID",
			gatewayTunnelID: 2002,
			wantReplyID:     2002,
		},
		{
			name:        "missing gateway ID falls back to local ID",
			wantReplyID: localTunnelID,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ourHash := common.Hash{1}
			gatewayHash := common.Hash{2}
			inboundConfig := tunnel.DefaultPoolConfig()
			inboundConfig.IsInbound = true
			inboundPool := tunnel.NewTunnelPoolWithConfig(nil, inboundConfig)
			t.Cleanup(inboundPool.Stop)
			outboundPool := tunnel.NewTunnelPool(nil)
			t.Cleanup(outboundPool.Stop)
			inboundPool.SetRouterHash(ourHash)
			outboundPool.SetRouterHash(ourHash)

			inboundPool.AddTunnel(&tunnel.TunnelState{
				ID:              localTunnelID,
				GatewayTunnelID: tc.gatewayTunnelID,
				Hops:            []common.Hash{gatewayHash},
				State:           tunnel.TunnelReady,
				CreatedAt:       time.Now(),
				IsInbound:       true,
			})
			outboundPool.AddTunnel(&tunnel.TunnelState{
				ID:        3003,
				Hops:      []common.Hash{{3}},
				State:     tunnel.TunnelReady,
				CreatedAt: time.Now(),
			})
			(&Router{}).wireReplyTunnelProviders(inboundPool, outboundPool)

			builder := &replyGatewayBuildRecorder{}
			outboundPool.SetTunnelBuilder(builder)
			require.NoError(t, outboundPool.RetryTunnelBuild(4004, false, 1))
			require.Len(t, builder.requests, 1)
			require.Equal(t, tc.wantReplyID, builder.requests[0].ReplyTunnelID,
				"the outbound endpoint must address the inbound gateway's receive tunnel")
			require.Equal(t, gatewayHash, builder.requests[0].ReplyGateway)

			inboundPool.SetTunnelBuilder(builder)
			require.NoError(t, inboundPool.RetryTunnelBuild(5005, true, 1))
			require.Len(t, builder.requests, 2)
			require.Zero(t, builder.requests[1].ReplyTunnelID,
				"an inbound endpoint must deliver locally without forwarding into another tunnel")
			require.Equal(t, ourHash, builder.requests[1].ReplyGateway)
		})
	}
}
