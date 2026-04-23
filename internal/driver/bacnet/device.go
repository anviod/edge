package bacnet

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"edge-gateway/internal/driver/bacnet/btypes"
	"edge-gateway/internal/driver/bacnet/btypes/ndpu"
	"edge-gateway/internal/driver/bacnet/datalink"
	"edge-gateway/internal/driver/bacnet/encoding"
	"edge-gateway/internal/driver/bacnet/helpers/validation"
	"edge-gateway/internal/driver/bacnet/tsm"
	"edge-gateway/internal/driver/bacnet/utsm"
)

const mtuHeaderLength = 4
const defaultStateSize = 64 // Increased from 20 to support more concurrent devices (32+)
const forwardHeaderLength = 10

type Client interface {
	io.Closer
	IsRunning() bool
	ClientRun()
	WhoIs(wh *WhoIsOpts) ([]btypes.Device, error)
	WhatIsNetworkNumber() []*btypes.Address
	IAm(dest btypes.Address, iam btypes.IAm) error
	WhoIsRouterToNetwork() (resp *[]btypes.Address)
	Objects(dev btypes.Device) (btypes.Device, error)
	ReadProperty(dest btypes.Device, rp btypes.PropertyData) (btypes.PropertyData, error)
	ReadMultiProperty(dev btypes.Device, rp btypes.MultiplePropertyData) (btypes.MultiplePropertyData, error)
	ReadPropertyWithTimeout(dest btypes.Device, rp btypes.PropertyData, timeout time.Duration) (btypes.PropertyData, error)
	ReadMultiPropertyWithTimeout(dev btypes.Device, rp btypes.MultiplePropertyData, timeout time.Duration) (btypes.MultiplePropertyData, error)
	WriteProperty(dest btypes.Device, wp btypes.PropertyData) error
	WriteMultiProperty(dev btypes.Device, wp btypes.MultiplePropertyData) error
}

type client struct {
	dataLink       datalink.DataLink
	tsm            *tsm.TSM
	utsm           *utsm.Manager
	readBufferPool sync.Pool
	running        bool
}

type ClientBuilder struct {
	DataLink   datalink.DataLink
	Interface  string
	Ip         string
	Port       int
	SubnetCIDR int
	MaxPDU     uint16
}

// NewClient creates a new client with the given interface and
func NewClient(cb *ClientBuilder) (Client, error) {
	var err error
	var dataLink datalink.DataLink
	iface := cb.Interface
	ip := cb.Ip
	port := cb.Port
	maxPDU := cb.MaxPDU
	//check ip
	ok := validation.ValidIP(ip)
	if !ok {

	}
	//check port
	if port == 0 {
		port = datalink.DefaultPort
	}
	ok = validation.ValidPort(port)
	if !ok {

	}
	//check adpu
	if maxPDU == 0 {
		maxPDU = btypes.MaxAPDU
	}
	//build datalink
	if cb.DataLink != nil {
		dataLink = cb.DataLink
	} else if iface != "" {
		dataLink, err = datalink.NewUDPDataLink(iface, port)
		if err != nil {
			return nil, err
		}
	} else {
		//check subnet
		sub := cb.SubnetCIDR
		ok = validation.ValidCIDR(ip, sub)
		if !ok {

		}
		dataLink, err = datalink.NewUDPDataLinkFromIP(ip, sub, port)
		if err != nil {
			return nil, err
		}
	}

	cli := &client{
		dataLink: dataLink,
		tsm:      tsm.New(defaultStateSize),
		utsm: utsm.NewManager(
			utsm.DefaultSubscriberTimeout(time.Second*time.Duration(10)),
			utsm.DefaultSubscriberLastReceivedTimeout(time.Second*time.Duration(2)),
		),
		readBufferPool: sync.Pool{New: func() interface{} {
			return make([]byte, maxPDU)
		}},
	}
	return cli, err
}

func (c *client) ClientRun() {
	var err error = nil
	c.running = true
	for err == nil {
		b := c.readBufferPool.Get().([]byte)
		var addr *btypes.Address
		var n int
		addr, n, err = c.dataLink.Receive(b)
		if err != nil {
			continue
		}
		go c.handleMsg(addr, b[:n])
	}
	c.running = false
}

