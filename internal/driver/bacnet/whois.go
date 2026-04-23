package bacnet

import (
	"edge-gateway/internal/driver/bacnet/btypes"
	"edge-gateway/internal/driver/bacnet/encoding"
	//"log"
	"time"
)

type WhoIsOpts struct {
	Low             int             `json:"low"`
	High            int             `json:"high"`
	GlobalBroadcast bool            `json:"global_broadcast"`
	NetworkNumber   uint16          `json:"network_number"`
	Destination     *btypes.Address `json:"-"`
}

// WhoIs finds all devices with ids between the provided low and high values.
// Use constant ArrayAll for both fields to scan the entire network at once.
// Using ArrayAll is highly discouraged for most networks since it can lead
// to a high congested network.
func (c *client) WhoIs(wh *WhoIsOpts) ([]btypes.Device, error) {
	dest := *c.dataLink.GetBroadcastAddress()
	if wh.Destination != nil {
		dest = *wh.Destination
	}
	// Log the destination
	if d, err := dest.UDPAddr(); err == nil {
		_ = d
		//log.Printf("[DEBUG] WhoIs broadcasting to: %s (Low: %d, High: %d)\n", d.String(), wh.Low, wh.High)
	}

	enc := encoding.NewEncoder()
	low := wh.Low
	high := wh.High
	if wh.GlobalBroadcast {
		wh.NetworkNumber = btypes.GlobalBroadcast //65535
	}
	if low <= 0 {
		low = 0
	}
	if high <= 0 {
		high = 4194304 //max dev id
	}

	dest.Net = wh.NetworkNumber
	npdu := &btypes.NPDU{
		Version:               btypes.ProtocolVersion,
		Destination:           &dest,
		Source:                c.dataLink.GetMyAddress(),
		IsNetworkLayerMessage: false,
		// We are not expecting a direct reply from a single destination
		ExpectingReply: false,
		Priority:       btypes.Normal,
		HopCount:       btypes.DefaultHopCount,
	}
	enc.NPDU(npdu)
	err := enc.WhoIs(int32(low), int32(high))
	if err != nil {
		return nil, err
	}
	// Subscribe to any changes in the range. If it is a broadcast,
	var start, end int
	if low == -1 || high == -1 {
		start = 0
		end = maxInt
	} else {
		start = low
		end = high
	}
	// Run in parallel
	errChan := make(chan error)
	go func() {
		time.Sleep(50 * time.Millisecond)
		_, err = c.Send(dest, npdu, enc.Bytes(), nil)
		errChan <- err
	}()
	values, err := c.utsm.Subscribe(start, end)
	if err != nil {
		return nil, err
	}
	err = <-errChan
	if err != nil {
		return nil, err
	}
	// Weed out values that are not important such as non object type
	// and that are not
	uniqueMap := make(map[btypes.ObjectInstance]btypes.Device)
	uniqueList := make([]btypes.Device, len(uniqueMap))

	for _, v := range values {

		r, ok := v.(btypes.IAm)
		// Skip non I AM responses
		if !ok {
			continue
		}
		// Check to see if we are in the map before inserting
		if _, ok := uniqueMap[r.ID.Instance]; !ok {
			dev := btypes.Device{
				DeviceID:     int(r.ID.Instance),
				Addr:         r.Addr,
				ID:           r.ID,
				MaxApdu:      r.MaxApdu,
				Segmentation: r.Segmentation,
				Vendor:       r.Vendor,
			}
			ip, err := r.Addr.UDPAddr()
			if err == nil {
				dev.Ip = ip.IP.String()
				dev.Port = ip.Port
			}

			uniqueMap[r.ID.Instance] = btypes.Device(dev)
			uniqueList = append(uniqueList, dev)
		}
	}
	return uniqueList, err
}
