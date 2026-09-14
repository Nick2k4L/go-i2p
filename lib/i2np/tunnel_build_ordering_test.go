package i2np

import (
	"errors"
	"testing"

	common "github.com/go-i2p/common/data"
	"github.com/go-i2p/common/router_info"
	"github.com/go-i2p/go-i2p/lib/tunnel"
	"github.com/go-i2p/go-i2p/lib/tunnel/build"
	"github.com/stretchr/testify/require"
)

type orderingBuildProvider struct {
	connect func(common.Hash) (build.BuildSession, error)
}

func (p orderingBuildProvider) GetSessionByHash(h common.Hash) (build.BuildSession, error) {
	return p.connect(h)
}

type orderingBuildSession func([]byte) error

func (s orderingBuildSession) Send(b []byte) error { return s(b) }

func TestInboundBuildReturnPathBeforeSend(t *testing.T) {
	for _, name := range []string{"connected", "unreachable", "send failure"} {
		failReturnPath := name == "unreachable"
		t.Run(name, func(t *testing.T) {
			first, _ := createTestHop(t)
			last, _ := createTestHop(t)
			firstHash, err := first.IdentHash()
			require.NoError(t, err)
			lastHash, err := last.IdentHash()
			require.NoError(t, err)
			tm := NewTunnelManager(&SimpleMockPeerSelector{})
			defer tm.Stop()
			result := &tunnel.TunnelBuildResult{
				TunnelID: 123, IsInbound: true, UseShortBuild: true,
				ReplyIVs: make([][16]byte, 2),
				Hops:     []router_info.RouterInfo{*first, *last},
				Records:  []tunnel.BuildRequestRecord{createTestTunnelRecord(t), createTestTunnelRecord(t)},
			}
			req := tunnel.BuildTunnelRequest{IsInbound: true}
			tm.trackPendingBuild(result, 456, req)
			var connections []common.Hash
			sent := false
			returnErr := errors.New("return peer unavailable")
			tm.buildSessionProv = orderingBuildProvider{connect: func(h common.Hash) (build.BuildSession, error) {
				connections = append(connections, h)
				if failReturnPath && h == lastHash {
					return nil, returnErr
				}
				return orderingBuildSession(func([]byte) error {
					sent = true
					require.Equal(t, []common.Hash{lastHash, firstHash}, connections)
					pending := tm.replyProcessor.GetPendingBuildInfo(result.TunnelID)
					require.NotNil(t, pending, "immediate replies must find their crypto context")
					require.Equal(t, result.ReplyKeys, pending.ReplyKeys)
					require.Len(t, pending.NoiseHashes, 2)
					require.Equal(t, result.NoiseHashes, pending.NoiseHashes)
					require.Equal(t, result.ReplyKeys, tm.pendingBuilds[456].replyKeys)
					if name == "send failure" {
						return errors.New("send failed")
					}
					return nil
				}), nil
			}}
			err = tm.sendBuildMessage(result, 456, req)
			if failReturnPath {
				require.ErrorContains(t, err, "failed to establish inbound return path")
				require.False(t, sent)
				require.Equal(t, []common.Hash{lastHash}, connections)
			} else if name == "send failure" {
				require.ErrorContains(t, err, "send failed")
				require.True(t, sent)
				tm.cleanupFailedBuild(result.TunnelID, 456, true)
				require.Nil(t, tm.replyProcessor.GetPendingBuildInfo(result.TunnelID))
				require.False(t, tm.HasPendingInboundBuild(456))
			} else {
				require.NoError(t, err)
				require.True(t, sent)
			}
		})
	}
}

type inboundReplyTracker struct {
	mockBuildReplyProcessor
	pending map[int]bool
}

func (p *inboundReplyTracker) HasPendingInboundBuild(id int) bool { return p.pending[id] }

func TestReturningInboundShortBuildDispatch(t *testing.T) {
	for _, pending := range []bool{true, false} {
		t.Run(map[bool]string{true: "pending", false: "unsolicited"}[pending], func(t *testing.T) {
			processor := NewMessageProcessor()
			tracker := &inboundReplyTracker{pending: map[int]bool{456: pending}}
			processor.SetBuildReplyProcessor(tracker)
			msg := NewBaseI2NPMessage(I2NPMessageTypeShortTunnelBuild)
			msg.SetMessageID(456)
			data := make([]byte, 1+ShortBuildRecordSize)
			data[0] = 1
			msg.SetData(data)
			err := processor.processShortTunnelBuildMessage(msg)
			if pending {
				require.NoError(t, err)
				require.True(t, tracker.called)
				require.Equal(t, 456, tracker.messageID)
				require.Equal(t, data[1:], tracker.handler.GetRawReplyRecords()[0])
			} else {
				require.ErrorContains(t, err, "participant manager not configured")
				require.False(t, tracker.called)
			}
		})
	}
}

func TestHasPendingInboundBuild(t *testing.T) {
	tm := NewTunnelManager(nil)
	defer tm.Stop()
	tm.pendingBuilds[1] = &buildRequest{isInbound: true}
	tm.pendingBuilds[2] = &buildRequest{isInbound: false}
	require.True(t, tm.HasPendingInboundBuild(1))
	require.False(t, tm.HasPendingInboundBuild(2))
	require.False(t, tm.HasPendingInboundBuild(3))
}
