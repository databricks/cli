package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagerFakeIterator(t *testing.T) {
	tests := []struct {
		name      string
		rows      int
		failAfter int
		// wantRows is the number of rows Next returns before it stops.
		wantRows int
		wantErr  string
	}{
		{name: "no failure yields every row", rows: 3, failAfter: -1, wantRows: 3},
		{name: "failure after two rows", rows: 5, failAfter: 2, wantRows: 2, wantErr: "injected iterator error after 2 rows"},
		{name: "failure on the first Next", rows: 5, failAfter: 0, wantRows: 0, wantErr: "injected iterator error after 0 rows"},
		{name: "failure past the end is never reached", rows: 2, failAfter: 5, wantRows: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			it := &pagerFakeIterator{rows: buildPagerRows(tc.rows), failAfter: tc.failAfter}

			got := 0
			var err error
			for it.HasNext(ctx) {
				var row pagerRow
				row, err = it.Next(ctx)
				if err != nil {
					break
				}
				require.Equal(t, buildPagerRows(tc.rows)[got], row)
				got++
			}
			assert.Equal(t, tc.wantRows, got)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantErr)
			}
		})
	}
}
