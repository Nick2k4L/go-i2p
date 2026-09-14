package i2np

import (
	common "github.com/go-i2p/common/data"
	"github.com/go-i2p/common/router_info"
	"github.com/go-i2p/go-i2p/lib/tunnel"
	"github.com/go-i2p/go-i2p/lib/tunnel/build"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClientBuildUsesOwningPool(t *testing.T) {
	hop, _ := createTestHop(t)
	selector := &MockPeerSelector{peers: []router_info.RouterInfo{*hop}}
	tm := NewTunnelManager(selector)
	defer tm.Stop()
	owner := tunnel.NewTunnelPoolWithConfig(selector, tunnel.DefaultPoolConfig())
	defer owner.Stop()
	tm.buildSessionProv = orderingBuildProvider{connect: func(common.Hash) (build.BuildSession, error) {
		return orderingBuildSession(func([]byte) error { return nil }), nil
	}}
	var hash common.Hash
	hash[0] = 1
	req := tunnel.BuildTunnelRequest{OwnerPool: owner, HopCount: 1, IsInbound: true, IsClientTunnel: true, ClientSessionID: 12, OurIdentity: hash, ReplyGateway: hash, UseShortBuild: true}
	id, _, err := tm.BuildTunnelFromRequest(req)
	require.NoError(t, err)
	state, ok := owner.GetTunnel(id)
	require.True(t, ok)
	_, ok = tm.inboundPool.GetTunnel(id)
	require.False(t, ok, "client tunnels must not enter exploratory pool")
	var messageID int
	for mid, pending := range tm.pendingBuilds {
		if pending.tunnelID == id {
			messageID = mid
		}
	}
	require.NotZero(t, messageID)
	// A delayed retry still belongs to this pool after pending correlation is gone.
	retry := tm.replyProcessor.GetPendingBuildInfo(id).retry
	require.NotNil(t, retry)
	capture := &captureBuildRequestBuilder{}
	owner.SetTunnelBuilder(capture)
	owner.SetClientSessionID(12)
	require.NoError(t, retry())
	require.Same(t, owner, capture.lastReq.OwnerPool)
	require.Equal(t, uint16(12), capture.lastReq.ClientSessionID)

	tm.updateTunnelStatesFromReply(messageID, []BuildResponseRecord{{Reply: TunnelBuildReplySuccess}}, nil)
	require.Equal(t, tunnel.TunnelReady, state.GetState())
	require.Len(t, owner.GetActiveTunnels(), 1)
	require.Empty(t, tm.inboundPool.GetActiveTunnels())
	removed := false
	state.OnRemove = func() { removed = true }
	owner.Stop()
	require.True(t, removed)
	require.Empty(t, owner.GetActiveTunnels())
	_, _, err = tm.BuildTunnelFromRequest(req)
	require.ErrorContains(t, err, "stopped")
}

func TestClientBuildExpiryUsesOwningPool(t *testing.T) {
	tm := NewTunnelManager(&SimpleMockPeerSelector{})
	defer tm.Stop()
	owner := tunnel.NewTunnelPool(&SimpleMockPeerSelector{})
	defer owner.Stop()
	state := &tunnel.TunnelState{ID: 123, State: tunnel.TunnelBuilding, CreatedAt: time.Now().Add(-2 * time.Minute), IsInbound: true}
	owner.AddTunnel(state)
	req := &buildRequest{ownerPool: owner, tunnelID: 123, isInbound: true, isClientTunnel: true, createdAt: state.CreatedAt}
	tm.markTunnelAsFailed(req)
	require.Equal(t, tunnel.TunnelFailed, state.GetState())
	require.Empty(t, tm.inboundPool.GetActiveTunnels())
}
