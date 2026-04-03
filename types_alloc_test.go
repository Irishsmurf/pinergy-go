package pinergy

import (
	"testing"
	"time"
)

func BenchmarkUnixTime_UnmarshalJSON(b *testing.B) {
	input := []byte(`"1773446400"`)
	var u UnixTime
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = u.UnmarshalJSON(input)
	}
}

func BenchmarkUnixTime_MarshalJSON(b *testing.B) {
	u := UnixTime{Time: time.Unix(1773446400, 0)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = u.MarshalJSON()
	}
}

func BenchmarkUnixTime_MarshalJSONZero(b *testing.B) {
	var u UnixTime // zero time
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = u.MarshalJSON()
	}
}
