package proxy

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/raye/sproxy/tunnel"
)

type Proxy struct {
	server *http.Server
	cfg    *tunnel.Config
}

func New(cfg *tunnel.Config) (*Proxy, error) {
	p := &Proxy{cfg: cfg}
	addr := fmt.Sprintf("%s:%d", cfg.ProxyBind, cfg.ProxyPort)
	p.server = &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(p.handle),
	}

	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("proxy error: %v", err)
		}
	}()

	return p, nil
}

func (p *Proxy) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = p.server.Shutdown(ctx)
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	if p.cfg.Username != "" {
		u, pass, ok := r.BasicAuth()
		if !ok || u != p.cfg.Username || pass != p.cfg.Password {
			w.Header().Set("Proxy-Authenticate", `Basic realm="sproxy"`)
			http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
			return
		}
	}

	if r.Method == http.MethodConnect {
		p.handleTunnel(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

func (p *Proxy) randomIPv6() net.IP {
	cidr := p.cfg.CIDRNet()
	base := cidr.IP.To16()
	mask := cidr.Mask

	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)

	ip := make(net.IP, 16)
	for i := 0; i < 16; i++ {
		ip[i] = (base[i] & mask[i]) | (randBytes[i] & ^mask[i])
	}
	return ip
}

func (p *Proxy) dialWithRandomIPv6(addr string) (net.Conn, error) {
	srcIP := p.randomIPv6()
	localAddr := &net.TCPAddr{IP: srcIP}
	dialer := &net.Dialer{
		LocalAddr: localAddr,
		Timeout:   15 * time.Second,
	}
	conn, err := dialer.Dial("tcp6", addr)
	if err != nil {
		return nil, err
	}
	log.Printf("connect %s via %s", addr, srcIP)
	return conn, nil
}

func (p *Proxy) handleTunnel(w http.ResponseWriter, r *http.Request) {
	dst, err := p.dialWithRandomIPv6(r.Host)
	if err != nil {
		log.Printf("dial %s failed: %v", r.Host, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer dst.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	client, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	_, err = fmt.Fprint(client, "HTTP/1.1 200 Connection Established\r\n\r\n")
	if err != nil {
		return
	}

	pipe(client, dst)
}

func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return p.dialWithRandomIPv6(addr)
		},
	}

	r.RequestURI = ""
	r.Header.Del("Proxy-Connection")

	resp, err := transport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}

	go cp(a, b)
	go cp(b, a)

	<-done
	_ = a.SetReadDeadline(time.Now())
	_ = b.SetReadDeadline(time.Now())
	<-done
}
