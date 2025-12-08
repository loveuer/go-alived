package vrrp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

const (
	VRRPVersion        = 2
	VRRPProtocolNumber = 112
)

type VRRPPacket struct {
	Version       uint8
	Type          uint8
	VirtualRtrID  uint8
	Priority      uint8
	CountIPAddrs  uint8
	AuthType      uint8
	AdvertInt     uint8
	Checksum      uint16
	IPAddresses   []net.IP
	AuthData      [8]byte
}

const (
	VRRPTypeAdvertisement = 1
)

const (
	AuthTypeNone       = 0
	AuthTypeSimpleText = 1
	AuthTypeIPAH       = 2
)

func NewAdvertisement(vrID uint8, priority uint8, advertInt uint8, ips []net.IP, authType uint8, authPass string) *VRRPPacket {
	pkt := &VRRPPacket{
		Version:      VRRPVersion,
		Type:         VRRPTypeAdvertisement,
		VirtualRtrID: vrID,
		Priority:     priority,
		CountIPAddrs: uint8(len(ips)),
		AuthType:     authType,
		AdvertInt:    advertInt,
		IPAddresses:  ips,
	}

	if authType == AuthTypeSimpleText && authPass != "" {
		copy(pkt.AuthData[:], authPass)
	}

	return pkt
}

func (p *VRRPPacket) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)

	versionType := (p.Version << 4) | p.Type
	if err := binary.Write(buf, binary.BigEndian, versionType); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, p.VirtualRtrID); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, p.Priority); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, p.CountIPAddrs); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, p.AuthType); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, p.AdvertInt); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint16(0)); err != nil {
		return nil, err
	}

	for _, ip := range p.IPAddresses {
		ip4 := ip.To4()
		if ip4 == nil {
			return nil, fmt.Errorf("invalid IPv4 address: %s", ip)
		}
		if err := binary.Write(buf, binary.BigEndian, ip4); err != nil {
			return nil, err
		}
	}

	if err := binary.Write(buf, binary.BigEndian, p.AuthData); err != nil {
		return nil, err
	}

	data := buf.Bytes()

	checksum := calculateChecksum(data)
	binary.BigEndian.PutUint16(data[6:8], checksum)

	return data, nil
}

func Unmarshal(data []byte) (*VRRPPacket, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("packet too short: %d bytes", len(data))
	}

	pkt := &VRRPPacket{}

	versionType := data[0]
	pkt.Version = versionType >> 4
	pkt.Type = versionType & 0x0F
	pkt.VirtualRtrID = data[1]
	pkt.Priority = data[2]
	pkt.CountIPAddrs = data[3]
	pkt.AuthType = data[4]
	pkt.AdvertInt = data[5]
	pkt.Checksum = binary.BigEndian.Uint16(data[6:8])

	offset := 8
	pkt.IPAddresses = make([]net.IP, pkt.CountIPAddrs)
	for i := 0; i < int(pkt.CountIPAddrs); i++ {
		if offset+4 > len(data) {
			return nil, fmt.Errorf("packet too short for IP addresses")
		}
		pkt.IPAddresses[i] = net.IPv4(data[offset], data[offset+1], data[offset+2], data[offset+3])
		offset += 4
	}

	if offset+8 > len(data) {
		return nil, fmt.Errorf("packet too short for auth data")
	}
	copy(pkt.AuthData[:], data[offset:offset+8])

	return pkt, nil
}

func calculateChecksum(data []byte) uint16 {
	sum := uint32(0)

	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}

	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return uint16(^sum)
}

func (p *VRRPPacket) Validate(authPass string) error {
	if p.Version != VRRPVersion {
		return fmt.Errorf("unsupported VRRP version: %d", p.Version)
	}

	if p.Type != VRRPTypeAdvertisement {
		return fmt.Errorf("unsupported VRRP type: %d", p.Type)
	}

	if p.AuthType == AuthTypeSimpleText {
		if authPass != "" {
			var expectedAuth [8]byte
			copy(expectedAuth[:], authPass)
			if !bytes.Equal(p.AuthData[:], expectedAuth[:]) {
				return fmt.Errorf("authentication failed")
			}
		}
	}

	return nil
}
