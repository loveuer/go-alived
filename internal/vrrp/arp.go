package vrrp

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/mdlayher/arp"
)

type ARPSender struct {
	client *arp.Client
	iface  *net.Interface
}

func NewARPSender(ifaceName string) (*ARPSender, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %s: %w", ifaceName, err)
	}

	client, err := arp.Dial(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to create ARP client: %w", err)
	}

	return &ARPSender{
		client: client,
		iface:  iface,
	}, nil
}

func (a *ARPSender) SendGratuitousARP(ip net.IP) error {
	if ip4 := ip.To4(); ip4 == nil {
		return fmt.Errorf("invalid IPv4 address: %s", ip)
	}

	addr, err := netip.ParseAddr(ip.String())
	if err != nil {
		return fmt.Errorf("failed to parse IP: %w", err)
	}

	pkt, err := arp.NewPacket(
		arp.OperationRequest,
		a.iface.HardwareAddr,
		addr,
		net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		addr,
	)
	if err != nil {
		return fmt.Errorf("failed to create ARP packet: %w", err)
	}

	if err := a.client.WriteTo(pkt, net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}); err != nil {
		return fmt.Errorf("failed to send gratuitous ARP: %w", err)
	}

	return nil
}

func (a *ARPSender) SendGratuitousARPForIPs(ips []net.IP) error {
	for _, ip := range ips {
		if err := a.SendGratuitousARP(ip); err != nil {
			return err
		}
	}
	return nil
}

func (a *ARPSender) Close() error {
	return a.client.Close()
}
