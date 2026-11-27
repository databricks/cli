package releases_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/releases"
	"github.com/stretchr/testify/require"
)

type stalledDownload struct{ started chan struct{} }

func (download stalledDownload) RoundTrip(request *http.Request) (*http.Response, error) {
	close(download.started)
	<-request.Context().Done()
	return nil, request.Context().Err()
}

func TestPublishedDownloadHonorsCancellation(t *testing.T) {
	started := make(chan struct{})
	previous := http.DefaultTransport
	http.DefaultTransport = stalledDownload{started: started}
	t.Cleanup(func() { http.DefaultTransport = previous })
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		_, err := releases.OpenPublished(ctx, "amd64", "1.0.0", "")
		finished <- err
	}()
	<-started
	cancel()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("release download ignored cancellation")
	}
}
