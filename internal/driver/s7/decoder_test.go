package s7

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAddress_DB(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		name     string
		addr     string
		expected *S7Area
		wantErr  bool
	}{
		{
			name: "DB double word",
			addr: "DB1.DBD0",
			expected: &S7Area{
				Area:     S7AreaDB,
				DBNumber: 1,
				ByteOff:  0,
				WordLen:  S7WLDWord,
			},
		},
		{
			name: "DB word",
			addr: "DB10.DBW20",
			expected: &S7Area{
				Area:     S7AreaDB,
				DBNumber: 10,
				ByteOff:  20,
				WordLen:  S7WLWord,
			},
		},
		{
			name: "DB byte",
			addr: "DB100.DBB4",
			expected: &S7Area{
				Area:     S7AreaDB,
				DBNumber: 100,
				ByteOff:  4,
				WordLen:  S7WLByte,
			},
		},
		{
			name: "DB bit",
			addr: "DB1.DBX0.7",
			expected: &S7Area{
				Area:     S7AreaDB,
				DBNumber: 1,
				ByteOff:  0,
				BitOff:   7,
				WordLen:  S7WLBit,
				IsBit:    true,
			},
		},
		{
			name: "DB bit at offset 7006",
			addr: "DB1.DBX7006.7",
			expected: &S7Area{
				Area:     S7AreaDB,
				DBNumber: 1,
				ByteOff:  7006,
				BitOff:   7,
				WordLen:  S7WLBit,
				IsBit:    true,
			},
		},
		{
			name: "DB double word at offset 7500",
			addr: "DB1.DBD7500",
			expected: &S7Area{
				Area:     S7AreaDB,
				DBNumber: 1,
				ByteOff:  7500,
				WordLen:  S7WLDWord,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			area, err := decoder.ParseAddress(tt.addr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Area, area.Area)
			assert.Equal(t, tt.expected.DBNumber, area.DBNumber)
			assert.Equal(t, tt.expected.ByteOff, area.ByteOff)
			assert.Equal(t, tt.expected.BitOff, area.BitOff)
			assert.Equal(t, tt.expected.WordLen, area.WordLen)
			assert.Equal(t, tt.expected.IsBit, area.IsBit)
		})
	}
}

func TestParseAddress_Merker(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		name     string
		addr     string
		expected *S7Area
	}{
		{
			name: "M bit",
			addr: "M0.0",
			expected: &S7Area{
				Area:    S7AreaMK,
				ByteOff: 0,
				BitOff:  0,
				WordLen: S7WLBit,
				IsBit:   true,
			},
		},
		{
			name: "M bit 3.5",
			addr: "M3.5",
			expected: &S7Area{
				Area:    S7AreaMK,
				ByteOff: 3,
				BitOff:  5,
				WordLen: S7WLBit,
				IsBit:   true,
			},
		},
		{
			name: "M double word",
			addr: "MD0",
			expected: &S7Area{
				Area:    S7AreaMK,
				ByteOff: 0,
				WordLen: S7WLDWord,
			},
		},
		{
			name: "M word",
			addr: "MW10",
			expected: &S7Area{
				Area:    S7AreaMK,
				ByteOff: 10,
				WordLen: S7WLWord,
			},
		},
		{
			name: "M byte",
			addr: "MB20",
			expected: &S7Area{
				Area:    S7AreaMK,
				ByteOff: 20,
				WordLen: S7WLByte,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			area, err := decoder.ParseAddress(tt.addr)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Area, area.Area)
			assert.Equal(t, tt.expected.ByteOff, area.ByteOff)
			assert.Equal(t, tt.expected.BitOff, area.BitOff)
			assert.Equal(t, tt.expected.WordLen, area.WordLen)
			assert.Equal(t, tt.expected.IsBit, area.IsBit)
		})
	}
}

