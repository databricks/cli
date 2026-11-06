package resources

import (
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/ml"

	"github.com/stretchr/testify/assert"
)

// TestMaterializationSetReportsNoWorkspaceURL pins the signal that keeps `bundle summary` from
// printing a URL line for a set. BaseResource.GetURL returns ("", true), which the summary reads as
// a supported type awaiting deployment and renders as "(not deployed)".
func TestMaterializationSetReportsNoWorkspaceURL(t *testing.T) {
	set := &MaterializationSet{ID: "0123456789abcdef0123456789abcdef"}
	set.URL = "https://example.cloud.databricks.com/explore/data/features"

	rawURL, supportsURL := set.GetURL()
	assert.False(t, supportsURL)
	assert.Empty(t, rawURL)

	set.InitializeURL(url.URL{Scheme: "https", Host: "example.cloud.databricks.com"})
	rawURL, supportsURL = set.GetURL()
	assert.False(t, supportsURL, "InitializeURL must not make the type look URL-bearing")
	assert.Empty(t, rawURL)
}

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

	field, value := MaterializedFeatureMemberKey(offline)
	assert.Equal(t, "member", field)
	assert.Equal(t, "feature_name=main.default.feat|is_streaming_mode=false", value)

	_, onlineValue := MaterializedFeatureMemberKey(online)
	assert.Equal(t, "feature_name=main.default.feat|online_store_name=store_one|is_streaming_mode=false", onlineValue)

	_, streamingValue := MaterializedFeatureMemberKey(streaming)
	assert.Equal(t, "feature_name=main.default.feat|is_streaming_mode=true", streamingValue)

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

	_, first := MaterializedFeatureMemberKey(nameHoldsSeparator)
	_, second := MaterializedFeatureMemberKey(storeHoldsSeparator)
	assert.NotEqual(t, first, second)

	// The escape character itself is escaped first, so a name cannot forge an escaped separator.
	_, forged := MaterializedFeatureMemberKey(ml.MaterializedFeature{
		FeatureName:       "main.default.a%7Cb",
		OnlineStoreConfig: &ml.OnlineStoreConfig{OnlineStoreName: "c"},
	})
	assert.NotEqual(t, first, forged)
}

// An absent online destination and an online destination carrying no store name are distinct
// members, which is why the component is omitted rather than left empty.
func TestMaterializationSetMemberKeySeparatesAbsentOnlineStoreFromEmptyName(t *testing.T) {
	_, offline := MaterializedFeatureMemberKey(ml.MaterializedFeature{
		FeatureName:        "main.default.feat",
		OfflineStoreConfig: &ml.OfflineStoreConfig{CatalogName: "main", SchemaName: "default"},
	})
	_, onlineWithoutName := MaterializedFeatureMemberKey(ml.MaterializedFeature{
		FeatureName:       "main.default.feat",
		OnlineStoreConfig: &ml.OnlineStoreConfig{CatalogName: "main", SchemaName: "default"},
	})

	assert.Equal(t, "feature_name=main.default.feat|is_streaming_mode=false", offline)
	assert.Equal(t,
		"feature_name=main.default.feat|online_store_name=|is_streaming_mode=false",
		onlineWithoutName)
	assert.NotEqual(t, offline, onlineWithoutName)
}
