package decommit

import (
	"os"
	"strconv"
	"sync"
	"testing"
)

func TestSlice(t *testing.T) {
	ps := os.Getpagesize()
	for _, i := range []int{0, 1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, ps - 1, ps, ps*2 - 1, ps * 2, ps*3 - 1, ps * 3} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			setDecommitHook(t)
			b := make([]byte, i)
			l := Slice(b)
			if i-(2*ps) > 0 && l <= i-(2*ps) {
				t.Errorf("decommitted length too low %x", l)
			}
		})
	}
}

func ExamplePool() {
	type buffer [16 * 1024]byte

	var pool = sync.Pool{
		New: func() interface{} {
			return &buffer{}
		},
	}

	getBuffer := func() *buffer {
		return pool.Get().(*buffer)
	}

	putBuffer := func(buf *buffer) {
		Slice(buf[:])
		pool.Put(buf)
	}

	buf := getBuffer()

	// use buf...

	putBuffer(buf)
}

func setDecommitHook(t *testing.T) {
	oldDecommitHook := decommitHook
	decommitHook = func(_ uintptr, _ uintptr, astart uintptr, aend uintptr, alength int) (uintptr, int) {
		if astart%uintptr(os.Getpagesize()) != 0 {
			t.Errorf("unaligned start %x", astart)
		}
		if aend%uintptr(os.Getpagesize()) != 0 {
			t.Errorf("unaligned end %x", aend)
		}
		if alength%os.Getpagesize() != 0 {
			t.Errorf("unaligned length %x", alength)
		}
		return astart, alength
	}
	t.Cleanup(func() {
		decommitHook = oldDecommitHook
	})
}
