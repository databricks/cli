package testserver

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/require"
)

func TestMergeInto(t *testing.T) {
	existing := sql.GetWarehouseResponse{ //exhaustruct:ignore
		Name:         "wh",
		ClusterSize:  "2X-Small",
		AutoStopMins: 10,
		Channel:      &sql.Channel{Name: sql.ChannelNameChannelNameCurrent}, //exhaustruct:ignore
		Tags: &sql.EndpointTags{CustomTags: []sql.EndpointTagPair{ //exhaustruct:ignore
			{Key: "team", Value: "eng"},
		}},
	}

	// A field the body omits keeps its value; that is what makes clearing it never converge.
	got, err := mergeInto(existing, []byte(`{"name":"wh2"}`))
	require.NoError(t, err)
	require.Equal(t, "wh2", got.Name)
	require.Equal(t, "2X-Small", got.ClusterSize)
	require.Equal(t, 10, got.AutoStopMins)

	// An explicit empty value does clear it, which is what ForceSendFields produces.
	got, err = mergeInto(existing, []byte(`{"cluster_size":""}`))
	require.NoError(t, err)
	require.Empty(t, got.ClusterSize)
	require.Equal(t, "wh", got.Name)

	// A nested object merges key by key rather than being replaced.
	got, err = mergeInto(existing, []byte(`{"channel":{"dbsql_version":"2024.15"}}`))
	require.NoError(t, err)
	require.Equal(t, sql.ChannelNameChannelNameCurrent, got.Channel.Name)
	require.Equal(t, "2024.15", got.Channel.DbsqlVersion)

	// A list is replaced whole: an element is not separately addressable.
	got, err = mergeInto(existing, []byte(`{"tags":{"custom_tags":[{"key":"owner","value":"me"}]}}`))
	require.NoError(t, err)
	require.Len(t, got.Tags.CustomTags, 1)
	require.Equal(t, "owner", got.Tags.CustomTags[0].Key)
	require.Equal(t, "me", got.Tags.CustomTags[0].Value)
}
