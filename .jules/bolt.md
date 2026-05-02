## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.

## 2025-02-12 - Go `math.Pow` vs bitwise shift
**Learning:** In Go, using `math.Pow(2, float64(n))` for power-of-2 integer calculations introduces unnecessary floating-point overhead and type conversions.
**Action:** Use bitwise left shifts (`1 << n`) instead to improve performance and reduce overhead.

## 2024-05-19 - Go JSON Unmarshal allocation optimization
**Learning:** In Go, string operations like `strings.Trim(string(b), "\"")` during JSON unmarshaling cause unnecessary heap allocations. Likewise, string concatenation and casting to `[]byte` in `MarshalJSON` causes multiple heap allocations.
**Action:** Use manual byte slice slicing (e.g., `b[1 : len(b)-1]`) to strip quotes and use `strconv.AppendInt` on a pre-allocated `[]byte` buffer to significantly reduce allocations and improve performance in frequently-called JSON marshaling/unmarshaling code.
## 2024-05-02 - Go `[]byte` slice allocation in JSON Marshaling
**Learning:** In Go, returning an inline slice declaration like `[]byte("0")` from a function (such as `MarshalJSON`) allocates on the heap every single time it's called, even though the string literal is constant.
**Action:** For constant byte slice returns, use a package-level variable instead (e.g. `var zeroTimeJSON = []byte("\"0\"")`) to eliminate all allocations and improve performance (reduces allocations from 1 to 0 and runtime from ~18ns to ~3.8ns). Ensure a comment is added warning that the returned slice is mutable.
