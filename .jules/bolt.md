## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.

## 2025-02-12 - Go `math.Pow` vs bitwise shift
**Learning:** In Go, using `math.Pow(2, float64(n))` for power-of-2 integer calculations introduces unnecessary floating-point overhead and type conversions.
**Action:** Use bitwise left shifts (`1 << n`) instead to improve performance and reduce overhead.

## 2024-05-19 - Go JSON Unmarshal allocation optimization
**Learning:** In Go, string operations like `strings.Trim(string(b), "\"")` during JSON unmarshaling cause unnecessary heap allocations. Likewise, string concatenation and casting to `[]byte` in `MarshalJSON` causes multiple heap allocations.
**Action:** Use manual byte slice slicing (e.g., `b[1 : len(b)-1]`) to strip quotes and use `strconv.AppendInt` on a pre-allocated `[]byte` buffer to significantly reduce allocations and improve performance in frequently-called JSON marshaling/unmarshaling code.

## 2025-02-13 - Go JSON zero-allocation marshal/unmarshal
**Learning:** Returning `[]byte("0")` or allocating a new `[]byte` slice for identical JSON null/zero representations like `[]byte("null")` creates unnecessary heap allocations on every struct serialization. Furthermore, `string(b)` casts allocate, while `string(b) == "null"` is optimized by the compiler to not allocate.
**Action:** Use `unsafe.String(unsafe.SliceData(b), len(b))` for parsing bytes as ints, compare `string(b) == "null"` directly, and use a package-level `var zeroBytes = []byte("0")` instead of inline slice declarations to reduce JSON zero-value and time unmarshaling/marshaling allocations to zero.
