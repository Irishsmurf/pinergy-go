## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.

## 2025-02-12 - Go `math.Pow` vs bitwise shift
**Learning:** In Go, using `math.Pow(2, float64(n))` for power-of-2 integer calculations introduces unnecessary floating-point overhead and type conversions.
**Action:** Use bitwise left shifts (`1 << n`) instead to improve performance and reduce overhead.

## 2024-05-19 - Go JSON Unmarshal allocation optimization
**Learning:** In Go, string operations like `strings.Trim(string(b), "\"")` during JSON unmarshaling cause unnecessary heap allocations. Likewise, string concatenation and casting to `[]byte` in `MarshalJSON` causes multiple heap allocations.
**Action:** Use manual byte slice slicing (e.g., `b[1 : len(b)-1]`) to strip quotes and use `strconv.AppendInt` on a pre-allocated `[]byte` buffer to significantly reduce allocations and improve performance in frequently-called JSON marshaling/unmarshaling code.

## 2024-03-16 - Go `MarshalJSON` optimization for empty values
**Learning:** In Go, returning inline byte slices like `[]byte("0")` or `[]byte("\"0\"")` from interface methods like `MarshalJSON()` results in a heap allocation for every call. The compiler cannot optimize this out because it cannot guarantee the caller will not modify the slice.
**Action:** Use pre-allocated package-level variables for static JSON responses (e.g. `var zeroUnixTimeJSON = []byte("\"0\"")`) and return those. Include a comment warning that the returned slice is mutable. This eliminates allocations entirely and makes the operation significantly faster.
