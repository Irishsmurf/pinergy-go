## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.

## 2025-02-12 - Go `math.Pow` vs bitwise shift
**Learning:** In Go, using `math.Pow(2, float64(n))` for power-of-2 integer calculations introduces unnecessary floating-point overhead and type conversions.
**Action:** Use bitwise left shifts (`1 << n`) instead to improve performance and reduce overhead.

## 2024-05-19 - Go JSON Unmarshal allocation optimization
**Learning:** In Go, string operations like `strings.Trim(string(b), "\"")` during JSON unmarshaling cause unnecessary heap allocations. Likewise, string concatenation and casting to `[]byte` in `MarshalJSON` causes multiple heap allocations.
**Action:** Use manual byte slice slicing (e.g., `b[1 : len(b)-1]`) to strip quotes and use `strconv.AppendInt` on a pre-allocated `[]byte` buffer to significantly reduce allocations and improve performance in frequently-called JSON marshaling/unmarshaling code.

## 2025-02-13 - Go `string` allocation optimizations in parsing
**Learning:** In Go, converting `[]byte` to `string` usually allocates, except when doing a comparison with a constant string (e.g. `string(b) == "null"`). Also, `strconv.ParseInt(string(b), 10, 64)` allocates string on each call.
**Action:** Use `string(b) == "null"` instead of `bytes.Equal(b, []byte("null"))` for zero-allocation checks, and use `unsafe.String(unsafe.SliceData(b), len(b))` to eliminate heap allocation when passing a `[]byte` slice directly to string-expecting functions like `strconv.ParseInt`.
