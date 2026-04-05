package pinergy

import (
	"testing"
	"time"
)

func BenchmarkUnixTime_MarshalJSON_Zero(b *testing.B) {
	u := UnixTime{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = u.MarshalJSON()
	}
}

func BenchmarkUnixTime_MarshalJSON_NonZero(b *testing.B) {
	u := UnixTime{Time: time.Now()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = u.MarshalJSON()
	}
}
