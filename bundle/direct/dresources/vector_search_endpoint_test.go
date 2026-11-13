package dresources

import (
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
)

func TestVectorSearchEndpointCreatedSince(t *testing.T) {
	start := time.UnixMilli(1_800_000_000_000)
	created := func(t time.Time) *vectorsearch.EndpointInfo {
		return &vectorsearch.EndpointInfo{CreationTimestamp: t.UnixMilli()}
	}

	assert.True(t, createdSince(created(start.Add(time.Second)), start))
	assert.True(t, createdSince(created(start.Add(-30*time.Second)), start), "within clock skew")
	assert.False(t, createdSince(created(start.Add(-2*time.Minute)), start), "pre-existing endpoint")
	assert.False(t, createdSince(&vectorsearch.EndpointInfo{}, start), "no creation timestamp")
}
