package tunnel

import (
	"fmt"
	"net"
)

type Config struct {
	Remote      string
	Local       string
	ClientIPv6  string
	GatewayIPv6 string
	CIDR        string
	ProxyPort   int
	ProxyBind   string
	Username    string
	Password    string

	// derived
	clientIP  net.IP
	clientNet *net.IPNet
	gatewayIP net.IP
	cidrNet   *net.IPNet
}

func (c *Config) Derive() error {
	_, cidrNet, err := net.ParseCIDR(c.CIDR)
	if err != nil {
		return fmt.Errorf("invalid cidr %q: %w", c.CIDR, err)
	}
	c.cidrNet = cidrNet

	if c.ClientIPv6 != "" {
		ip, ipnet, err := net.ParseCIDR(c.ClientIPv6)
		if err != nil {
			return fmt.Errorf("invalid client-ipv6 %q: %w", c.ClientIPv6, err)
		}
		c.clientIP = ip
		c.clientNet = ipnet

		if c.GatewayIPv6 == "" {
			gw := make(net.IP, 16)
			copy(gw, ipnet.IP.To16())
			gw[15] = 1
			c.gatewayIP = gw
			c.GatewayIPv6 = gw.String()
		} else {
			c.gatewayIP = net.ParseIP(c.GatewayIPv6)
			if c.gatewayIP == nil {
				return fmt.Errorf("invalid gateway-ipv6 %q", c.GatewayIPv6)
			}
		}
	}

	return nil
}

func (c *Config) ClientIP() net.IP      { return c.clientIP }
func (c *Config) ClientNet() *net.IPNet { return c.clientNet }
func (c *Config) GatewayIP() net.IP     { return c.gatewayIP }
func (c *Config) CIDRNet() *net.IPNet   { return c.cidrNet }
