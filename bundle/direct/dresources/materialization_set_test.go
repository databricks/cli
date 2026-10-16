package dresources

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
)

func TestMaterializationSetMemberKey(t *testing.T) {
	offline := ml.MaterializedFeature{
		FeatureName:        "main.default.feat",
		OfflineStoreConfig: &ml.OfflineStoreConfig{CatalogName: "main", SchemaName: "default"},
	}
	online := ml.MaterializedFeature{
		FeatureName:       "main.default.feat",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "store_one"},
	}
	streaming := ml.MaterializedFeature{
		FeatureName:        "main.default.feat",
		OfflineStoreConfig: &ml.OfflineStoreConfig{CatalogName: "main", SchemaName: "default"},
		StreamingMode:      &ml.StreamingMode{},
	}

	field, value := materializationSetMemberKey(offline)
	assert.Equal(t, "member", field)
	assert.Equal(t, "main.default.feat||false", value)

	_, onlineValue := materializationSetMemberKey(online)
	assert.Equal(t, "main.default.feat|store_one|false", onlineValue)

	_, streamingValue := materializationSetMemberKey(streaming)
	assert.Equal(t, "main.default.feat||true", streamingValue)

	// The three members of one feature must not collide with each other.
	assert.NotEqual(t, value, onlineValue)
	assert.NotEqual(t, value, streamingValue)
}

// A feature name may contain the separator, so without escaping these two distinct members would
// share a key: the plan would diff them as one and the update request would omit the member the
// server then deletes.
func TestMaterializationSetMemberKeyEscapesSeparator(t *testing.T) {
	nameHoldsSeparator := ml.MaterializedFeature{
		FeatureName:       "main.default.a|b",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "c"},
	}
	storeHoldsSeparator := ml.MaterializedFeature{
		FeatureName:       "main.default.a",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "b|c"},
	}

	_, first := materializationSetMemberKey(nameHoldsSeparator)
	_, second := materializationSetMemberKey(storeHoldsSeparator)
	assert.NotEqual(t, first, second)

	// The escape character itself is escaped first, so a name cannot forge an escaped separator.
	_, forged := materializationSetMemberKey(ml.MaterializedFeature{
		FeatureName:       "main.default.a%7Cb",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "c"},
	})
	assert.NotEqual(t, first, forged)
}
