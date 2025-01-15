package testdiff

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReplacementsMap_Nil(t *testing.T) {
	ctx := context.Background()
	repls := GetReplacementsMap(ctx)
	assert.Nil(t, repls)
}

func TestGetReplacementsMap_NotNil(t *testing.T) {
	ctx := context.Background()
	ctx, _ = WithReplacementsMap(ctx)
	repls := GetReplacementsMap(ctx)
	assert.NotNil(t, repls)
}

func TestWithReplacementsMap_UseExisting(t *testing.T) {
	ctx := context.Background()
	ctx, r1 := WithReplacementsMap(ctx)
	ctx, r2 := WithReplacementsMap(ctx)
	repls := GetReplacementsMap(ctx)
	assert.Equal(t, r1, repls)
	assert.Equal(t, r2, repls)
}
