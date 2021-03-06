# decommit

Temporarily decommit the memory that backs allocated byte slices. 

[![Go Reference](https://pkg.go.dev/badge/github.com/CAFxX/decommit.svg)](https://pkg.go.dev/github.com/CAFxX/decommit)
![Build](https://github.com/CAFxX/decommit/actions/workflows/go.yml/badge.svg)

This is useful in some limited scenarios where byte slices are long-lived or reused, e.g. when passing a reused byte slice to a read operation that can block for a long time (like when reading from a network connection).

The contents of a slice that has been decommitted are undeterminate: they could contain zeros, the original data, or other arbitrary data. Once the contents of the decommitted slice are accessed, either for reading or writing, the slice is transparently committed in memory again. Do not call `decommit.Slice` if you care about the current contents of the slice!

## Usage

```golang
// Assume that buf is a slice larger than a page (normally 4KB, but OS dependent:
// see os.Getpagesize()) and that we do not need its contents anymore.
decommit.Slice(buf)
// If the OS supports it, the memory backing the slice should have been decommitted,
// and it will remain so until the slice contents (that are now undetermined) are
// accessed for reading or writing.
```

A common pattern is to use it together with `sync.Pool`:

```golang
type buffer [16*1024]byte

var pool = sync.Pool{
    New: func() interface{} {
        return &buffer{}
    },
}

func getBuffer() *buffer {
    return pool.Get().(*buffer)
}

func putBuffer(buf *buffer) {
    decommit.Slice(buf[:])
    pool.Put(buf)
}
```

## Notes 

- This operation is transparent to the Go runtime, it only affects the OS. As a result, it can **not** be detected via `runtime.MemStats` or other runtime-provided mechanisms.
- This operation is best-effort. It requests the OS to eagerly decommit memory, but there is no guarantee that the OS will effectively do it (or when it will do it).
- Most operating systems place restrictions on the granularity of the decommit operation: most commonly they require that only whole pages are decommitted (i.e. that the start of the range is pagesize-aligned, and that the length of the range is a multiple of the pagesize). The functions in this package will perform the required alignment operations to decommit as much memory as possible.
- Decommitting a slice normally requires performing a syscall.
- Decommitting is performed via `madvise(MADV_DONTNEED)` on linux/mac/bsd and `DiscardVirtualMemory` on windows, but this may change in the future.
- It does not make sense to decommit memory of a newly-allocated slice because newly-allocated slices are normally already not committed (until accessed for read/write).
- It is recommended to allocate the slices with a power-of-2 capacity greater or equal than 4KB. As of go 1.17, the Go runtime always returns [at least 4KB-aligned allocations in this case](https://play.golang.org/p/qWDJ8YOTNNL).
- The whole memory used by the slice is affected by the call, i.e. `s[0:cap(s)]`; if needed you can limit the affected range by reducing the capacity of the slice e.g. with `decommit.Slice(s[:n:n])`.
- For safety, this approach can only be applied to memory that does not contain pointers/references. So e.g. a slice of pointers, strings, maps, channels or structs containing pointer/references can not be decommitted safely (as the GC will need to scan that memory, with the result of undoing the decommit operation, and may interpret garbage contained in the recommitted memory as valid pointers/references, potentially leaking memory or causing other misbehaviors). 

## License

[MIT](LICENSE)
