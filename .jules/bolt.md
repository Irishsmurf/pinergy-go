## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.

## 2025-02-12 - Go `math.Pow` vs bitwise shift
**Learning:** In Go, using `math.Pow(2, float64(n))` for power-of-2 integer calculations introduces unnecessary floating-point overhead and type conversions.
**Action:** Use bitwise left shifts (`1 << n`) instead to improve performance and reduce overhead.

## 2024-05-19 - Go JSON Unmarshal allocation optimization
**Learning:** In Go, string operations like `strings.Trim(string(b), "\"")` during JSON unmarshaling cause unnecessary heap allocations. Likewise, string concatenation and casting to `[]byte` in `MarshalJSON` causes multiple heap allocations.
**Action:** Use manual byte slice slicing (e.g., `b[1 : len(b)-1]`) to strip quotes and use `strconv.AppendInt` on a pre-allocated `[]byte` buffer to significantly reduce allocations and improve performance in frequently-called JSON marshaling/unmarshaling code.

## $(date +%Y-%m-%d) - Zero-Allocation JSON Methods
**Learning:** `bytes.Equal(b, []byte("null"))` allocates memory for the `[]byte("null")` literal. In Go 1.20+, checking against a constant string like `string(b) == "null"` is recognized by the compiler and prevents allocations. For JSON marshaling, returning an inline literal `[]byte("0")` also triggers a heap allocation on each call, which can be avoided by returning a package-level `var zeroBytes = []byte("0")`.
**Action:** Always prefer compiler-optimized string comparisons for small constants in `UnmarshalJSON` and use package-level variables for static `[]byte` returns in `MarshalJSON` to avoid unnecessary allocations in hot paths.
