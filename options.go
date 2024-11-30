package sc

import (
	"crypto/tls"
	"github.com/go-chassis/cari/rbac"
	"time"

	"context"
)

// Options is the list of dynamic parameter's which can be passed to the Client while creating a new client
type Options struct {
	Endpoints []string
	EnableSSL bool
	Timeout   time.Duration
	TLSConfig *tls.Config
	// Other options can be stored in a context
	Context         context.Context
	Compressed      bool
	Verbose         bool
	EnableAuth      bool
	AuthUser        *rbac.AuthUser
	TokenExpiration time.Duration
}

//CallOptions is options when you call a API
type CallOptions struct {
	WithoutRevision bool
	Revision        string
	WithGlobal      bool
	Address         string
}

//WithoutRevision ignore current revision number
func WithoutRevision() CallOption {
	return func(o *CallOptions) {
		o.WithoutRevision = true
	}
}

//WithGlobal query resources include other aggregated SC
func WithGlobal() CallOption {
	return func(o *CallOptions) {
		o.WithGlobal = true
	}
}

//WithAddress query resources with the sc address
func WithAddress(address string) CallOption {
	return func(o *CallOptions) {
		o.Address = address
	}
}

//CallOption is receiver for options and chang the attribute of it
type CallOption func(*CallOptions)
