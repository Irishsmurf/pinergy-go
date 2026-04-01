package pinergy

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnixTime_Unmarshal(t *testing.T) {
	tests := []struct {
		json     string
		wantSec  int64
		wantZero bool
	}{
		{`"1773446400"`, 1773446400, false},
		{`"null"`, 0, true},
		{`""`, 0, true},
	}
	for _, tt := range tests {
		var u UnixTime
		if err := json.Unmarshal([]byte(tt.json), &u); err != nil {
			t.Errorf("Unmarshal(%s): %v", tt.json, err)
			continue
		}
		if tt.wantZero {
			if !u.IsZero() {
				t.Errorf("Unmarshal(%s): expected zero time", tt.json)
			}
		} else {
			got := u.Unix()
			if got != tt.wantSec {
				t.Errorf("Unmarshal(%s): Unix() = %d, want %d", tt.json, got, tt.wantSec)
			}
		}
	}
}

func TestUnixTime_Marshal(t *testing.T) {
	u := UnixTime{Time: time.Unix(1773446400, 0).UTC()}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `"1773446400"`
	if string(b) != want {
		t.Errorf("Marshal = %s, want %s", b, want)
	}
}

func TestUnixTime_RoundTrip(t *testing.T) {
	original := UnixTime{Time: time.Unix(1773446400, 0).UTC()}
	b, _ := json.Marshal(original)
	var restored UnixTime
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !original.Equal(restored.Time) {
		t.Errorf("round-trip: got %v, want %v", restored.Time, original.Time)
	}
}

func TestUnixTime_UnmarshalInvalid(t *testing.T) {
	var u UnixTime
	if err := json.Unmarshal([]byte(`"not-a-number"`), &u); err == nil {
		t.Error("expected error for non-numeric timestamp")
	}
}

func BenchmarkUnixTime_UnmarshalJSON(b *testing.B) {
	data := []byte(`"1773446400"`)
	var u UnixTime
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = u.UnmarshalJSON(data)
	}
}

func BenchmarkUnixTime_UnmarshalJSON_Null(b *testing.B) {
	data := []byte(`null`)
	var u UnixTime
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = u.UnmarshalJSON(data)
	}
}

func BenchmarkUnixTime_MarshalJSON(b *testing.B) {
	u := UnixTime{Time: time.Unix(1773446400, 0)}
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
