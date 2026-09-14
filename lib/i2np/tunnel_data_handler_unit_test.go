package i2np

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"
)

// mockTunnelDataHandler implements TunnelDataHandler for testing
type mockTunnelDataHandler struct {
	called    bool
	msg       Message
	returnErr error
}

func (m *mockTunnelDataHandler) HandleTunnelData(msg Message) error {
	m.called = true
	m.msg = msg
	return m.returnErr
}

// TestSetTunnelDataHandler verifies the setter wires the handler correctly
func TestSetTunnelDataHandler(t *testing.T) {
	proc := NewMessageProcessor()
	proc.DisableExpirationCheck()

	handler := &mockTunnelDataHandler{}
	proc.SetTunnelDataHandler(handler)

	if proc.tunnelDataHandler == nil {
		t.Fatal("expected tunnelDataHandler to be set")
	}
}

// TestProcessTunnelDataDelegatesToHandler verifies that when a TunnelDataHandler
// is configured, incoming TunnelData messages are delegated to it.
func TestProcessTunnelDataDelegatesToHandler(t *testing.T) {
	proc := NewMessageProcessor()
	proc.DisableExpirationCheck()

	handler := &mockTunnelDataHandler{}
	proc.SetTunnelDataHandler(handler)

	// Create a TunnelData message using the constructor
	var data [1024]byte
	msg := NewTunnelDataMessage(12345, data)
	msg.BaseI2NPMessage.SetExpiration(time.Now().Add(5 * time.Minute))

	err := proc.ProcessMessage(msg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !handler.called {
		t.Fatal("expected handler to be called")
	}
}

// Transport decoders return a generic BaseI2NPMessage, including for TunnelData.
// Exercise both wire headers so dispatch cannot depend on a typed constructor.
func TestProcessTunnelDataFromWire(t *testing.T) {
	var data [1024]byte
	for i := range data {
		data[i] = byte(i)
	}
	original := NewTunnelDataMessage(0x12345678, data)
	original.SetMessageID(0x23456789)
	original.SetExpiration(time.Now().Add(time.Minute).Truncate(time.Second))
	for _, short := range []bool{false, true} {
		t.Run(fmt.Sprintf("short=%t", short), func(t *testing.T) {
			var wire []byte
			var err error
			decoded := &BaseI2NPMessage{}
			if short {
				wire, err = original.MarshalShortI2NP()
				if err == nil {
					err = decoded.UnmarshalShortI2NP(wire)
				}
			} else {
				wire, err = original.MarshalBinary()
				if err == nil {
					err = decoded.UnmarshalBinary(wire)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			proc := NewMessageProcessor()
			handler := &mockTunnelDataHandler{}
			proc.SetTunnelDataHandler(handler)
			if err := proc.ProcessMessage(decoded); err != nil {
				t.Fatal(err)
			}
			if !handler.called {
				t.Fatal("wire TunnelData was not delivered to handler")
			}
			carrier, ok := handler.msg.(TunnelCarrier)
			if !ok || carrier.GetTunnelID() != original.GetTunnelID() || !bytes.Equal(carrier.GetTunnelData(), data[:]) {
				t.Fatal("handler did not receive the original tunnel ID and encrypted frame")
			}
			if handler.msg.MessageID() != original.MessageID() || !handler.msg.Expiration().Equal(original.Expiration()) {
				t.Fatal("I2NP metadata changed during TunnelData parsing")
			}
		})
	}
}

func TestProcessTunnelDataRejectsMalformedWirePayload(t *testing.T) {
	for _, size := range []int{0, 4, 1024, 1027, 1029} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			proc := NewMessageProcessor()
			handler := &mockTunnelDataHandler{}
			proc.SetTunnelDataHandler(handler)
			msg := NewBaseI2NPMessage(I2NPMessageTypeTunnelData)
			msg.SetData(make([]byte, size))
			if err := proc.ProcessMessage(msg); err == nil {
				t.Fatal("malformed TunnelData payload was accepted")
			}
			if handler.called {
				t.Fatal("malformed TunnelData was delivered to handler")
			}
		})
	}
}

// TestProcessTunnelDataHandlerError verifies handler errors are propagated.
func TestProcessTunnelDataHandlerError(t *testing.T) {
	proc := NewMessageProcessor()
	proc.DisableExpirationCheck()

	handler := &mockTunnelDataHandler{
		returnErr: errors.New("handler test error"),
	}
	proc.SetTunnelDataHandler(handler)

	var data [1024]byte
	msg := NewTunnelDataMessage(12345, data)
	msg.BaseI2NPMessage.SetExpiration(time.Now().Add(5 * time.Minute))

	err := proc.ProcessMessage(msg)
	if err == nil {
		t.Fatal("expected error from handler")
	}
	if err.Error() != "handler test error" {
		t.Fatalf("expected 'handler test error', got: %v", err)
	}
}

// TestProcessTunnelDataWithoutHandler verifies that without a handler,
// TunnelData messages are validated but not delivered.
func TestProcessTunnelDataWithoutHandler(t *testing.T) {
	proc := NewMessageProcessor()
	proc.DisableExpirationCheck()

	var data [1024]byte
	msg := NewTunnelDataMessage(12345, data)
	msg.BaseI2NPMessage.SetExpiration(time.Now().Add(5 * time.Minute))

	// Should succeed without error (validated but not delivered)
	err := proc.ProcessMessage(msg)
	if err != nil {
		t.Fatalf("expected no error without handler, got: %v", err)
	}
}

// TestProcessTunnelDataInvalidMessage verifies that a raw TunnelData message
// without the required payload is rejected.
func TestProcessTunnelDataInvalidMessage(t *testing.T) {
	proc := NewMessageProcessor()
	proc.DisableExpirationCheck()

	// A transport-decoded message must contain TunnelID plus encrypted data.
	msg := NewBaseI2NPMessage(I2NPMessageTypeTunnelData)
	msg.SetExpiration(time.Now().Add(5 * time.Minute))

	err := proc.processTunnelDataMessage(msg)
	if err == nil {
		t.Fatal("expected error for TunnelData without a payload")
	}
}

// TestTunnelDataHandlerInterface verifies compile-time interface compliance.
func TestTunnelDataHandlerInterface(t *testing.T) {
	var _ TunnelDataHandler = (*mockTunnelDataHandler)(nil)
}
