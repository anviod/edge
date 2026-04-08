package config

import (
	"edge-gateway/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestPointsFullPersistence(t *testing.T) {
	// Create a temporary directory for test configs
	tempDir, err := os.MkdirTemp("", "test-points-full-persistence")
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
  - id: point3
    name: Humidity
    address: "2"
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

	// Step 1: Initial load
	cfg1, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	device1 := cfg1.Channels[0].Devices[0]
	if len(device1.Points) != 3 {
		t.Errorf("Expected 3 points, got %d", len(device1.Points))
	}

	// Step 2: Modify a point
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
    address: "10"
    datatype: int32
  - id: point3
    name: Humidity
    address: "2"
    datatype: float32
`
	if err := os.WriteFile(devicePath, []byte(updatedDeviceContent), 0644); err != nil {
		t.Fatalf("Failed to update test-device.yaml: %v", err)
	}

	// Step 3: Load after modification
	cfg2, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config after modification: %v", err)
	}

	device2 := cfg2.Channels[0].Devices[0]
	if len(device2.Points) != 3 {
		t.Errorf("Expected 3 points after modification, got %d", len(device2.Points))
	}

	// Verify the modified point
	modifiedPoint := device2.Points[1]
	if modifiedPoint.ID != "point2" {
		t.Errorf("Expected modified point ID 'point2', got '%s'", modifiedPoint.ID)
	}
	if modifiedPoint.Name != "Pressure" {
		t.Errorf("Expected modified point Name 'Pressure', got '%s'", modifiedPoint.Name)
	}
	if modifiedPoint.Address != "10" {
		t.Errorf("Expected modified point Address '10', got '%s'", modifiedPoint.Address)
	}
	if modifiedPoint.DataType != "int32" {
		t.Errorf("Expected modified point DataType 'int32', got '%s'", modifiedPoint.DataType)
	}

	// Step 4: Delete a point
	deletedDeviceContent := `
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
  - id: point3
    name: Humidity
    address: "2"
    datatype: float32
`
	if err := os.WriteFile(devicePath, []byte(deletedDeviceContent), 0644); err != nil {
		t.Fatalf("Failed to update test-device.yaml: %v", err)
	}

	// Step 5: Load after deletion
	cfg3, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config after deletion: %v", err)
	}

	device3 := cfg3.Channels[0].Devices[0]
	if len(device3.Points) != 2 {
		t.Errorf("Expected 2 points after deletion, got %d", len(device3.Points))
	}

	// Verify the remaining points
	expectedPoints := []model.Point{
		{
			ID:       "point1",
			Name:     "Temperature",
			Address:  "0",
			DataType: "float32",
		},
		{
			ID:       "point3",
			Name:     "Humidity",
			Address:  "2",
			DataType: "float32",
		},
	}

	for i, expected := range expectedPoints {
		actual := device3.Points[i]
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

	// Step 6: Load again (simulate restart)
	cfg4, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load config after restart: %v", err)
	}

	device4 := cfg4.Channels[0].Devices[0]
	if len(device4.Points) != 2 {
		t.Errorf("Expected 2 points after restart, got %d", len(device4.Points))
	}

	// Verify consistency after restart
	for i, point := range device4.Points {
		expected := device3.Points[i]
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

	t.Log("Full points persistence test passed: all points operations are consistent after restart")
}
