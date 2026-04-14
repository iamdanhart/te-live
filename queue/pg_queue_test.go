package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeNewPosition(t *testing.T) {
	entries := []positionRow{
		{id: 1, pos: 1.0},
		{id: 2, pos: 2.0},
		{id: 3, pos: 3.0},
	}

	tests := []struct {
		name     string
		entries  []positionRow
		afterID  int
		want     float64
		wantOk   bool
	}{
		{
			name:    "move to front",
			entries: entries,
			afterID: 0,
			want:    0.0,
			wantOk:  true,
		},
		{
			name:    "move to front with empty queue",
			entries: nil,
			afterID: 0,
			want:    1.0,
			wantOk:  true,
		},
		{
			name:    "move to end",
			entries: entries,
			afterID: 3,
			want:    4.0,
			wantOk:  true,
		},
		{
			name:    "move between first and second",
			entries: entries,
			afterID: 1,
			want:    1.5,
			wantOk:  true,
		},
		{
			name:    "move between second and third",
			entries: entries,
			afterID: 2,
			want:    2.5,
			wantOk:  true,
		},
		{
			name:    "afterID not found",
			entries: entries,
			afterID: 99,
			want:    0,
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := computeNewPosition(tt.entries, tt.afterID)
			assert.Equal(t, tt.wantOk, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}