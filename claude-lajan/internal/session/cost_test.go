package session

import (
	"math"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		input       int64
		output      int64
		cacheCreate int64
		cacheRead   int64
		wantUSD     float64
	}{
		{
			name:    "sonnet-4-5: 1M input + 1M output",
			model:   "claude-sonnet-4-5",
			input:   1_000_000,
			output:  1_000_000,
			wantUSD: 3.00 + 15.00, // $18
		},
		{
			name:    "opus-4-6: 1M input + 1M output",
			model:   "claude-opus-4-6",
			input:   1_000_000,
			output:  1_000_000,
			wantUSD: 15.00 + 75.00, // $90
		},
		{
			name:    "haiku-4-5: 1M input + 1M output",
			model:   "claude-haiku-4-5",
			input:   1_000_000,
			output:  1_000_000,
			wantUSD: 0.80 + 4.00, // $4.80
		},
		{
			name:    "unknown model falls back to sonnet pricing",
			model:   "claude-unknown-model",
			input:   1_000_000,
			output:  1_000_000,
			wantUSD: 3.00 + 15.00, // $18
		},
		{
			name:        "cache read reduces effective cost",
			model:       "claude-sonnet-4-5",
			input:       0,
			output:      0,
			cacheCreate: 1_000_000,
			cacheRead:   1_000_000,
			wantUSD:     3.75 + 0.30, // $4.05
		},
		{
			name:    "zero tokens → zero cost",
			model:   "claude-sonnet-4-5",
			input:   0,
			output:  0,
			wantUSD: 0.0,
		},
		{
			name:    "small token count (100 input, 50 output)",
			model:   "claude-sonnet-4-5",
			input:   100,
			output:  50,
			wantUSD: (100*3.00 + 50*15.00) / 1_000_000.0,
		},
	}

	const eps = 1e-9

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateCost(tc.model, tc.input, tc.output, tc.cacheCreate, tc.cacheRead)
			if math.Abs(got-tc.wantUSD) > eps {
				t.Errorf("CalculateCost(%q, %d, %d, %d, %d) = %.10f, want %.10f",
					tc.model, tc.input, tc.output, tc.cacheCreate, tc.cacheRead, got, tc.wantUSD)
			}
		})
	}
}
