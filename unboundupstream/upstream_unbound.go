// +build agh_mod_unbound

package unboundupstream

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/miekg/unbound"
)

// Context for an upstream which uses libunbound
type UnboundUpstream struct {
	ctx *unbound.Unbound
}

// New creates a new libunbound context
// The list of supported settings: https://nlnetlabs.nl/documentation/unbound/unbound.conf/
func New(settings []string) (*UnboundUpstream, error) {
	u := UnboundUpstream{}
	u.ctx = unbound.New()
	for _, s := range settings {
		keyAndVal := strings.SplitAfterN(s, ":", 2)
		err := u.ctx.SetOption(keyAndVal[0], strings.TrimSpace(keyAndVal[1]))
		if err != nil {
			return nil, fmt.Errorf("set option '%s': %s", s, err)
		}
	}
	return &u, nil
}

// Close destroys libunbound context
func (u *UnboundUpstream) Close() {
	u.ctx.Destroy()
}

// Address overrides Upstream interface
func (u *UnboundUpstream) Address() string {
	return "[unbound]"
}

// Exchange overrides Upstream interface
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