func TestParseAddress_IO(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		name     string
		addr     string
		expected *S7Area
	}{
		{
			name: "Input bit",
			addr: "I0.0",
			expected: &S7Area{
				Area:    S7AreaPE,
				ByteOff: 0,
				BitOff:  0,
				WordLen: S7WLBit,
				IsBit:   true,
			},
		},
		{
			name: "Input bit 1.3",
			addr: "I1.3",
			expected: &S7Area{
				Area:    S7AreaPE,
				ByteOff: 1,
				BitOff:  3,
				WordLen: S7WLBit,
				IsBit:   true,
			},
		},
		{
			name: "Output bit",
			addr: "Q0.0",
			expected: &S7Area{
				Area:    S7AreaPA,
				ByteOff: 0,
				BitOff:  0,
				WordLen: S7WLBit,
				IsBit:   true,
			},
		},
		{
			name: "Output double word",
			addr: "QD10",
			expected: &S7Area{
				Area:    S7AreaPA,
				ByteOff: 10,
				WordLen: S7WLDWord,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			area, err := decoder.ParseAddress(tt.addr)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Area, area.Area)
			assert.Equal(t, tt.expected.ByteOff, area.ByteOff)
			assert.Equal(t, tt.expected.BitOff, area.BitOff)
			assert.Equal(t, tt.expected.WordLen, area.WordLen)
			assert.Equal(t, tt.expected.IsBit, area.IsBit)
		})
	}
}

func TestParseAddress_Timer_Counter(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		name     string
		addr     string
		expected *S7Area
	}{
		{
			name: "Timer",
			addr: "T0",
			expected: &S7Area{
				Area:    S7AreaTM,
				ByteOff: 0,
				WordLen: S7WLTimer,
			},
		},
		{
			name: "Counter",
			addr: "C5",
			expected: &S7Area{
				Area:    S7AreaCT,
				ByteOff: 5,
				WordLen: S7WLCounter,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			area, err := decoder.ParseAddress(tt.addr)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Area, area.Area)
			assert.Equal(t, tt.expected.ByteOff, area.ByteOff)
			assert.Equal(t, tt.expected.WordLen, area.WordLen)
		})
	}
}

func TestParseAddress_CaseInsensitive(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		addr string
	}{
		{"db1.dbd0"},
		{"DB1.DBD0"},
		{"Db1.Dbd0"},
		{"m0.0"},
		{"M0.0"},
		{"i0.0"},
		{"I0.0"},
		{"q0.0"},
		{"Q0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			_, err := decoder.ParseAddress(tt.addr)
			assert.NoError(t, err)
		})
	}
}

func TestParseAddress_Invalid(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		name string
		addr string
	}{
		{"empty", ""},
		{"random", "abc"},
		{"invalid db", "DB1.XXX0"},
		{"missing offset", "DB1."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decoder.ParseAddress(tt.addr)
			assert.Error(t, err)
		})
	}
}

