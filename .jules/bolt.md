## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.

## 2025-02-12 - Go `math.Pow` vs bitwise shift
**Learning:** In Go, using `math.Pow(2, float64(n))` for power-of-2 integer calculations introduces unnecessary floating-point overhead and type conversions.
**Action:** Use bitwise left shifts (`1 << n`) instead to improve performance and reduce overhead.

## 2024-05-19 - Go JSON Unmarshal allocation optimization
**Learning:** In Go, string operations like `strings.Trim(string(b), "\"")` during JSON unmarshaling cause unnecessary heap allocations. Likewise, string concatenation and casting to `[]byte` in `MarshalJSON` causes multiple heap allocations.
**Action:** Use manual byte slice slicing (e.g., `b[1 : len(b)-1]`) to strip quotes and use `strconv.AppendInt` on a pre-allocated `[]byte` buffer to significantly reduce allocations and improve performance in frequently-called JSON marshaling/unmarshaling code.

## 2024-05-18 - [Optimizing JSON Unmarshaling of Timestamps]
**Learning:** `bytes.Equal(b, []byte("null"))` allocates a new byte slice on every call in Go when comparing to constant strings. The Go compiler specifically optimizes `string(b) == "null"` to avoid this allocation. Further, when passing a `[]byte` to a string-expecting function like `strconv.ParseInt()`, converting `string(b)` will cause a heap allocation unless `unsafe.String(unsafe.SliceData(b), len(b))` is used. This allocation bottleneck can be a common pitfall in high-throughput JSON processing like unmarshaling repetitive timestamps.
**Action:** Replace `bytes.Equal(b, []byte("constant"))` with `string(b) == "constant"`, and utilize `unsafe.String` when converting byte slices strictly for passing to string-parsing stdlib functions.
