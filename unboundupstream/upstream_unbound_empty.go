// +build !agh_mod_unbound

package unboundupstream

import (
	"fmt"

	"github.com/miekg/dns"
)

type UnboundUpstream struct {
}

// New always returns an error
func New(settings []string) (*UnboundUpstream, error) {
	return nil, fmt.Errorf("AdGuardHome isn't compiled with libunbound support")
}

// Close does nothing
func (u *UnboundUpstream) Close() {
}

// Address overrides Upstream interface
func (u *UnboundUpstream) Address() string {
	return ""
}

// Exchange overrides Upstream interface
func (u *UnboundUpstream) Exchange(m *dns.Msg) (*dns.Msg, error) {
	return nil, nil
}
