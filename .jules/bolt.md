## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.

## 2025-02-12 - Go `math.Pow` vs bitwise shift
**Learning:** In Go, using `math.Pow(2, float64(n))` for power-of-2 integer calculations introduces unnecessary floating-point overhead and type conversions.
**Action:** Use bitwise left shifts (`1 << n`) instead to improve performance and reduce overhead.

## 2024-05-19 - Go JSON Unmarshal allocation optimization
**Learning:** In Go, string operations like `strings.Trim(string(b), "\"")` during JSON unmarshaling cause unnecessary heap allocations. Likewise, string concatenation and casting to `[]byte` in `MarshalJSON` causes multiple heap allocations.
**Action:** Use manual byte slice slicing (e.g., `b[1 : len(b)-1]`) to strip quotes and use `strconv.AppendInt` on a pre-allocated `[]byte` buffer to significantly reduce allocations and improve performance in frequently-called JSON marshaling/unmarshaling code.

## 2024-05-10 - Replace fmt.Sprintf with strconv.Itoa
**Learning:** In Go, `fmt.Sprintf` uses reflection and allocates memory, which is slow for simple string formatting in error paths or performance-critical code.
**Action:** Use `strconv.Itoa` and string concatenation instead of `fmt.Sprintf` to avoid reflection overhead and reduce memory allocations for simple integer-to-string formatting.

## 2024-05-10 - Go MarshalJSON zero value slice allocation
**Learning:** Returning inline slice declarations like `[]byte("0")` from a custom `MarshalJSON` method causes heap allocations on every call, even for zero values. Standard library `encoding/json` does not mutate returned slices, so it is safe to return a global.
**Action:** Return a pre-allocated package-level variable (e.g., `var zeroTime = []byte("\"0\"")`) instead of an inline slice for zero values in `MarshalJSON` to eliminate allocations. Add a comment warning that the returned slice is mutable.
