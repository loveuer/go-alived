package vrrp

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"golang.org/x/net/ipv4"
)

const (
	VRRPMulticastAddr = "224.0.0.18"
)

type Socket struct {
	conn     *ipv4.RawConn
	iface    *net.Interface
	localIP  net.IP
	groupIP  net.IP
}

func NewSocket(ifaceName string) (*Socket, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %s: %w", ifaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for %s: %w", ifaceName, err)
	}

	var localIP net.IP
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipv4 := ipNet.IP.To4(); ipv4 != nil {
				localIP = ipv4
				break
			}
		}
	}

	if localIP == nil {
		return nil, fmt.Errorf("no IPv4 address found on interface %s", ifaceName)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, VRRPProtocolNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to create raw socket: %w", err)
	}

	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to set SO_REUSEADDR: %w", err)
	}

	file := os.NewFile(uintptr(fd), "vrrp-socket")
	defer file.Close()

	packetConn, err := net.FilePacketConn(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create packet connection: %w", err)
	}

	rawConn, err := ipv4.NewRawConn(packetConn)
	if err != nil {
		packetConn.Close()
		return nil, fmt.Errorf("failed to create raw connection: %w", err)
	}

	groupIP := net.ParseIP(VRRPMulticastAddr).To4()
	if groupIP == nil {
		rawConn.Close()
		return nil, fmt.Errorf("invalid multicast address: %s", VRRPMulticastAddr)
	}

	if err := rawConn.JoinGroup(iface, &net.IPAddr{IP: groupIP}); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("failed to join multicast group: %w", err)
	}

	if err := rawConn.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("failed to set control message: %w", err)
	}

	return &Socket{
		conn:    rawConn,
		iface:   iface,
		localIP: localIP,
		groupIP: groupIP,
	}, nil
}

func (s *Socket) Send(pkt *VRRPPacket) error {
	data, err := pkt.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal packet: %w", err)
	}

	header := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      0xC0,
		TotalLen: ipv4.HeaderLen + len(data),
		TTL:      255,
		Protocol: VRRPProtocolNumber,
		Dst:      s.groupIP,
		Src:      s.localIP,
	}

	if err := s.conn.WriteTo(header, data, nil); err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}

func (s *Socket) Receive() (*VRRPPacket, net.IP, error) {
	buf := make([]byte, 1500)

	header, payload, _, err := s.conn.ReadFrom(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to receive packet: %w", err)
	}

	pkt, err := Unmarshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal packet: %w", err)
	}

	return pkt, header.Src, nil
}

func (s *Socket) Close() error {
	if err := s.conn.LeaveGroup(s.iface, &net.IPAddr{IP: s.groupIP}); err != nil {
		return err
	}
	return s.conn.Close()
}