func TestDecodeValue_Bool(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		name     string
		buffer   []byte
		area     *S7Area
		dataType string
		expected interface{}
	}{
		{
			name:     "bool from bit 0",
			buffer:   []byte{0x01},
			area:     &S7Area{IsBit: true, BitOff: 0},
			dataType: "bool",
			expected: true,
		},
		{
			name:     "bool from bit 0 false",
			buffer:   []byte{0x00},
			area:     &S7Area{IsBit: true, BitOff: 0},
			dataType: "bool",
			expected: false,
		},
		{
			name:     "bool from bit 3",
			buffer:   []byte{0x08}, // bit 3 set
			area:     &S7Area{IsBit: true, BitOff: 3},
			dataType: "bool",
			expected: true,
		},
		{
			name:     "bool from bit 7",
			buffer:   []byte{0x80}, // bit 7 set
			area:     &S7Area{IsBit: true, BitOff: 7},
			dataType: "bool",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := decoder.DecodeValue(tt.buffer, tt.area, tt.dataType)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func TestDecodeValue_Int16(t *testing.T) {
	decoder := NewS7Decoder()

	// S7 uses big-endian (network byte order)
	// 100 = 0x0064
	buffer := []byte{0x00, 0x64}
	area := &S7Area{WordLen: S7WLWord}

	val, err := decoder.DecodeValue(buffer, area, "int16")
	require.NoError(t, err)
	assert.Equal(t, int16(100), val)
}

func TestDecodeValue_Uint16(t *testing.T) {
	decoder := NewS7Decoder()

	// 1000 = 0x03E8
	buffer := []byte{0x03, 0xE8}
	area := &S7Area{WordLen: S7WLWord}

	val, err := decoder.DecodeValue(buffer, area, "uint16")
	require.NoError(t, err)
	assert.Equal(t, uint16(1000), val)
}

func TestDecodeValue_Float32(t *testing.T) {
	decoder := NewS7Decoder()

	// IEEE 754 for 3.14: 0x4048F5C3
	buffer := []byte{0x40, 0x48, 0xF5, 0xC3}
	area := &S7Area{WordLen: S7WLReal}

	val, err := decoder.DecodeValue(buffer, area, "float32")
	require.NoError(t, err)
	assert.InDelta(t, float32(3.14), val, 0.001)
}

func TestDecodeValue_Float64(t *testing.T) {
	decoder := NewS7Decoder()

	// IEEE 754 for 3.14 as double: 0x40091EB851EB851F
	buffer := []byte{0x40, 0x09, 0x1E, 0xB8, 0x51, 0xEB, 0x85, 0x1F}
	area := &S7Area{WordLen: S7WLDWord}

	val, err := decoder.DecodeValue(buffer, area, "float64")
	require.NoError(t, err)
	assert.InDelta(t, 3.14, val, 0.001)
}

func TestEncodeDecode_RoundTrip(t *testing.T) {
	decoder := NewS7Decoder()

	t.Run("int16 round trip", func(t *testing.T) {
		buffer := make([]byte, 2)
		area := &S7Area{WordLen: S7WLWord}

		err := decoder.EncodeValue(buffer, area, "int16", int16(12345))
		require.NoError(t, err)

		val, err := decoder.DecodeValue(buffer, area, "int16")
		require.NoError(t, err)
		assert.Equal(t, int16(12345), val)
	})

	t.Run("float32 round trip", func(t *testing.T) {
		buffer := make([]byte, 4)
		area := &S7Area{WordLen: S7WLReal}

		err := decoder.EncodeValue(buffer, area, "float32", float32(3.14))
		require.NoError(t, err)

		val, err := decoder.DecodeValue(buffer, area, "float32")
		require.NoError(t, err)
		assert.InDelta(t, float32(3.14), val, 0.001)
	})

	t.Run("bool bit round trip", func(t *testing.T) {
		buffer := []byte{0x00}
		area := &S7Area{IsBit: true, BitOff: 3}

		err := decoder.EncodeValue(buffer, area, "bool", true)
		require.NoError(t, err)

		val, err := decoder.DecodeValue(buffer, area, "bool")
		require.NoError(t, err)
		assert.Equal(t, true, val)
	})
}

func TestReadSizeForArea(t *testing.T) {
	decoder := NewS7Decoder()

	tests := []struct {
		area     *S7Area
		expected int
	}{
		{&S7Area{WordLen: S7WLBit}, 1},
		{&S7Area{WordLen: S7WLByte}, 1},
		{&S7Area{WordLen: S7WLWord}, 2},
		{&S7Area{WordLen: S7WLDWord}, 4},
		{&S7Area{WordLen: S7WLReal}, 4},
		{&S7Area{WordLen: S7WLCounter}, 2},
		{&S7Area{WordLen: S7WLTimer}, 2},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			size := decoder.ReadSizeForArea(tt.area)
			assert.Equal(t, tt.expected, size)
		})
	}
}
