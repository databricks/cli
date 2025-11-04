package proxy

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"golang.org/x/sync/errgroup"
)

func RunClientProxy(ctx context.Context, src io.Reader, dst io.Writer, requestHandoverTick func() <-chan time.Time, createConn createWebsocketConnectionFunc) error {
	proxy := newProxyConnection(createConn)
	cmdio.LogString(ctx, "Establishing SSH proxy connection...")
	g, gCtx := errgroup.WithContext(ctx)
	if err := proxy.connect(gCtx); err != nil {
		return fmt.Errorf("failed to connect to proxy: %w", err)
	}
	defer proxy.close()
	cmdio.LogString(ctx, "SSH proxy connection established")

	g.Go(func() error {
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case <-requestHandoverTick():
				err := proxy.initiateHandover(gCtx)
				if err != nil {
					return err
				}
			}
		}
	})

	g.Go(func() error {
		return proxy.start(gCtx, src, dst)
	})

	return g.Wait()
}
