package home

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
)

const (
	rdnsTimeout = 3 * time.Second // max time to wait for rDNS response
)

type dnsContext struct {
	rdnsChannel chan string // pass data from DNS request handling thread to rDNS thread
	// contains IP addresses of clients to be resolved by rDNS
	// if IP address couldn't be resolved, it stays here forever to prevent further attempts to resolve the same IP
	rdnsIP   map[string]bool
	rdnsLock sync.Mutex        // synchronize access to rdnsIP
	upstream upstream.Upstream // Upstream object for our own DNS server
}

// initDNSServer creates an instance of the dnsforward.Server
// Please note that we must do it even if we don't start it
// so that we had access to the query log and the stats
func initDNSServer(baseDir string) {
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		log.Fatalf("Cannot create DNS data dir at %s: %s", baseDir, err)
	}

	config.dnsServer = dnsforward.NewServer(baseDir, false)

	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	resolverAddress := fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)
	opts := upstream.Options{
		Timeout: rdnsTimeout,
	}
	config.dnsctx.upstream, err = upstream.AddressToUpstream(resolverAddress, opts)
	if err != nil {
		log.Error("upstream.AddressToUpstream: %s", err)
		return
	}
	config.dnsctx.rdnsIP = make(map[string]bool)
	config.dnsctx.rdnsChannel = make(chan string, 256)
	go asyncRDNSLoop()
	go func() {
		top := config.dnsServer.GetStatsTop(int(config.DNS.QueryLogInterval) * 24)
		for k := range top.Clients {
			beginAsyncRDNS(k)
		}
	}()
}

func isRunning() bool {
	return config.dnsServer != nil && config.dnsServer.IsRunning()
}

func beginAsyncRDNS(ip string) {
	if config.clients.Exists(ip) {
		return
	}

	// add IP to rdnsIP, if not exists
	config.dnsctx.rdnsLock.Lock()
	defer config.dnsctx.rdnsLock.Unlock()
	_, ok := config.dnsctx.rdnsIP[ip]
	if ok {
		return
	}
	config.dnsctx.rdnsIP[ip] = true

	log.Tracef("Adding %s for rDNS resolve", ip)
	select {
	case config.dnsctx.rdnsChannel <- ip:
		//
	default:
		log.Tracef("rDNS queue is full")
	}
}

// Use rDNS to get hostname by IP address
func resolveRDNS(ip string) string {
	log.Tracef("Resolving host for %s", ip)

	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{
			Qtype:  dns.TypePTR,
			Qclass: dns.ClassINET,
		},
	}
	var err error
	req.Question[0].Name, err = dns.ReverseAddr(ip)
	if err != nil {
		log.Debug("Error while calling dns.ReverseAddr(%s): %s", ip, err)
		return ""
	}

	resp, err := config.dnsctx.upstream.Exchange(&req)
	if err != nil {
		log.Error("Error while making an rDNS lookup for %s: %s", ip, err)
		return ""
	}
	if len(resp.Answer) != 1 {
		log.Debug("No answer for rDNS lookup of %s", ip)
		return ""
	}
	ptr, ok := resp.Answer[0].(*dns.PTR)
	if !ok {
		log.Error("not a PTR response for %s", ip)
		return ""
	}

	log.Tracef("PTR response for %s: %s", ip, ptr.String())
	if strings.HasSuffix(ptr.Ptr, ".") {
		ptr.Ptr = ptr.Ptr[:len(ptr.Ptr)-1]
	}

	return ptr.Ptr
}

// Wait for a signal and then synchronously resolve hostname by IP address
// Add the hostname:IP pair to "Clients" array
func asyncRDNSLoop() {
	for {
		var ip string
		ip = <-config.dnsctx.rdnsChannel

		host := resolveRDNS(ip)
		if len(host) == 0 {
			continue
		}

		config.dnsctx.rdnsLock.Lock()
		delete(config.dnsctx.rdnsIP, ip)
		config.dnsctx.rdnsLock.Unlock()

		_, _ = config.clients.AddHost(ip, host, ClientSourceRDNS)
	}
}

