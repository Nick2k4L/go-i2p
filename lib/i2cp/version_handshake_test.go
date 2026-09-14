package i2cp

import (
	"encoding/binary"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetDateSetDateWireVersion(t *testing.T) {
	server := newTestI2CPServer(t, "")
	conn := startServerAndConnect(t, server)
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))

	// I2CP String uses a one-byte length. Send and inspect wire bytes directly
	// so a shared encoder/decoder cannot hide an incompatible version prefix.
	request := []byte{0, 0, 0, 7, 32, 6, '0', '.', '9', '.', '6', '7'}
	before := time.Now().UnixMilli()
	_, err := conn.Write(request)
	require.NoError(t, err)

	var header [5]byte
	_, err = io.ReadFull(conn, header[:])
	require.NoError(t, err)
	require.Equal(t, byte(33), header[4])
	require.Equal(t, uint32(15), binary.BigEndian.Uint32(header[:4]))

	var payload [15]byte
	_, err = io.ReadFull(conn, payload[:])
	require.NoError(t, err)
	require.Equal(t, []byte{6, '0', '.', '9', '.', '6', '7'}, payload[8:])
	date := int64(binary.BigEndian.Uint64(payload[:8]))
	require.GreaterOrEqual(t, date, before)
	require.LessOrEqual(t, date, time.Now().UnixMilli())
}

func TestParseClientVersionI2CPString(t *testing.T) {
	maxVersion := strings.Repeat("v", 255)
	tests := []struct {
		name    string
		payload []byte
		want    string
	}{
		{name: "legacy omitted version"},
		{name: "empty version", payload: []byte{0}},
		{name: "version", payload: []byte{6, '0', '.', '9', '.', '6', '7'}, want: "0.9.67"},
		{name: "trailing authentication mapping", payload: []byte{6, '0', '.', '9', '.', '6', '7', 0, 0}, want: "0.9.67"},
		{name: "missing version bytes", payload: []byte{6}},
		{name: "truncated version", payload: []byte{6, '0', '.', '9'}},
		{name: "maximum string length", payload: append([]byte{255}, maxVersion...), want: maxVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, parseClientVersion(tt.payload))
		})
	}
}

func TestGetDateAuthenticationMappingOffset(t *testing.T) {
	mapping := []byte{0, 3, 1, 'x', ';'}
	payload := append([]byte{6, '0', '.', '9', '.', '6', '7'}, mapping...)
	require.Equal(t, mapping, parseGetDatePayload(payload))
	for _, short := range [][]byte{nil, {0}, {6}, {6, '0', '.', '9'}, {6, '0', '.', '9', '.', '6', '7'}} {
		require.Nil(t, parseGetDatePayload(short))
	}
}
