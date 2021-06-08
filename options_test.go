package sc_test

import (
	"testing"

	"github.com/go-chassis/sc-client"
	"github.com/stretchr/testify/assert"
)

func TestWithGlobal(t *testing.T) {
	o := sc.WithGlobal()
	opts := &sc.CallOptions{}
	o(opts)
	assert.True(t, opts.WithGlobal)
}