func onDNSRequest(d *proxy.DNSContext) {
	qType := d.Req.Question[0].Qtype
	if qType != dns.TypeA && qType != dns.TypeAAAA {
		return
	}

	ip := dnsforward.GetIPString(d.Addr)
	if ip == "" {
		// This would be quite weird if we get here
		return
	}

	ipAddr := net.ParseIP(ip)
	if !ipAddr.IsLoopback() {
		beginAsyncRDNS(ip)
	}
}

func generateServerConfig() (dnsforward.ServerConfig, error) {
	filters := []dnsfilter.Filter{}
	userFilter := userFilter()
	filters = append(filters, dnsfilter.Filter{
		ID:   userFilter.ID,
		Data: userFilter.Data,
	})
	for _, filter := range config.Filters {
		if !filter.Enabled {
			continue
		}
		filters = append(filters, dnsfilter.Filter{
			ID:       filter.ID,
			FilePath: filter.Path(),
		})
	}

	newconfig := dnsforward.ServerConfig{
		UDPListenAddr:   &net.UDPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		TCPListenAddr:   &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.DNS.Port},
		FilteringConfig: config.DNS.FilteringConfig,
		Filters:         filters,
	}
	bindhost := config.DNS.BindHost
	if config.DNS.BindHost == "0.0.0.0" {
		bindhost = "127.0.0.1"
	}
	newconfig.ResolverAddress = fmt.Sprintf("%s:%d", bindhost, config.DNS.Port)

	if config.TLS.Enabled {
		newconfig.TLSConfig = config.TLS.TLSConfig
		if config.TLS.PortDNSOverTLS != 0 {
			newconfig.TLSListenAddr = &net.TCPAddr{IP: net.ParseIP(config.DNS.BindHost), Port: config.TLS.PortDNSOverTLS}
		}
	}

	upstreamConfig, err := proxy.ParseUpstreamsConfig(config.DNS.UpstreamDNS, config.DNS.BootstrapDNS, dnsforward.DefaultTimeout)
	if err != nil {
		return newconfig, fmt.Errorf("Couldn't get upstreams configuration cause: %s", err)
	}
	newconfig.Upstreams = upstreamConfig.Upstreams
	newconfig.DomainsReservedUpstreams = upstreamConfig.DomainReservedUpstreams
	newconfig.AllServers = config.DNS.AllServers
	newconfig.FilterHandler = applyClientSettings
	newconfig.OnDNSRequest = onDNSRequest
	return newconfig, nil
}

// If a client has his own settings, apply them
func applyClientSettings(clientAddr string, setts *dnsfilter.RequestFilteringSettings) {
	c, ok := config.clients.Find(clientAddr)
	if !ok || !c.UseOwnSettings {
		return
	}

	log.Debug("Using settings for client with IP %s", clientAddr)
	setts.FilteringEnabled = c.FilteringEnabled
	setts.SafeSearchEnabled = c.SafeSearchEnabled
	setts.SafeBrowsingEnabled = c.SafeBrowsingEnabled
	setts.ParentalEnabled = c.ParentalEnabled
}

func startDNSServer() error {
	if isRunning() {
		return fmt.Errorf("unable to start forwarding DNS server: Already running")
	}

	newconfig, err := generateServerConfig()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}
	err = config.dnsServer.Start(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}

func reconfigureDNSServer() error {
	newconfig, err := generateServerConfig()
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}
	err = config.dnsServer.Reconfigure(&newconfig)
	if err != nil {
		return errorx.Decorate(err, "Couldn't start forwarding DNS server")
	}

	return nil
}

func stopDNSServer() error {
	if !isRunning() {
		return nil
	}

	err := config.dnsServer.Stop()
	if err != nil {
		return errorx.Decorate(err, "Couldn't stop forwarding DNS server")
	}

	return nil
}