func (c *client) handleMsg(src *btypes.Address, b []byte) {
//	log.Printf("[DEBUG] handleMsg: len=%d src=%v", len(b), src)
	var header btypes.BVLC
	var npdu btypes.NPDU
	var apdu btypes.APDU
	dec := encoding.NewDecoder(b)
	err := dec.BVLC(&header)
	if err != nil {
		log.Printf("[ERROR] bacnet decode error: %v", err)
		return
	}
	//log.Printf("[DEBUG] handleMsg: BVLC Function=%x", header.Function)

	if header.Function == btypes.BacFuncBroadcast || header.Function == btypes.BacFuncUnicast || header.Function == btypes.BacFuncForwardedNPDU {
		// Remove the header information
		b = b[mtuHeaderLength:]
		networkList, err := dec.NPDU(&npdu)
		if err != nil {
			log.Printf("[ERROR] NPDU decode error: %v", err)
			return
		}

		if npdu.IsNetworkLayerMessage {
			log.Printf("[DEBUG] Ignored Network Layer Message: %v", npdu.NetworkLayerMessageType)
			if npdu.NetworkLayerMessageType == ndpu.NetworkIs {
				c.utsm.Publish(int(npdu.Source.Net), npdu)
				//return
			}
			if npdu.NetworkLayerMessageType == ndpu.IamRouterToNetwork {
				c.utsm.Publish(int(npdu.Source.Net), networkList)
				//return
			}
			// Don't return here, APDU might follow? No, usually distinct.
			// But original code didn't return.
		}

		// We want to keep the APDU intact, so we will get a snapshot before decoding
		send := dec.Bytes()
		err = dec.APDU(&apdu)
		if err != nil {
			log.Printf("[ERROR] Issue decoding APDU: %v", err)
			return
		}
//		log.Printf("[DEBUG] APDU DataType=%d UnconfirmedService=%d", apdu.DataType, apdu.UnconfirmedService)
		switch apdu.DataType {
		case btypes.UnconfirmedServiceRequest:
			if apdu.UnconfirmedService == btypes.ServiceUnconfirmedIAm {
				dec := encoding.NewDecoder(apdu.RawData)
				iam := btypes.IAm{}
				err := dec.IAm(&iam)
				if err != nil {
					log.Printf("[DEBUG] Unable to decode IAm: %v", err)
					return
				}
				// Populate Source for IAm
				iam.Addr = *src
				//log.Printf("[INFO] Received IAm from %d, Src: %v", iam.ID.Instance, src)

				c.utsm.Publish(int(iam.ID.Instance), iam)
			} else if apdu.UnconfirmedService == btypes.ServiceUnconfirmedWhoIs {
				dec := encoding.NewDecoder(apdu.RawData)
				var low, high int32
				dec.WhoIs(&low, &high)
				//log.Printf("[DEBUG] Received WhoIs request (Low: %d, High: %d) from %v - ignoring", low, high, src)
			} else {
				//log.Printf("[DEBUG] Unconfirmed Service: %d from %v", apdu.UnconfirmedService, src)
			}
		case btypes.SimpleAck:
			log.Printf("[DEBUG] Received Simple Ack")
			err := c.tsm.Send(int(apdu.InvokeId), send)
			if err != nil {
				return
			}
		case btypes.ComplexAck:
			//log.Printf("[DEBUG] Received Complex Ack")
			err := c.tsm.Send(int(apdu.InvokeId), send)
			if err != nil {
				return
			}
		case btypes.ConfirmedServiceRequest:
			log.Printf("[DEBUG] Received  Confirmed Service Request")
			err := c.tsm.Send(int(apdu.InvokeId), send)
			if err != nil {
				return
			}
		case btypes.Error:
			err := fmt.Errorf("error class %s code %s", apdu.Error.Class.String(), apdu.Error.Code.String())
			err = c.tsm.Send(int(apdu.InvokeId), err)
			if err != nil {
				log.Printf("[DEBUG] unable to Send error to %d: %v", apdu.InvokeId, err)
			}
		default:
			// Ignore it
			log.Printf("[DEBUG] An ignored packet went through raw=%v", b)
		}
	}

	if header.Function == btypes.BacFuncForwardedNPDU {
		// Right now we are ignoring the NPDU data that is stored in the packet. Eventually
		// we will need to check it for any additional information we can gleam.
		// NDPU has source
		b = b[forwardHeaderLength:]
		log.Printf("[DEBUG] Ignored NDPU Forwarded")
	}
}

type SetBroadcastType struct { //used to override the header.Function
	Set     bool
	BacFunc btypes.BacFunc
}

// Send transfers the raw apdu byte slice to the destination address.
func (c *client) Send(dest btypes.Address, npdu *btypes.NPDU, data []byte, broadcastType *SetBroadcastType) (int, error) {
	//broadcastType = &SetBroadcastType{}
	var header btypes.BVLC
	// Set packet type
	header.Type = btypes.BVLCTypeBacnetIP
	//if Adr is > 0 it must be an mst-tp device so send a UNICAST
	// if len(dest.Adr) > 0 { //(aidan) not sure if this is correct, but it needs to be set to work to send (UNICAST) messages over a bacnet network
	// 	// SET UNICAST FLAG
	// 	// see http://www.bacnet.org/Tutorial/HMN-Overview/sld033.
	// 	// see https://github.com/JoelBender/bacpypes/blob/9fca3f608a97a20807cd188689a2b9ff60b05085/doc/source/gettingstarted/gettingstarted001.rst#udp-communications-issues
	// 	header.Function = btypes.BacFuncUnicast
	// } else

	if dest.IsBroadcast() || dest.IsSubBroadcast() {
		// SET BROADCAST FLAG
		header.Function = btypes.BacFuncBroadcast
	} else {
		// SET UNICAST FLAG
		header.Function = btypes.BacFuncUnicast
	}

	if broadcastType != nil {
		if broadcastType.Set {
			header.Function = broadcastType.BacFunc
		}
	}

	header.Length = uint16(mtuHeaderLength + len(data))
	header.Data = data
	e := encoding.NewEncoder()
	err := e.BVLC(header)
	if err != nil {
		return 0, err
	}
	// use default udp type, src = network address (nil)
	return c.dataLink.Send(e.Bytes(), npdu, &dest)
}

// Close free resources for the client. Always call this function when using NewClient
func (c *client) Close() error {
	if c.dataLink != nil {
		c.dataLink.Close()
	}
	c.running = false
	return nil
}

func (c *client) IsRunning() bool {
	return c.running
}
