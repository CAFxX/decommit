# decommit

Temporarily decommit the memory that backs allocated byte slices.

This is useful in some limited scenarios where byte slices are long-lived or reused, e.g. when passing a reused byte slice to a read operation that can block for a long time (like when reading from a network connection).

The contents of a slice that has been decommitted are undeterminate: they could contain zeros, the original data, or other arbitrary data. Once the contents of the decommitted slice are accessed, either for reading or writing, the slice is transparently committed in memory again. Do not call `decommit.Slice` if you care about the current contents of the slice!

## Usage

```golang
// Assume that buf is a slice larger than a page (normally 4KB, but OS dependent:
// see os.Getpagesize())
decommit.Slice(buf)
// If the OS supports it, the memory backing the slice has been decommitted, and
// it will remain so until the slice contents (that are no undetermined) are
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
- Decommitting a slice normally requires performing a syscall.
- Decommitting is performed via `madvise(MADV_DONTNEED)` on linux/mac/bsd and `DiscardVirtualMemory` on windows, but this may change in the future.
- It does not make sense to decommit memory of a newly-allocated slice because newly-allocated slices are normally already not committed (until accessed for read/write).

## License

[MIT](LICENSE)
