package decommit

import (
	"os"
	"testing"
)

func BenchmarkSlice(b *testing.B) {
	ps := os.Getpagesize()
	buf := make([]byte, 2*ps)
	for i := 0; i < b.N; i++ {
		buf[0] = 255
		buf[ps] = 255
		Slice(buf)
	}
}
