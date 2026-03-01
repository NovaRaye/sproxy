package tunnel

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	ifaceName  = "sproxy"
	routeTable = 101
)

type Tunnel struct {
	cfg     *Config
	created bool
}

func findExistingSitTunnel(localIP, remoteIP net.IP) netlink.Link {
	links, err := netlink.LinkList()
	if err != nil {
		return nil
	}
	for _, link := range links {
		sit, ok := link.(*netlink.Sittun)
		if !ok {
			continue
		}
		if sit.Local.Equal(localIP) && sit.Remote.Equal(remoteIP) {
			return link
		}
	}
	return nil
}

func Setup(cfg *Config) (*Tunnel, error) {
	localIP := net.ParseIP(cfg.Local).To4()
	remoteIP := net.ParseIP(cfg.Remote).To4()
	if localIP == nil || remoteIP == nil {
		return nil, fmt.Errorf("invalid local/remote IPv4")
	}

	var link netlink.Link
	created := false

	if existing := findExistingSitTunnel(localIP, remoteIP); existing != nil {
		log.Printf("reusing existing tunnel interface %s", existing.Attrs().Name)
		link = existing
	} else {
		if old, err := netlink.LinkByName(ifaceName); err == nil {
			_ = netlink.LinkSetDown(old)
			_ = netlink.LinkDel(old)
		}
		sit := &netlink.Sittun{
			LinkAttrs: netlink.LinkAttrs{Name: ifaceName},
			Local:     localIP,
			Remote:    remoteIP,
			Ttl:       64,
		}
		if err := netlink.LinkAdd(sit); err != nil {
			return nil, fmt.Errorf("create tunnel: %w", err)
		}
		var err error
		link, err = netlink.LinkByName(ifaceName)
		if err != nil {
			return nil, fmt.Errorf("find tunnel iface: %w", err)
		}
		created = true
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return nil, fmt.Errorf("set tunnel up: %w", err)
	}

	addr := &netlink.Addr{IPNet: &net.IPNet{
		IP:   cfg.ClientIP(),
		Mask: cfg.ClientNet().Mask,
	}}
	if err := netlink.AddrAdd(link, addr); err != nil && !errors.Is(err, syscall.EEXIST) {
		return nil, fmt.Errorf("assign ipv6 addr: %w", err)
	}

	cidrRoute := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       cfg.CIDRNet(),
	}
	if err := netlink.RouteAdd(cidrRoute); err != nil && !errors.Is(err, syscall.EEXIST) {
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
	if err := netlink.RouteAdd(defaultRoute); err != nil && !errors.Is(err, syscall.EEXIST) {
		return nil, fmt.Errorf("add default route in table %d: %w", routeTable, err)
	}

	if err := os.WriteFile("/proc/sys/net/ipv6/ip_nonlocal_bind", []byte("1"), 0644); err != nil {
		log.Printf("warn: set ip_nonlocal_bind: %v", err)
	}

	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return nil, fmt.Errorf("find lo: %w", err)
	}
	localRoute := &netlink.Route{
		LinkIndex: lo.Attrs().Index,
		Dst:       cfg.CIDRNet(),
		Type:      unix.RTN_LOCAL,
		Table:     unix.RT_TABLE_LOCAL,
	}
	if err := netlink.RouteAdd(localRoute); err != nil && !errors.Is(err, syscall.EEXIST) {
		return nil, fmt.Errorf("add local cidr route: %w", err)
	}

	log.Printf("tunnel %s up, gateway %s, cidr %s", link.Attrs().Name, cfg.GatewayIPv6, cfg.CIDR)
	return &Tunnel{cfg: cfg, created: created}, nil
}

func (t *Tunnel) Teardown() {
	rule := netlink.NewRule()
	rule.Src = t.cfg.CIDRNet()
	rule.Table = routeTable
	rule.Family = netlink.FAMILY_V6
	if err := netlink.RuleDel(rule); err != nil {
		log.Printf("warn: remove ip rule: %v", err)
	}

	if lo, err := netlink.LinkByName("lo"); err == nil {
		localRoute := &netlink.Route{
			LinkIndex: lo.Attrs().Index,
			Dst:       t.cfg.CIDRNet(),
			Type:      unix.RTN_LOCAL,
			Table:     unix.RT_TABLE_LOCAL,
		}
		if err := netlink.RouteDel(localRoute); err != nil {
			log.Printf("warn: remove local cidr route: %v", err)
		}
	}

	if t.created {
		if link, err := netlink.LinkByName(ifaceName); err == nil {
			_ = netlink.LinkDel(link)
		}
		log.Printf("tunnel %s removed", ifaceName)
	}
}
