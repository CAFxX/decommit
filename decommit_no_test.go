//go:build !testing

package decommit

import "testing"

func TestDecommit(t *testing.T) {
	t.Fatal("testing build tag not set; use `go test -tags testing`")
}
