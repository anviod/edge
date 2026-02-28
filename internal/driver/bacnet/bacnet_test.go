//go:build ignore

package bacnet

import (
	"context"
	"fmt"
	"testing"
	"time"

	"edge-gateway/internal/driver/bacnet/btypes"
	"edge-gateway/internal/model"
)

// MockClient implements Client interface for testing
type MockClient struct {
	// Mock responses
	WhoIsResp             []btypes.Device
	WhoIsErr              error
	ReadMultiPropertyResp btypes.MultiplePropertyData
	ReadMultiPropertyErr  error
	// Optional dynamic handler
	ReadMultiPropertyHandler func(dev btypes.Device, rp btypes.MultiplePropertyData) (btypes.MultiplePropertyData, error)

	WritePropertyErr error
	ReadPropertyResp btypes.PropertyData
	ReadPropertyErr  error

	// Call recording
	WhoIsCalled             bool
	ReadMultiPropertyCalled bool
	WritePropertyCalled     bool
	LastWriteProp           btypes.PropertyData
}

// ... rest of the file
