package netif

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type Interface struct {
	Name  string
	Index int
	Link  netlink.Link
}

func GetInterface(name string) (*Interface, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to find interface %s: %w", name, err)
	}

	return &Interface{
		Name:  name,
		Index: link.Attrs().Index,
		Link:  link,
	}, nil
}

func (iface *Interface) AddIP(ipCIDR string) error {
	addr, err := netlink.ParseAddr(ipCIDR)
	if err != nil {
		return fmt.Errorf("invalid IP address %s: %w", ipCIDR, err)
	}

	if err := netlink.AddrAdd(iface.Link, addr); err != nil {
		return fmt.Errorf("failed to add IP %s to %s: %w", ipCIDR, iface.Name, err)
	}

	return nil
}

func (iface *Interface) DeleteIP(ipCIDR string) error {
	addr, err := netlink.ParseAddr(ipCIDR)
	if err != nil {
		return fmt.Errorf("invalid IP address %s: %w", ipCIDR, err)
	}

	if err := netlink.AddrDel(iface.Link, addr); err != nil {
		return fmt.Errorf("failed to delete IP %s from %s: %w", ipCIDR, iface.Name, err)
	}

	return nil
}

func (iface *Interface) HasIP(ipCIDR string) (bool, error) {
	targetAddr, err := netlink.ParseAddr(ipCIDR)
	if err != nil {
		return false, fmt.Errorf("invalid IP address %s: %w", ipCIDR, err)
	}

	addrs, err := netlink.AddrList(iface.Link, 0)
	if err != nil {
		return false, fmt.Errorf("failed to list addresses on %s: %w", iface.Name, err)
	}

	for _, addr := range addrs {
		if addr.IPNet.String() == targetAddr.IPNet.String() {
			return true, nil
		}
	}

	return false, nil
}

func (iface *Interface) GetHardwareAddr() (net.HardwareAddr, error) {
	return iface.Link.Attrs().HardwareAddr, nil
}

func (iface *Interface) IsUp() bool {
	return iface.Link.Attrs().Flags&net.FlagUp != 0
}