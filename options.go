package sc

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/go-chassis/cari/rbac"
)

// Options is the list of dynamic parameter's which can be passed to the Client while creating a new client
type Options struct {
	DiffAzEndpoints []string
	Endpoints       []string
	EnableSSL       bool
	Timeout         time.Duration
	TLSConfig       *tls.Config
	// Other options can be stored in a context
	Context         context.Context
	Compressed      bool
	Verbose         bool
	EnableAuth      bool
	AuthUser        *rbac.AuthUser
	AuthToken       string
	TokenExpiration time.Duration
	SignRequest     func(*http.Request) error
}

// CallOptions is options when you call a API
type CallOptions struct {
	WithoutRevision bool
	Revision        string
	WithGlobal      bool
	Address         string
}

// WithoutRevision ignore current revision number
func WithoutRevision() CallOption {
	return func(o *CallOptions) {
		o.WithoutRevision = true
	}
}

// WithRevision query resources with the revision
func WithRevision(revision string) CallOption {
	return func(o *CallOptions) {
		o.Revision = revision
	}
}

// WithGlobal query resources include other aggregated SC
func WithGlobal() CallOption {
	return func(o *CallOptions) {
		o.WithGlobal = true
	}
}

// WithAddress query resources with the sc address
func WithAddress(address string) CallOption {
	return func(o *CallOptions) {
		o.Address = address
	}
}

// CallOption is receiver for options and chang the attribute of it
type CallOption func(*CallOptions)
