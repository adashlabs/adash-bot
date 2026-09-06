package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type dnsCacheEntry struct {
	ips     []net.IP
	expires time.Time
}

type DoHResolver struct {
	client    *http.Client
	endpoints []string
	mu        sync.RWMutex
	cache     map[string]dnsCacheEntry
}

func newDoHResolver() *DoHResolver {
	// Standard HTTP transport using direct IP addresses for DoH endpoints
	// to avoid needing a bootstrap DNS server.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &DoHResolver{
		client: &http.Client{
			Transport: transport,
			Timeout:   3 * time.Second,
		},
		endpoints: []string{
			"https://1.1.1.1/dns-query",
			"https://1.0.0.1/dns-query",
			"https://8.8.8.8/dns-query",
			"https://9.9.9.9/dns-query",
		},
		cache: make(map[string]dnsCacheEntry),
	}
}

func (r *DoHResolver) buildDNSQuery(domain string) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x12, 0x34}) // ID
	buf.Write([]byte{0x01, 0x00}) // Flags: standard query, recursion desired
	buf.Write([]byte{0x00, 0x01}) // QDCOUNT: 1
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	parts := strings.Split(domain, ".")
	for _, part := range parts {
		buf.WriteByte(byte(len(part)))
		buf.WriteString(part)
	}
	buf.WriteByte(0x00)

	buf.Write([]byte{0x00, 0x01}) // QTYPE: A (IPv4)
	buf.Write([]byte{0x00, 0x01}) // QCLASS: IN
	return buf.Bytes()
}

func (r *DoHResolver) parseDNSResponse(data []byte) ([]net.IP, time.Duration, error) {
	if len(data) < 12 {
		return nil, 0, errors.New("dns response too short")
	}
	ancount := binary.BigEndian.Uint16(data[6:8])
	if ancount == 0 {
		return nil, 0, errors.New("no answers in dns response")
	}

	offset := 12
	// Skip Question section
	for offset < len(data) {
		lenByte := int(data[offset])
		if lenByte == 0 {
			offset += 5 // null + type(2) + class(2)
			break
		}
		offset += 1 + lenByte
	}

	var ips []net.IP
	minTTL := uint32(300)

	for i := 0; i < int(ancount) && offset < len(data); i++ {
		if offset+10 > len(data) {
			break
		}
		if data[offset]&0xC0 == 0xC0 {
			offset += 2
		} else {
			for offset < len(data) && data[offset] != 0 {
				offset += 1 + int(data[offset])
			}
			offset++
		}
		if offset+10 > len(data) {
			break
		}

		rtype := binary.BigEndian.Uint16(data[offset : offset+2])
		ttl := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		rdlength := binary.BigEndian.Uint16(data[offset+8 : offset+10])
		offset += 10

		if offset+int(rdlength) > len(data) {
			break
		}

		if rtype == 1 && rdlength == 4 { // Type A (IPv4)
			ip := make(net.IP, 4)
			copy(ip, data[offset:offset+4])
			ips = append(ips, ip)
			if ttl < minTTL && ttl > 10 {
				minTTL = ttl
			}
		}
		offset += int(rdlength)
	}

	if len(ips) == 0 {
		return nil, 0, errors.New("no IPv4 records in response")
	}
	return ips, time.Duration(minTTL) * time.Second, nil
}

func (r *DoHResolver) Resolve(ctx context.Context, domain string) (net.IP, error) {
	// If already an IP address, return directly
	if ip := net.ParseIP(domain); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return v4, nil
		}
		return ip, nil
	}

	domain = strings.ToLower(strings.TrimSuffix(domain, "."))

	// Check cache
	r.mu.RLock()
	entry, found := r.cache[domain]
	r.mu.RUnlock()

	if found && time.Now().Before(entry.expires) && len(entry.ips) > 0 {
		return entry.ips[0], nil
	}

	// Query DoH
	queryPacket := r.buildDNSQuery(domain)

	type result struct {
		ips []net.IP
		ttl time.Duration
		err error
	}
	resChan := make(chan result, len(r.endpoints))

	for _, ep := range r.endpoints {
		go func(endpoint string) {
			req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(queryPacket))
			if err != nil {
				resChan <- result{err: err}
				return
			}
			req.Header.Set("Content-Type", "application/dns-message")
			req.Header.Set("Accept", "application/dns-message")

			resp, err := r.client.Do(req)
			if err != nil {
				resChan <- result{err: err}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				resChan <- result{err: fmt.Errorf("status %d from %s", resp.StatusCode, endpoint)}
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				resChan <- result{err: err}
				return
			}

			ips, ttl, err := r.parseDNSResponse(body)
			resChan <- result{ips: ips, ttl: ttl, err: err}
		}(ep)
	}

	var firstErr error
	for i := 0; i < len(r.endpoints); i++ {
		res := <-resChan
		if res.err == nil && len(res.ips) > 0 {
			// Update cache
			r.mu.Lock()
			r.cache[domain] = dnsCacheEntry{
				ips:     res.ips,
				expires: time.Now().Add(res.ttl),
			}
			r.mu.Unlock()
			return res.ips[0], nil
		}
		if firstErr == nil && res.err != nil {
			firstErr = res.err
		}
	}

	// Fallback to standard system lookup if DoH fails
	sysIPs, err := net.LookupIP(domain)
	if err == nil {
		for _, ip := range sysIPs {
			if v4 := ip.To4(); v4 != nil {
				r.mu.Lock()
				r.cache[domain] = dnsCacheEntry{
					ips:     []net.IP{v4},
					expires: time.Now().Add(1 * time.Minute),
				}
				r.mu.Unlock()
				return v4, nil
			}
		}
	}

	if firstErr != nil {
		return nil, fmt.Errorf("doh resolve %s: %w", domain, firstErr)
	}
	return nil, fmt.Errorf("failed to resolve %s", domain)
}

