// +build !agh_mod_unbound

package unboundupstream

import "github.com/miekg/dns"

type UnboundUpstream struct {
}

func New() *UnboundUpstream {
	return nil
}

func (u *UnboundUpstream) Close() {
}

func (u *UnboundUpstream) Address() string {
	return ""
}

func (u *UnboundUpstream) Exchange(m *dns.Msg) (*dns.Msg, error) {
	return nil, nil
}
