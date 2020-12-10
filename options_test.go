package sc_test

import (
	"github.com/go-chassis/sc-client"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithGlobal(t *testing.T) {
	o := sc.WithGlobal()
	opts := &sc.CallOptions{}
	o(opts)
	assert.True(t, opts.WithGlobal)
}