func dialSOCKS5(socksAddr string, targetHost string, targetPort int) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", socksAddr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	// SOCKS5 Handshake: Version 5, 1 Auth Method, Method 0x00 (No Auth)
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		conn.Close()
		return nil, err
	}

	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		conn.Close()
		return nil, err
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("socks5 auth rejected: %v", resp)
	}

	// SOCKS5 CONNECT Request
	var req bytes.Buffer
	req.Write([]byte{0x05, 0x01, 0x00}) // VER, CMD (CONNECT), RSV

	if ip := net.ParseIP(targetHost); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			req.WriteByte(0x01) // IPv4
			req.Write(v4)
		} else {
			req.WriteByte(0x04) // IPv6
			req.Write(ip.To16())
		}
	} else {
		req.WriteByte(0x03) // DOMAINNAME
		req.WriteByte(byte(len(targetHost)))
		req.WriteString(targetHost)
	}

	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(targetPort))
	req.Write(portBuf)

	if _, err := conn.Write(req.Bytes()); err != nil {
		conn.Close()
		return nil, err
	}

	// Read SOCKS5 Response: VER, REP, RSV, ATYP
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		conn.Close()
		return nil, err
	}
	if header[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("socks5 connect failed with code: 0x%02x", header[1])
	}

	// Read and discard BND.ADDR + BND.PORT
	switch header[3] {
	case 0x01: // IPv4
		discard := make([]byte, 4+2)
		if _, err := io.ReadFull(conn, discard); err != nil {
			conn.Close()
			return nil, err
		}
	case 0x03: // DOMAINNAME
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			conn.Close()
			return nil, err
		}
		discard := make([]byte, int(lenBuf[0])+2)
		if _, err := io.ReadFull(conn, discard); err != nil {
			conn.Close()
			return nil, err
		}
	case 0x04: // IPv6
		discard := make([]byte, 16+2)
		if _, err := io.ReadFull(conn, discard); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

type ProxyBridge struct {
	doh       *DoHResolver
	socksAddr string
}

func (p *ProxyBridge) handleHTTP(w http.ResponseWriter, r *http.Request) {
	// Health check endpoint
	if r.Method == http.MethodGet && (r.URL.Path == "/health" || r.URL.Path == "/healthz") {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
		return
	}

	if r.Method != http.MethodConnect {
		http.Error(w, "Only HTTP CONNECT is supported", http.StatusMethodNotAllowed)
		return
	}

	host, portStr, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
		portStr = "443"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 443
	}

	// Resolve target host using DoH
	resolvedTarget := host
	if ip := net.ParseIP(host); ip == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		resolvedIP, err := p.doh.Resolve(ctx, host)
		cancel()
		if err == nil {
			resolvedTarget = resolvedIP.String()
		} else {
			log.Printf("[bridge] doh resolve failed for %s, passing domain to byedpi: %v", host, err)
		}
	}

	// Connect through ByeDPI SOCKS5
	socksConn, err := dialSOCKS5(p.socksAddr, resolvedTarget, port)
	if err != nil {
		log.Printf("[bridge] failed to dial socks5 for %s:%d (target: %s): %v", host, port, resolvedTarget, err)
		http.Error(w, fmt.Sprintf("Failed to connect upstream: %v", err), http.StatusBadGateway)
		return
	}
	defer socksConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	// Send 200 Connection Established
	if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	// Tunnel bidirectionally
	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(socksConn, clientConn)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, socksConn)
		errc <- err
	}()

	<-errc
}

