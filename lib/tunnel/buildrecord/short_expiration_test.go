package buildrecord

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestShortRequestLifetime(t *testing.T) {
	record := BuildRequestRecord{RequestTime: time.Now()}
	wire := record.ShortBytes()
	// ECIES short records put the lifetime at cleartext offset 48.
	// The network currently supports only ten-minute tunnels.
	if got := binary.BigEndian.Uint32(wire[96:100]); got != 600 {
		t.Fatalf("wire lifetime = %d seconds, want 600", got)
	}
}
