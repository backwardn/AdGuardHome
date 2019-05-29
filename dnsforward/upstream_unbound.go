// +build agh_mod_unbound

package dnsforward

import (
	"github.com/miekg/dns"
	"github.com/miekg/unbound"
)

type unboundUpstream struct {
	ctx *unbound.Unbound
}

func unboundUpstreamNew() *unboundUpstream {
	u := unboundUpstream{}
	u.ctx = unbound.New()
	return &u
}

func unboundUpstreamClose(u *unboundUpstream) {
	u.ctx.Destroy()
}

func (u *unboundUpstream) Address() string {
	return "[unbound]"
}

func (u *unboundUpstream) Exchange(m *dns.Msg) (*dns.Msg, error) {
	r, err := u.ctx.Resolve(m.Question[0].Name, m.Question[0].Qtype, dns.ClassINET)
	if err != nil {
		return nil, err
	}

	resp := dns.Msg{}
	resp.SetRcode(m, r.Rcode)
	resp.Answer = r.AnswerPacket.Answer
	resp.Ns = r.AnswerPacket.Ns
	resp.Extra = r.AnswerPacket.Extra
	return &resp, nil
}
