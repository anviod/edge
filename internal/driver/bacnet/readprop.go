package bacnet

import (
	"context"
	"fmt"
	//"log"
	"time"

	"edge-gateway/internal/driver/bacnet/btypes"
	"edge-gateway/internal/driver/bacnet/encoding"
)

// ReadProperty reads a single property from a single object in the given device.
func (c *client) ReadProperty(device btypes.Device, rp btypes.PropertyData) (btypes.PropertyData, error) {
	return c.ReadPropertyWithTimeout(device, rp, 10*time.Second)
}

func (c *client) ReadPropertyWithTimeout(device btypes.Device, rp btypes.PropertyData, timeout time.Duration) (btypes.PropertyData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	id, err := c.tsm.ID(ctx)
	if err != nil {
		return btypes.PropertyData{}, fmt.Errorf("unable to get an transaction id: %v", err)
	}
	defer c.tsm.Put(id)
	enc := encoding.NewEncoder()
	device.Addr.SetLength()
	npdu := &btypes.NPDU{
		Version:               btypes.ProtocolVersion,
		Destination:           &device.Addr,
		Source:                c.dataLink.GetMyAddress(),
		IsNetworkLayerMessage: false,
		ExpectingReply:        true,
		Priority:              btypes.Normal,
		HopCount:              btypes.DefaultHopCount,
	}
	enc.NPDU(npdu)

	err = enc.ReadProperty(uint8(id), rp)
	if enc.Error() != nil || err != nil {
		return btypes.PropertyData{}, err
	}

	// the value filled doesn't matter. it just needs to be non nil
	err = fmt.Errorf("go")
	for count := 0; err != nil && count < retryCount; count++ {
		var b []byte
		var out btypes.PropertyData
		_, err = c.Send(device.Addr, npdu, enc.Bytes(), nil)
		if err != nil {
			//log.Print(err)
			continue
		}

		var raw interface{}
		raw, err = c.tsm.Receive(id, timeout)
		if err != nil {
			continue
		}
		switch v := raw.(type) {
		case error:
			return out, v
		case []byte:
			b = v
		default:
			return out, fmt.Errorf("received unknown datatype %T", raw)
		}

		dec := encoding.NewDecoder(b)

		var apdu btypes.APDU
		if err = dec.APDU(&apdu); err != nil {
			continue
		}
		if apdu.Error.Class != 0 || apdu.Error.Code != 0 {
			err = fmt.Errorf("received error, class: %d, code: %d", apdu.Error.Class, apdu.Error.Code)
			continue
		}

		if err = dec.ReadProperty(&out); err != nil {
			continue
		}
		return out, dec.Error()
	}
	return btypes.PropertyData{}, err
}
