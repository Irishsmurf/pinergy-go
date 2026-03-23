package pinergy

import (
	"testing"
	"time"
)

func BenchmarkUnixTime_UnmarshalJSON(b *testing.B) {
	data := []byte(`"1773446400"`)
	var u UnixTime
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = u.UnmarshalJSON(data)
	}
}

func BenchmarkUnixTime_MarshalJSON(b *testing.B) {
	u := UnixTime{Time: time.Unix(1773446400, 0).UTC()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = u.MarshalJSON()
	}
}

func BenchmarkUnixTime_MarshalJSON_Zero(b *testing.B) {
	var u UnixTime
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = u.MarshalJSON()
	}
}
