//go:build testing

package decommit

import (
	"os"
	"strconv"
	"sync"
	"testing"

	sigar "github.com/cloudfoundry/gosigar"
)

const isTesting = true

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

func TestNilSlice(t *testing.T) {
	setDecommitHook(t)
	if Slice(nil) != 0 {
		t.Error("nil slice decommitted!")
	}
}

func TestPageAlign(t *testing.T) {
	oldps, oldpsm := ps, psm
	t.Cleanup(func() {
		ps, psm = oldps, oldpsm
	})

	ps4, psm4 := uintptr(1<<12), uintptr((1<<12)-1)
	cases := []struct {
		ps      uintptr
		psm     uintptr
		start   uintptr
		end     uintptr
		astart  uintptr
		alength int
	}{
		{ps4, psm4, 5 * ps4, 6 * ps4, 5 * ps4, int(ps4)},
		{ps4, psm4, 5 * ps4, 6*ps4 - 1, 0, 0},
		{ps4, psm4, 5 * ps4, 6*ps4 + 1, 5 * ps4, int(ps4)},
		{ps4, psm4, 5*ps4 - 1, 6 * ps4, 5 * ps4, int(ps4)},
		{ps4, psm4, 5*ps4 + 1, 6 * ps4, 0, 0},
		{ps4, psm4, 5*ps4 - 1, 6*ps4 - 1, 0, 0},
		{ps4, psm4, 5*ps4 + 1, 6*ps4 - 1, 0, 0},
		{ps4, psm4, 5*ps4 - 1, 6*ps4 + 1, 5 * ps4, int(ps4)},
		{ps4, psm4, 5*ps4 + 1, 6*ps4 + 1, 0, 0},

		{ps4, psm4, 5 * ps4, 7 * ps4, 5 * ps4, int(2 * ps4)},
		{ps4, psm4, 5*ps4 + 1, 7 * ps4, 6 * ps4, int(ps4)},
		{ps4, psm4, 5 * ps4, 7*ps4 - 1, 5 * ps4, int(ps4)},
		{ps4, psm4, 5*ps4 + 1, 7*ps4 - 1, 0, 0},

		{ps4, psm4, 5 * ps4, 8 * ps4, 5 * ps4, int(3 * ps4)},
		{ps4, psm4, 5*ps4 + 1, 8 * ps4, 6 * ps4, int(2 * ps4)},
		{ps4, psm4, 5 * ps4, 8*ps4 - 1, 5 * ps4, int(2 * ps4)},
		{ps4, psm4, 5*ps4 + 1, 8*ps4 - 1, 6 * ps4, int(ps4)},

		{ps4, psm4, 6 * ps4, 5 * ps4, 0, 0},
	}
	for idx, c := range cases {
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			t.Log(c)
			ps, psm = c.ps, c.psm
			astart, _, alength := pageAlign(c.start, c.end)
			if astart != c.astart {
				t.Error("astart", astart)
			}
			if alength != c.alength {
				t.Error("alength", alength)
			}
		})
	}
}

func TestLottaAllocs(t *testing.T) {
	mem := sigar.Mem{}
	mem.Get()
	swap := sigar.Swap{}
	swap.Get()

	mb := int(((mem.Total + swap.Total) * 2) / (1024 * 1024))

	t.Logf("allocating %d MB", mb)

	var holder [][]byte
	for i := 0; i < mb; i++ {
		buf := make([]byte, 1024*1024)
		for k := range buf {
			buf[k] = 255
		}
		Slice(buf)
		holder = append(holder, buf)
	}

	_ = holder // if we get here without getting killed, the test was a success
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
	t.Cleanup(func() {
		decommitHook = oldDecommitHook
	})

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
}
