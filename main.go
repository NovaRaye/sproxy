package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raye/sproxy/proxy"
	"github.com/raye/sproxy/tunnel"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println("sproxy", version)
		return
	}

	cfg := &tunnel.Config{}

	flag.StringVar(&cfg.CIDR, "cidr", "", "IPv6 CIDR block to use for proxy IPs (required)")
	flag.StringVar(&cfg.Remote, "remote", "", "tunnel server IPv4 (enables tunnel mode)")
	flag.StringVar(&cfg.Local, "local", "", "your server's public IPv4 (tunnel mode)")
	flag.StringVar(&cfg.ClientIPv6, "client-ipv6", "", "tunnel client IPv6 with prefix, e.g. 2001:db8::2/64 (tunnel mode)")
	flag.StringVar(&cfg.GatewayIPv6, "gateway-ipv6", "", "tunnel gateway IPv6 (derived from --client-ipv6 if omitted)")
	flag.IntVar(&cfg.ProxyPort, "port", 1080, "proxy listen port")
	flag.StringVar(&cfg.ProxyBind, "bind", "0.0.0.0", "proxy bind address")
	flag.StringVar(&cfg.Username, "username", "", "auth username (optional)")
	flag.StringVar(&cfg.Password, "password", "", "auth password (optional)")
	flag.Parse()

	if cfg.CIDR == "" {
		fmt.Fprintf(os.Stderr, "Usage of sproxy:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	tunnelMode := cfg.Remote != ""
	if tunnelMode && (cfg.Local == "" || cfg.ClientIPv6 == "") {
		log.Fatal("tunnel mode requires --local and --client-ipv6")
	}

	if err := cfg.Derive(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	var t *tunnel.Tunnel
	if tunnelMode {
		log.Printf("tunnel mode: %s -> %s", cfg.Local, cfg.Remote)
		var err error
		t, err = tunnel.Setup(cfg)
		if err != nil {
			log.Fatalf("tunnel setup failed: %v", err)
		}
	} else {
		log.Printf("proxy-only mode")
	}

	log.Printf("starting proxy on %s:%d cidr=%s", cfg.ProxyBind, cfg.ProxyPort, cfg.CIDR)
	p, err := proxy.New(cfg)
	if err != nil {
		if t != nil {
			t.Teardown()
		}
		log.Fatalf("proxy start failed: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down...")
	p.Stop()
	if t != nil {
		t.Teardown()
	}
	log.Println("done")
}
