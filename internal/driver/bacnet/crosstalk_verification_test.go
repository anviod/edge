package bacnet

import (
	"context"
	"fmt"
	"testing"
	"time"

	"edge-gateway/internal/driver/bacnet/btypes"
	"edge-gateway/internal/model"
)

// SmartMockClient implements Client interface for multi-device testing
type SmartMockClient struct {
	Devices map[int]btypes.Device  // InstanceID -> Device Info
	Values  map[string]interface{} // "InstanceID:ObjectType:ObjectInstance" -> Value
}

func (m *SmartMockClient) Close() error    { return nil }
func (m *SmartMockClient) IsRunning() bool { return true }
func (m *SmartMockClient) ClientRun()      {}

func (m *SmartMockClient) WhoIs(wh *WhoIsOpts) ([]btypes.Device, error) {
	var found []btypes.Device
	// Simulate discovery based on Low/High limits
	low := -1
	high := -1
	if wh != nil {
		low = int(wh.Low)
		high = int(wh.High)
	}

	for id, dev := range m.Devices {
		if (low == -1 && high == -1) || (id >= low && id <= high) {
			found = append(found, dev)
		}
	}
	return found, nil
}

func (m *SmartMockClient) WhatIsNetworkNumber() []*btypes.Address         { return nil }
func (m *SmartMockClient) IAm(dest btypes.Address, iam btypes.IAm) error  { return nil }
func (m *SmartMockClient) WhoIsRouterToNetwork() (resp *[]btypes.Address) { return nil }
func (m *SmartMockClient) Objects(dev btypes.Device) (btypes.Device, error) {
	return dev, nil
}

func (m *SmartMockClient) ReadProperty(dest btypes.Device, rp btypes.PropertyData) (btypes.PropertyData, error) {
	// Not used by ReadPoints (which uses ReadMultiProperty), but implemented just in case fallback occurs
	key := fmt.Sprintf("%d:%d:%d", dest.DeviceID, rp.Object.ID.Type, rp.Object.ID.Instance)
	if val, ok := m.Values[key]; ok {
		rp.Object.Properties[0].Data = val
		return rp, nil
	}
	return rp, fmt.Errorf("property not found: %s", key)
}

func (m *SmartMockClient) ReadMultiProperty(dev btypes.Device, rp btypes.MultiplePropertyData) (btypes.MultiplePropertyData, error) {
	// Construct response based on requested objects
	resp := btypes.MultiplePropertyData{
		ErrorClass: 0,
		ErrorCode:  0,
	}

	for _, obj := range rp.Objects {
		respObj := btypes.Object{
			ID: obj.ID,
		}

		// Look up value
		key := fmt.Sprintf("%d:%d:%d", dev.DeviceID, obj.ID.Type, obj.ID.Instance)

		if val, ok := m.Values[key]; ok {
			// Success
			respObj.Properties = []btypes.Property{
				{
					Type:       btypes.PropPresentValue, // Assuming we are reading PresentValue
					ArrayIndex: 0,                       // Not used usually
					Data:       val,
				},
			}
		} else {
			// Error for this object (simulate Property Error)
			// In real BACnet, this might set an error code per property, but for now we just return empty or error
			return resp, fmt.Errorf("mock value not found for key: %s", key)
		}
		resp.Objects = append(resp.Objects, respObj)
	}
	return resp, nil
}

func (m *SmartMockClient) WriteProperty(dest btypes.Device, wp btypes.PropertyData) error {
	return nil
}
func (m *SmartMockClient) WriteMultiProperty(dev btypes.Device, wp btypes.MultiplePropertyData) error {
	return nil
}

