package i2cp

import (
	"testing"
	"time"

	common "github.com/go-i2p/common/data"
	"github.com/go-i2p/go-i2p/lib/tunnel"
	"github.com/stretchr/testify/require"
)

type clientReplyRecordingBuilder struct {
	requests chan tunnel.BuildTunnelRequest
}

func (b *clientReplyRecordingBuilder) BuildTunnel(req tunnel.BuildTunnelRequest) (*tunnel.BuildTunnelResult, error) {
	select {
	case b.requests <- req:
	default:
	}
	return &tunnel.BuildTunnelResult{TunnelID: 9001}, nil
}

func TestClientOutboundPoolLeavesBuildReplyRoutingToRouter(t *testing.T) {
	routerHash := common.Hash{1}
	clientGateway := common.Hash{2}
	server := &Server{routerHash: routerHash, hasRouterHash: true}

	config := DefaultSessionConfig()
	config.OutboundTunnelCount = 1
	config.OutboundBackupQuantity = 1
	session, err := NewSession(7, nil, config)
	require.NoError(t, err)
	t.Cleanup(session.Stop)

	selector := &mockPeerSelector{}
	inboundConfig := tunnel.DefaultPoolConfig()
	inboundConfig.IsInbound = true
	inboundConfig.IsClientPool = true
	inboundPool := tunnel.NewTunnelPoolWithConfig(selector, inboundConfig)
	session.SetInboundPool(inboundPool)
	inboundPool.AddTunnel(&tunnel.TunnelState{
		ID:              41001,
		GatewayTunnelID: 41002,
		Hops:            []common.Hash{clientGateway},
		State:           tunnel.TunnelReady,
		CreatedAt:       time.Now(),
		IsInbound:       true,
	})
	require.Len(t, inboundPool.GetActiveTunnels(), 1)

	builder := &clientReplyRecordingBuilder{requests: make(chan tunnel.BuildTunnelRequest, 1)}
	outboundPool, err := server.createOutboundPoolWithConfig(session, config, builder, selector, inboundPool)
	require.NoError(t, err)

	select {
	case req := <-builder.requests:
		require.False(t, req.IsInbound)
		require.True(t, req.IsClientTunnel)
		require.Equal(t, session.ID(), req.ClientSessionID)
		require.Same(t, outboundPool, req.OwnerPool)
		require.Equal(t, routerHash, req.OurIdentity)
		// Router-encrypted build replies need an exploratory endpoint. Leaving
		// this unset lets the shared builder choose one despite an active client
		// inbound tunnel being available.
		require.Zero(t, req.ReplyTunnelID)
		require.Equal(t, routerHash, req.ReplyGateway)
	case <-time.After(5 * time.Second):
		t.Fatal("outbound pool did not request a tunnel build")
	}
}
