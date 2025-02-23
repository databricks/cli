package testdiff

import (
	"context"
)

type key int

const (
	replacementsMapKey = key(1)
)

func WithReplacementsMap(ctx context.Context) (context.Context, *ReplacementsContext) {
	value := ctx.Value(replacementsMapKey)
	if value != nil {
		if existingMap, ok := value.(*ReplacementsContext); ok {
			return ctx, existingMap
		}
	}

	newMap := &ReplacementsContext{}
	ctx = context.WithValue(ctx, replacementsMapKey, newMap)
	return ctx, newMap
}

func GetReplacementsMap(ctx context.Context) *ReplacementsContext {
	value := ctx.Value(replacementsMapKey)
	if value != nil {
		if existingMap, ok := value.(*ReplacementsContext); ok {
			return existingMap
		}
	}
	return nil
}
