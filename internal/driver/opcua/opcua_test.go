package opcua

import (
	"context"
	"testing"

	"edge-gateway/internal/driver"
	"edge-gateway/internal/model"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/assert"
)

// MockClient for testing browseNode logic without real connection
// Since opcua.Client is a struct, we cannot easily mock it unless we wrap it.
// However, OpcUaDriver uses *opcua.Client directly.
// To test browseNode, we might need to set up a real mock server or refactor OpcUaDriver to use an interface.
// For now, let's test the helper functions logic like lookupDataType and castValue thoroughly,
// and if possible, integration test with a mock server.

// TestLookupDataType tests the DataType lookup logic
func TestLookupDataType(t *testing.T) {
	tests := []struct {
		id   int
		want string
	}{
		{1, "Boolean"},
		{2, "SByte"},
		{3, "Byte"},
		{4, "Int16"},
		{5, "UInt16"},
		{6, "Int32"},
		{7, "UInt32"},
		{8, "Int64"},
		{9, "UInt64"},
		{10, "Float"},
		{11, "Double"},
		{12, "String"},
		{13, "DateTime"},
		{999, "ns=0;i=999"},
	}

	for _, tt := range tests {
		nodeID := ua.NewNumericNodeID(0, uint32(tt.id))
		got := lookupDataType(nodeID)
		if got != tt.want {
			t.Errorf("lookupDataType(%d) = %s, want %s", tt.id, got, tt.want)
		}
	}

	// Test Namespace != 0
	nodeID := ua.NewNumericNodeID(2, 1234)
	if got := lookupDataType(nodeID); got != "ns=2;i=1234" {
		t.Errorf("lookupDataType(ns=2;i=1234) = %s, want ns=2;i=1234", got)
	}
}

func TestCastValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		dataType string
		want     any
		wantErr  bool
	}{
		// Integer conversions
		{"Float64 to Int16", float64(123), "int16", int16(123), false},
		{"Float64 to UInt16", float64(123), "uint16", uint16(123), false},
		{"Float64 to Int32", float64(123), "int32", int32(123), false},
		{"String to Int16", "123", "int16", int16(123), false},

		// Byte/SByte conversions
		{"Float64 to Byte", float64(255), "byte", uint8(255), false},
		{"Float64 to SByte", float64(127), "sbyte", int8(127), false},
		{"String to Byte", "255", "byte", uint8(255), false},
		{"String to SByte", "-128", "sbyte", int8(-128), false},

		// Float conversions
		{"Float64 to Float32", float64(123.45), "float32", float32(123.45), false},
		{"String to Float32", "123.45", "float32", float32(123.45), false},

		// Boolean conversions
		{"Bool to Bool", true, "bool", true, false},
		{"String to Bool", "true", "bool", true, false},
		{"Int to Bool (1)", 1, "bool", true, false},
		{"Int to Bool (0)", 0, "bool", false, false},

		// Errors
		{"Invalid String to Int", "abc", "int16", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := castValue(tt.input, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("castValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("castValue() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

// TestDriverMethodsCoverage covers basic driver lifecycle methods
func TestDriverMethodsCoverage(t *testing.T) {
	d := NewOpcUaDriver()
	err := d.Init(model.DriverConfig{})
	assert.NoError(t, err)

	err = d.Connect(context.Background())
	assert.NoError(t, err)

	status := d.Health()
	// Should be unknown as no client connected
	assert.Equal(t, driver.HealthStatusUnknown, status)

	err = d.Disconnect()
	assert.NoError(t, err)
}

// TestSetDeviceConfigCoverage covers configuration parsing
func TestSetDeviceConfigCoverage(t *testing.T) {
	d := NewOpcUaDriver().(*OpcUaDriver)

	// Test use_dataformat_decoder
	config := map[string]any{
		"endpoint":               "opc.tcp://localhost:4840", // Required field
		"use_dataformat_decoder": true,
	}
	err := d.SetDeviceConfig(config)
	assert.NoError(t, err)
	assert.True(t, d.useDataformatDecoder)

	config["use_dataformat_decoder"] = "false"
	err = d.SetDeviceConfig(config)
	assert.NoError(t, err)
	assert.False(t, d.useDataformatDecoder)
}