func (p *ProxyBridge) handleSOCKS5(clientConn net.Conn) {
	defer clientConn.Close()

	// SOCKS5 Handshake
	authBuf := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, authBuf); err != nil {
		return
	}
	if authBuf[0] != 0x05 {
		return
	}
	numMethods := int(authBuf[1])
	methods := make([]byte, numMethods)
	if _, err := io.ReadFull(clientConn, methods); err != nil {
		return
	}

	// Accept no auth
	if _, err := clientConn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	// Read request: VER, CMD, RSV, ATYP
	reqHeader := make([]byte, 4)
	if _, err := io.ReadFull(clientConn, reqHeader); err != nil {
		return
	}
	if reqHeader[0] != 0x05 || reqHeader[1] != 0x01 { // Only CONNECT command
		clientConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var targetHost string
	switch reqHeader[3] {
	case 0x01: // IPv4
		ipBuf := make([]byte, 4)
		if _, err := io.ReadFull(clientConn, ipBuf); err != nil {
			return
		}
		targetHost = net.IP(ipBuf).String()
	case 0x03: // DOMAINNAME
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(clientConn, lenBuf); err != nil {
			return
		}
		domainBuf := make([]byte, lenBuf[0])
		if _, err := io.ReadFull(clientConn, domainBuf); err != nil {
			return
		}
		targetHost = string(domainBuf)
	case 0x04: // IPv6
		ipBuf := make([]byte, 16)
		if _, err := io.ReadFull(clientConn, ipBuf); err != nil {
			return
		}
		targetHost = net.IP(ipBuf).String()
	default:
		return
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, portBuf); err != nil {
		return
	}
	port := int(binary.BigEndian.Uint16(portBuf))

	// Resolve domain via DoH
	resolvedTarget := targetHost
	if ip := net.ParseIP(targetHost); ip == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resolvedIP, err := p.doh.Resolve(ctx, targetHost)
		cancel()
		if err == nil {
			resolvedTarget = resolvedIP.String()
		}
	}

	// Connect to ByeDPI SOCKS5
	socksConn, err := dialSOCKS5(p.socksAddr, resolvedTarget, port)
	if err != nil {
		clientConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer socksConn.Close()

	// Respond 0x00 (Success)
	if _, err := clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0, 0}); err != nil {
		return
	}

	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(socksConn, clientConn)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, socksConn)
		errc <- err
	}()

	<-errc
}

func startByeDPI(ctx context.Context, ciadpiBin, socksAddr, argsStr string) {
	host, port, err := net.SplitHostPort(socksAddr)
	if err != nil {
		host = "127.0.0.1"
		port = "1081"
	}

	defaultArgs := []string{"-i", host, "-p", port}
	customArgs := strings.Fields(argsStr)
	cmdArgs := append(defaultArgs, customArgs...)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			log.Printf("[ciadpi] starting: %s %s", ciadpiBin, strings.Join(cmdArgs, " "))
			cmd := exec.CommandContext(ctx, ciadpiBin, cmdArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Printf("[ciadpi] process exited: %v (restarting in 1s)", err)
					time.Sleep(1 * time.Second)
				}
			}
		}
	}()
}

func main() {
	httpListenAddr := os.Getenv("HTTP_LISTEN_ADDR")
	if httpListenAddr == "" {
		httpListenAddr = ":8080"
	}

	socksListenAddr := os.Getenv("SOCKS5_LISTEN_ADDR")
	if socksListenAddr == "" {
		socksListenAddr = ":1080"
	}

	byedpiAddr := os.Getenv("BYEDPI_ADDR")
	if byedpiAddr == "" {
		byedpiAddr = "127.0.0.1:1081"
	}

	byedpiArgs := os.Getenv("BYEDPI_ARGS")
	if byedpiArgs == "" {
		byedpiArgs = "--split 1 --disorder 3+s --auto=torst --tlsrec 1+s"
	}

	ciadpiBin := os.Getenv("CIADPI_BIN")
	if ciadpiBin == "" {
		ciadpiBin = "/usr/local/bin/ciadpi"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// If ciadpi binary exists or path is specified, supervise it
	if _, err := exec.LookPath(ciadpiBin); err == nil {
		startByeDPI(ctx, ciadpiBin, byedpiAddr, byedpiArgs)
		// Give ciadpi 200ms to bind
		time.Sleep(200 * time.Millisecond)
	} else {
		log.Printf("[bridge] ciadpi binary (%s) not found in PATH, assuming byedpi is managed externally at %s", ciadpiBin, byedpiAddr)
	}

	bridge := &ProxyBridge{
		doh:       newDoHResolver(),
		socksAddr: byedpiAddr,
	}

	// Warm up DoH cache for common Discord domains
	go func() {
		warmupDomains := []string{"discord.com", "gateway.discord.gg", "cdn.discordapp.com"}
		for _, d := range warmupDomains {
			wCtx, wCancel := context.WithTimeout(context.Background(), 4*time.Second)
			ip, err := bridge.doh.Resolve(wCtx, d)
			wCancel()
			if err == nil {
				log.Printf("[doh] cached %s -> %s", d, ip)
			}
		}
	}()

	// Start SOCKS5 listener
	socksListener, err := net.Listen("tcp", socksListenAddr)
	if err != nil {
		log.Fatalf("[socks5] listen error: %v", err)
	}
	defer socksListener.Close()

	go func() {
		log.Printf("[socks5] listening on %s (forwarding to byedpi at %s)", socksListenAddr, byedpiAddr)
		for {
			conn, err := socksListener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go bridge.handleSOCKS5(conn)
		}
	}()

	// Start HTTP CONNECT listener
	httpServer := &http.Server{
		Addr:    httpListenAddr,
		Handler: http.HandlerFunc(bridge.handleHTTP),
	}

	go func() {
		log.Printf("[http] listening on %s (CONNECT proxy + /health)", httpListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[http] server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("[bridge] shutting down gracefully...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
	log.Println("[bridge] stopped")
}
