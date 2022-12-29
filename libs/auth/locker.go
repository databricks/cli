package auth

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/databricks/databricks-sdk-go/retries"
)

// TODO: pick the one not in use: https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml?search=9991
const lockPort = 9991

type portLocker struct {
	l net.Listener
}

func (pl *portLocker) Lock(ctx context.Context) error {
	var lc net.ListenConfig
	lockAddr := fmt.Sprintf("localhost:%d", lockPort)
	l, err := retries.Poll(ctx, 15*time.Second, func() (*net.Listener, *retries.Err) {
		l, err := lc.Listen(ctx, "tcp", lockAddr)
		if err != nil {
			return nil, retries.Continue(err)
		}
		return &l, nil
	})
	if err != nil {
		return err
	}
	pl.l = *l
	return nil
}

func (pl *portLocker) Unlock() {
	pl.l.Close()
}
