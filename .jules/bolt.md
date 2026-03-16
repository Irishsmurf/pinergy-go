## 2024-03-16 - Go `sha1` hashing allocation
**Learning:** In Go, using `sha1.New()`, `.Write()`, and `fmt.Sprintf("%x")` for hashing causes extra heap allocations for the hash state and string formatting overhead.
**Action:** Use `sha1.Sum()` directly (allocates on stack) and `hex.EncodeToString` instead to reduce allocations by ~33% and speed up the hash operation by ~60%.
