package datalink

import (
	"context"
	"fmt"
	//"log"
	"net"
	"strings"
	"syscall"

	"edge-gateway/internal/driver/bacnet/btypes"
)

// DefaultPort that BacnetIP will use if a port is not given. Valid ports for
// the bacnet protocol is between 0xBAC0 and 0xBAC9
const DefaultPort = 0xBAC0 //47808

type udpDataLink struct {
	// netInterface                *net.Interface
	myAddress, broadcastAddress *btypes.Address
	port                        int
	listener                    *net.UDPConn
}

/*
NewUDPDataLink returns udp listener
pass in your iface port by name, see an alternative NewUDPDataLinkFromIP if you wish to pass in by ip and subnet
  - inter: eth0
  - addr: 47808
*/
func NewUDPDataLink(inter string, port int) (link DataLink, err error) {
	if port == 0 {
		port = DefaultPort
	}
	addr := inter
	if !strings.ContainsRune(inter, '/') {
		addr, err = FindCIDRAddress(inter)
		if err != nil {
			return nil, err
		}
		fmt.Printf("find addr: %s -> %s\n", inter, addr)
	}
	link, err = dataLink(addr, port)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func NewUDPDataLinkFromCIDR(cidr string, port int) (link DataLink, err error) {
	return dataLink(cidr, port)
}

/*
NewUDPDataLinkFromIP returns udp listener
  - addr: 192.168.15.10
  - subNet: 24
  - addr: 47808
*/
func NewUDPDataLinkFromIP(addr string, subNet, port int) (link DataLink, err error) {
	addr = fmt.Sprintf("%s/%d", addr, subNet)
	link, err = dataLink(addr, port)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func dataLink(ipAddr string, port int) (DataLink, error) {
	if port == 0 {
		port = DefaultPort
	}

	ip, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return nil, err
	}

	broadcast := net.IP(make([]byte, 4))
	for i := range broadcast {
		broadcast[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}

	udpAddrStr := fmt.Sprintf("%s:%d", ip.String(), port)

	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				opErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}

	conn, err := lc.ListenPacket(context.Background(), "udp4", udpAddrStr)
	if err != nil {
		return nil, err
	}

	udpConn := conn.(*net.UDPConn)

	//log.Printf("[DEBUG] UDP DataLink listening on %s:%d (Broadcast: %s)\n", ipAddr, port, broadcast.String())

	return &udpDataLink{
		listener:         udpConn,
		port:             port,
		myAddress:        IPPortToAddress(ip, port),
		broadcastAddress: IPPortToAddress(broadcast, DefaultPort),
	}, nil
}

func (c *udpDataLink) Close() error {
	if c.listener != nil {
		return c.listener.Close()
	}
	return nil
}

func (c *udpDataLink) Receive(data []byte) (*btypes.Address, int, error) {
	n, adr, err := c.listener.ReadFromUDP(data)
	if err != nil {
		return nil, n, err
	}
	//log.Printf("[DEBUG] UDP Received %d bytes from %s\n", n, adr.String())
	adr.IP = adr.IP.To4()
	udpAddr := UDPToAddress(adr)
	return udpAddr, n, nil
}

func (c *udpDataLink) GetMyAddress() *btypes.Address {
	return c.myAddress
}

// GetBroadcastAddress uses the given address with subnet to return the broadcast address
func (c *udpDataLink) GetBroadcastAddress() *btypes.Address {
	return c.broadcastAddress
}

func (c *udpDataLink) Send(data []byte, npdu *btypes.NPDU, dest *btypes.Address) (int, error) {
	// Get IP Address
	d, err := dest.UDPAddr()
	if err != nil {
		return 0, err
	}
	//log.Printf("[DEBUG] UDP Sending %d bytes to %s\n", len(data), d.String())
	return c.listener.WriteTo(data, &d)
}

// IPPortToAddress converts a given udp address into a bacnet address
func IPPortToAddress(ip net.IP, port int) *btypes.Address {
	return UDPToAddress(&net.UDPAddr{
		IP:   ip.To4(),
		Port: port,
	})
}

// UDPToAddress converts a given udp address into a bacnet address
func UDPToAddress(n *net.UDPAddr) *btypes.Address {
	a := &btypes.Address{}
	p := uint16(n.Port)
	// Length of IP plus the port
	length := net.IPv4len + 2
	a.Mac = make([]uint8, length)
	//Encode ip
	for i := 0; i < net.IPv4len; i++ {
		a.Mac[i] = n.IP[i]
	}
	// Encode port
	a.Mac[net.IPv4len+0] = uint8(p >> 8)
	a.Mac[net.IPv4len+1] = uint8(p & 0x00FF)

	a.MacLen = uint8(length)
	return a
}

// FindCIDRAddress find out CIDR address from net interface
func FindCIDRAddress(inter string) (string, error) {
	i, err := net.InterfaceByName(inter)
	if err != nil {
		return "", err
	}

	uni, err := i.Addrs()
	if err != nil {
		return "", err
	}

	if len(uni) == 0 {
		return "", fmt.Errorf("interface %s has no addresses", inter)
	}

	// Find the first IP4 ip
	for _, adr := range uni {
		IP, _, _ := net.ParseCIDR(adr.String())

		// To4 is non nil when the type is ip4
		if IP.To4() != nil {
			return adr.String(), nil
		}
	}
	// We couldn't find a interface or all of them are ip6
	return "", fmt.Errorf("no valid broadcasting address was found on interface %s", inter)
}
