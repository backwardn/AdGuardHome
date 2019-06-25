// +build agh_mod_unbound

package unboundupstream

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
	"github.com/miekg/unbound"
)

type UnboundUpstream struct {
	ctx *unbound.Unbound
}

func New() *UnboundUpstream {
	u := UnboundUpstream{}
	u.ctx = unbound.New()
	e := u.ctx.AddTaFile("./keys") // "dig . DNSKEY >keys"
	if e != nil {
		log.Fatal(e)
		u.Close()
		return nil
	}
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

	if r.Bogus {
		// Bogus (security failed) can have many reasons, DNSSEC protects against alteration of the data in transit, signatures can expire, the trusted keys can be rolled over to fresh trusted keys, and many others
		return nil, fmt.Errorf("Bogus: %s", r.WhyBogus)
	}

	if !r.Secure {
		// Insecure happens when no DNSSEC security is configured for the domain name (or you simply forgot to add the trusted key)
		return nil, fmt.Errorf("Insecure")
	}

	// Secure means that one of the trusted keys verifies the signatures on the data

	resp := dns.Msg{}
	resp.SetRcode(m, r.Rcode)
	resp.Answer = r.AnswerPacket.Answer
	resp.Ns = r.AnswerPacket.Ns
	resp.Extra = r.AnswerPacket.Extra
	return &resp, nil
}