// TestCrosstalkVerification verifies that points are read from the correct devices
// and that crosstalk (reading Device A's data for Device B) is prevented.
func TestCrosstalkVerification(t *testing.T) {
	// 1. Setup Mock Data
	mock := &SmartMockClient{
		Devices: map[int]btypes.Device{
			2228316: {DeviceID: 2228316, Ip: "192.168.3.112", Port: 63501, Addr: btypes.Address{Mac: []byte{192, 168, 3, 112, 0xF8, 0x0D}}}, // 63501
			2228317: {DeviceID: 2228317, Ip: "192.168.3.112", Port: 63502, Addr: btypes.Address{Mac: []byte{192, 168, 3, 112, 0xF8, 0x0E}}}, // 63502
			2228318: {DeviceID: 2228318, Ip: "192.168.3.112", Port: 63503, Addr: btypes.Address{Mac: []byte{192, 168, 3, 112, 0xF8, 0x0F}}}, // 63503
			2228319: {DeviceID: 2228319, Ip: "192.168.3.112", Port: 57611, Addr: btypes.Address{Mac: []byte{192, 168, 3, 112, 0xE1, 0x0B}}}, // 57611
		},
		Values: map[string]interface{}{
			// Key format: InstanceID:ObjectType:ObjectInstance
			// ObjectType AnalogValue is 2 (from btypes/object.go but here we use btypes.AnalogValue constant)
			fmt.Sprintf("2228316:%d:1", btypes.AnalogValue): float32(316.0),
			fmt.Sprintf("2228317:%d:1", btypes.AnalogValue): float32(317.0),
			fmt.Sprintf("2228318:%d:1", btypes.AnalogValue): float32(318.0),
			fmt.Sprintf("2228319:%d:1", btypes.AnalogValue): float32(319.0),
		},
	}

	// 2. Initialize Driver
	d := NewBACnetDriver().(*BACnetDriver)
	// Inject Mock Factory
	d.clientFactory = func(cb *ClientBuilder) (Client, error) {
		return mock, nil
	}

	config := model.DriverConfig{
		Config: map[string]any{
			"interface_ip":   "0.0.0.0",
			"interface_port": 47808,
		},
	}
	d.Init(config)
	d.Connect(context.Background())
	defer d.Disconnect()

	// 3. Define Test Cases
	tests := []struct {
		ID         string
		InstanceID int
		Expected   float32
	}{
		{"bacnet-16", 2228316, 316.0},
		{"bacnet-17", 2228317, 317.0},
		{"bacnet-18", 2228318, 318.0},
		{"Room_FC_2014_19", 2228319, 319.0},
	}

	// 4. Run Verification
	for _, tc := range tests {
		t.Run(tc.ID, func(t *testing.T) {
			// 4.1 Set Device Config (Simulate Channel Manager)
			devConfig := map[string]any{
				"instance_id":         tc.InstanceID,
				"ip":                  "192.168.3.112",
				"port":                mock.Devices[tc.InstanceID].Port,
				"_internal_device_id": tc.ID,
			}
			err := d.SetDeviceConfig(devConfig)
			if err != nil {
				t.Fatalf("SetDeviceConfig failed: %v", err)
			}

			// 4.2 Read Points
			points := []model.Point{
				{
					ID:       "Setpoint.1",
					DeviceID: tc.ID,
					Address:  "AnalogValue:1",
					DataType: "float32",
				},
			}

			// We need a short delay or retry because SetDeviceConfig triggers async discovery
			// But since our Mock WhoIs is synchronous and fast, it might happen quickly.
			// However, the driver runs discovery in a goroutine.
			// So we might need to wait for the device context to be ready.
			// The driver's ReadPoints checks context, if missing it returns error.
			// Let's retry a few times.
			var val float32
			var ok bool

			// Wait for context to be created (async discovery)
			for i := 0; i < 10; i++ {
				d.mu.Lock()
				_, exists := d.deviceContexts[tc.InstanceID]
				d.mu.Unlock()
				if exists {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}

			results, err := d.ReadPoints(context.Background(), points)
			if err != nil {
				t.Fatalf("ReadPoints failed: %v", err)
			}

			if result, found := results["Setpoint.1"]; found {
				if v, typeOk := result.Value.(float32); typeOk {
					val = v
					ok = true
				} else {
					t.Errorf("Result type mismatch: expected float32, got %T", result.Value)
				}
			} else {
				t.Errorf("Point Setpoint.1 not found in results")
			}

			if ok {
				if val != tc.Expected {
					t.Errorf("Value Mismatch! Expected: %.2f, Got: %.2f", tc.Expected, val)
				} else {
					t.Logf("✅ Verified %s: Got %.2f", tc.ID, val)
				}
			}
		})
	}

	// 5. Verify Crosstalk Prevention
	t.Run("CrosstalkPrevention", func(t *testing.T) {
		// Try to read points from multiple devices in one call
		// This should fail based on our fix
		mixedPoints := []model.Point{
			{ID: "P1", DeviceID: "bacnet-16", Address: "AnalogValue:1"},
			{ID: "P2", DeviceID: "bacnet-18", Address: "AnalogValue:1"},
		}

		// Set config to one of them (doesn't matter which, as check happens before context lookup)
		d.SetDeviceConfig(map[string]any{"_internal_device_id": "bacnet-16", "instance_id": 2228316})

		_, err := d.ReadPoints(context.Background(), mixedPoints)
		if err == nil {
			t.Error("Expected error when reading mixed device points, but got success")
		} else {
			t.Logf("✅ Verified Crosstalk Prevention: %v", err)
		}
	})
}
