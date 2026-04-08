package config

import (
	"edge-gateway/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestPointsPersistence(t *testing.T) {
	// Create a temporary directory for test configs
	tempDir, err := os.MkdirTemp("", "test-points-persistence")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create devices directory
	devicesDir := filepath.Join(tempDir, "devices")
	if err := os.Mkdir(devicesDir, 0755); err != nil {
		t.Fatalf("Failed to create devices dir: %v", err)
	}

	// Create initial device file with points
	deviceContent := `
id: test-device
name: Test Device
enable: true
interval: 10s
config:
  slave_id: 1
points:
  - id: point1
    name: Temperature
    address: "0"
    datatype: float32
  - id: point2
    name: Pressure
    address: "1"
    datatype: float32
`
	devicePath := filepath.Join(devicesDir, "test-device.yaml")
	if err := os.WriteFile(devicePath, []byte(deviceContent), 0644); err != nil {
		t.Fatalf("Failed to write test-device.yaml: %v", err)
	}

	// Create channels.yaml
	channelsContent := `
- id: test-channel
  name: Test Channel
  protocol: modbus-tcp
  enable: true
  config:
    url: tcp://127.0.0.1:502
  devices:
    - id: test-device
      device_file: "devices/test-device.yaml"
`
	channelsPath := filepath.Join(tempDir, "channels.yaml")
	if err := os.WriteFile(channelsPath, []byte(channelsContent), 0644); err != nil {
		t.Fatalf("Failed to write channels.yaml: %v", err)
	}

	// Create other required config files
	createEmptyFile(t, tempDir, "server.yaml")
	createEmptyFile(t, tempDir, "storage.yaml")
	createEmptyFile(t, tempDir, "northbound.yaml")
	createEmptyFile(t, tempDir, "edge_rules.yaml")
	createEmptyFile(t, tempDir, "system.yaml")
	createEmptyFile(t, tempDir, "users.yaml")

	// First load: verify initial points
	cfg1, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg1.Channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(cfg1.Channels))
	}

	device1 := cfg1.Channels[0].Devices[0]
	if len(device1.Points) != 2 {
		t.Errorf("Expected 2 points, got %d", len(device1.Points))
	}

	// Verify initial points
	expectedPoints := []model.Point{
		{
			ID:       "point1",
			Name:     "Temperature",
			Address:  "0",
			DataType: "float32",
		},
		{
			ID:       "point2",
			Name:     "Pressure",
			Address:  "1",
			DataType: "float32",
		},
	}

	for i, expected := range expectedPoints {
		actual := device1.Points[i]
		if actual.ID != expected.ID {
			t.Errorf("Expected point %d ID '%s', got '%s'", i, expected.ID, actual.ID)
		}
		if actual.Name != expected.Name {
			t.Errorf("Expected point %d Name '%s', got '%s'", i, expected.Name, actual.Name)
		}
		if actual.Address != expected.Address {
			t.Errorf("Expected point %d Address '%s', got '%s'", i, expected.Address, actual.Address)
		}
		if actual.DataType != expected.DataType {
			t.Errorf("Expected point %d DataType '%s', got '%s'", i, expected.DataType, actual.DataType)
		}
	}

	// Modify device file: add a new point
	updatedDeviceContent := `
id: test-device
name: Test Device
enable: true
interval: 10s
config:
  slave_id: 1
points:
  - id: point1
    name: Temperature
    address: "0"
    datatype: float32
  - id: point2
    name: Pressure
    address: "1"
    datatype: float32
  - id: point3
    name: Humidity
    address: "2"
    datatype: float32
`
	if err := os.WriteFile(devicePath, []byte(updatedDeviceContent), 0644); err != nil {
		t.Fatalf("Failed to update test-device.yaml: %v", err)
	}

	// Second load: verify updated points
	cfg2, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config after update: %v", err)
	}

	device2 := cfg2.Channels[0].Devices[0]
	if len(device2.Points) != 3 {
		t.Errorf("Expected 3 points after update, got %d", len(device2.Points))
	}

	// Verify the new point was added
	newPoint := device2.Points[2]
	if newPoint.ID != "point3" {
		t.Errorf("Expected new point ID 'point3', got '%s'", newPoint.ID)
	}
	if newPoint.Name != "Humidity" {
		t.Errorf("Expected new point Name 'Humidity', got '%s'", newPoint.Name)
	}
	if newPoint.Address != "2" {
		t.Errorf("Expected new point Address '2', got '%s'", newPoint.Address)
	}
	if newPoint.DataType != "float32" {
		t.Errorf("Expected new point DataType 'float32', got '%s'", newPoint.DataType)
	}

	// Third load: verify consistency after "restart"
	cfg3, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config after restart: %v", err)
	}

	device3 := cfg3.Channels[0].Devices[0]
	if len(device3.Points) != 3 {
		t.Errorf("Expected 3 points after restart, got %d", len(device3.Points))
	}

	// Verify all points are consistent
	for i, point := range device3.Points {
		expected := device2.Points[i]
		if point.ID != expected.ID {
			t.Errorf("Expected point %d ID '%s' after restart, got '%s'", i, expected.ID, point.ID)
		}
		if point.Name != expected.Name {
			t.Errorf("Expected point %d Name '%s' after restart, got '%s'", i, expected.Name, point.Name)
		}
		if point.Address != expected.Address {
			t.Errorf("Expected point %d Address '%s' after restart, got '%s'", i, expected.Address, point.Address)
		}
		if point.DataType != expected.DataType {
			t.Errorf("Expected point %d DataType '%s' after restart, got '%s'", i, expected.DataType, point.DataType)
		}
	}

	t.Log("Points persistence test passed: all points are consistent after restart")
}
