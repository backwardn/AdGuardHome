// +build !agh_mod_unbound

package dnsforward

import "github.com/miekg/dns"

type unboundUpstream struct {
}

func unboundUpstreamNew() *unboundUpstream {
	return nil
}

func unboundUpstreamClose(u *unboundUpstream) {
}

func (u *unboundUpstream) Address() string {
	return ""
}

func (u *unboundUpstream) Exchange(m *dns.Msg) (*dns.Msg, error) {
	return nil, nil
}
