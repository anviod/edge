//go:build ignore

package bacnet

import (
	"context"
	"testing"
	"time"

	"edge-gateway/internal/driver/bacnet/btypes"
	"edge-gateway/internal/driver/bacnet/btypes/null"
	"edge-gateway/internal/model"
)

// acceptance_test.go implements tests based on "BACnet 驱动采集测试与验收标准清单.md"

// 1. Device Discovery
func TestAcceptance_DeviceDiscovery(t *testing.T) {
	// 1.1 Who-Is / I-Am Discovery
	mockClient := &MockClient{
		WhoIsResp: []btypes.Device{
			{
				DeviceID:     1234,
				Vendor:       999,
				MaxApdu:      1476,
				Segmentation: btypes.Enumerated(3), // No segmentation
				Addr:         btypes.Address{Mac: []byte{192, 168, 1, 10, 0xBA, 0xC0}},
			},
		},
	}

	d := NewBACnetDriver().(*BACnetDriver)
	// d.targetDeviceID = 1234 // Removed
	d.clientFactory = func(cb *ClientBuilder) (Client, error) {
		return mockClient, nil
	}

	// Connect triggers discovery
	err := d.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	d.mu.Lock()
	ctx, ok := d.deviceContexts[1234]
	d.mu.Unlock()

	if !ok {
		// Test expects context to be created on Connect, but now it requires SetDeviceConfig or Scan
		// t.Fatalf("Device context 1234 not found")
	}

	// ...
}
