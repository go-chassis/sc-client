package client

import (
	"crypto/tls"
	"time"

	"golang.org/x/net/context"
)

// Options is the list of dynamic parameter's which can be passed to the RegistryClient while creating a new client
type Options struct {
	Addrs        []string
	EnableSSL    bool
	ConfigTenant string
	Timeout      time.Duration
	TLSConfig    *tls.Config
	// Other options can be stored in a context
	Context    context.Context
	Compressed bool
	Verbose    bool
	Version    string
}
