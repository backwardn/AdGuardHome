// +build agh_mod_unbound

package unboundupstream

import (
	"github.com/miekg/dns"
	"github.com/miekg/unbound"
)

type UnboundUpstream struct {
	ctx *unbound.Unbound
}

func New() *UnboundUpstream {
	u := UnboundUpstream{}
	u.ctx = unbound.New()
	return &u
}

func (u *UnboundUpstream) Close() {
	u.ctx.Destroy()
}

func (u *UnboundUpstream) Address() string {
	return "[unbound]"
}

func (u *UnboundUpstream) Exchange(m *dns.Msg) (*dns.Msg, error) {
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
