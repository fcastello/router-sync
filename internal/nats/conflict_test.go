package nats

import (
	"testing"
	"time"
)

func TestShouldAcceptWrite(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		existGen uint64
		existAt  time.Time
		existW   string
		inGen    uint64
		inAt     time.Time
		inW      string
		want     bool
	}{
		{
			name:     "higher generation wins",
			existGen: 1,
			existAt:  base,
			existW:   "r2",
			inGen:    2,
			inAt:     base.Add(-time.Hour),
			inW:      "r1",
			want:     true,
		},
		{
			name:     "lower generation loses",
			existGen: 5,
			existAt:  base,
			existW:   "r2",
			inGen:    4,
			inAt:     base.Add(time.Hour),
			inW:      "r1",
			want:     false,
		},
		{
			name:     "same generation newer timestamp wins",
			existGen: 3,
			existAt:  base,
			existW:   "r2",
			inGen:    3,
			inAt:     base.Add(time.Minute),
			inW:      "r1",
			want:     true,
		},
		{
			name:     "same generation older timestamp loses",
			existGen: 3,
			existAt:  base,
			existW:   "r2",
			inGen:    3,
			inAt:     base.Add(-time.Minute),
			inW:      "r1",
			want:     false,
		},
		{
			name:     "tiebreak writer id",
			existGen: 2,
			existAt:  base,
			existW:   "r1",
			inGen:    2,
			inAt:     base,
			inW:      "r2",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldAcceptWrite(tt.existGen, tt.existAt, tt.existW, tt.inGen, tt.inAt, tt.inW)
			if got != tt.want {
				t.Fatalf("ShouldAcceptWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}
