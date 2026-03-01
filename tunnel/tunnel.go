package tunnel

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/vishvananda/netlink"
)

const (
	ifaceName  = "sproxy"
	routeTable = 101
)

type Tunnel struct {
	cfg *Config
}

func Setup(cfg *Config) (*Tunnel, error) {
	if old, err := netlink.LinkByName(ifaceName); err == nil {
		_ = netlink.LinkSetDown(old)
		_ = netlink.LinkDel(old)
	}

	localIP := net.ParseIP(cfg.Local).To4()
	remoteIP := net.ParseIP(cfg.Remote).To4()
	if localIP == nil || remoteIP == nil {
		return nil, fmt.Errorf("invalid local/remote IPv4")
	}

	sit := &netlink.Sittun{
		LinkAttrs: netlink.LinkAttrs{Name: ifaceName},
		Local:     localIP,
		Remote:    remoteIP,
		Ttl:       64,
	}
	if err := netlink.LinkAdd(sit); err != nil {
		if !errors.Is(err, syscall.EEXIST) {
			return nil, fmt.Errorf("create tunnel: %w", err)
		}
		// kernel hasn't released the interface yet, force delete and retry
		if old, lerr := netlink.LinkByName(ifaceName); lerr == nil {
			_ = netlink.LinkSetDown(old)
			_ = netlink.LinkDel(old)
		}
		if err := netlink.LinkAdd(sit); err != nil {
			return nil, fmt.Errorf("create tunnel: %w", err)
		}
	}

	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("find tunnel iface: %w", err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return nil, fmt.Errorf("set tunnel up: %w", err)
	}

	addr := &netlink.Addr{IPNet: &net.IPNet{
		IP:   cfg.ClientIP(),
		Mask: cfg.ClientNet().Mask,
	}}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return nil, fmt.Errorf("assign ipv6 addr: %w", err)
	}

	cidrRoute := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       cfg.CIDRNet(),
	}
	if err := netlink.RouteAdd(cidrRoute); err != nil {
		return nil, fmt.Errorf("add cidr route: %w", err)
	}

	rule := netlink.NewRule()
	rule.Src = cfg.CIDRNet()
	rule.Table = routeTable
	rule.Family = netlink.FAMILY_V6
	_ = netlink.RuleDel(rule)
	if err := netlink.RuleAdd(rule); err != nil {
		return nil, fmt.Errorf("add ip rule: %w", err)
	}

	defaultRoute := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Gw:        cfg.GatewayIP(),
		Dst: &net.IPNet{
			IP:   net.IPv6zero,
			Mask: net.CIDRMask(0, 128),
		},
		Table: routeTable,
	}
	if err := netlink.RouteAdd(defaultRoute); err != nil {
		return nil, fmt.Errorf("add default route in table %d: %w", routeTable, err)
	}

	if err := os.WriteFile("/proc/sys/net/ipv6/ip_nonlocal_bind", []byte("1"), 0644); err != nil {
		log.Printf("warn: set ip_nonlocal_bind: %v", err)
	}

	log.Printf("tunnel %s up, gateway %s, cidr %s", ifaceName, cfg.GatewayIPv6, cfg.CIDR)
	return &Tunnel{cfg: cfg}, nil
}

func (t *Tunnel) Teardown() {
	rule := netlink.NewRule()
	rule.Src = t.cfg.CIDRNet()
	rule.Table = routeTable
	rule.Family = netlink.FAMILY_V6
	if err := netlink.RuleDel(rule); err != nil {
		log.Printf("warn: remove ip rule: %v", err)
	}

	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		log.Printf("warn: find tunnel iface for cleanup: %v", err)
		return
	}
	if err := netlink.LinkDel(link); err != nil {
		log.Printf("warn: remove tunnel iface: %v", err)
	}

	log.Printf("tunnel %s removed", ifaceName)
}
