package bee

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"go.uber.org/multierr"

	"github.com/racin/phd/code/evaluation/cli"
	"github.com/racin/phd/code/evaluation/util"
)

type beeConnection struct {
	host  string
	close func() error
}

type Controller interface {
	// Connector returns the connector used by this controller.
	Connector() *Connector
	// IDs returns a list of bee ids that this controller is managing.
	IDs() []int
	// BaseLen returns then umber of bees that the base controller is managing.
	BaseLen() int
	// Len returns the number of bees that this controller is managing.
	Len() int
	// Parallel runs beeFunc on all bees connected to this controller.
	Parallel(ctx context.Context, beeFunc func(id int, host string) (err error)) error
	// Sub returns a controller that manages only the ids specified in ids.
	Sub(ids []int) Controller
	// Close closes the bee connections.
	Close() error
	// CheckHealth checks that bees are reachable, and reconnects to bees that are not reachable.
	CheckHealth(ctx context.Context) error
}

type baseController struct {
	conn     Connector
	bees     []*beeConnection
	parallel int
}

func NewController(ctx context.Context, connector Connector, numBees int, parallel int, retries int, retryInterval time.Duration) (_ Controller, err error) {
	var (
		mut sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, parallel)
	)

	ctl := &baseController{
		conn:     connector,
		bees:     make([]*beeConnection, numBees),
		parallel: parallel,
	}

	pw, err := cli.NewProgress(os.Stdout, "Connecting to Bees: ", numBees)
	if err != nil {
		return nil, fmt.Errorf("failed to create progress writer: %w", err)
	}
	defer pw.Done()

	errChan := make(chan error, 1)

	for i := 0; i < numBees; i++ {
		if len(errChan) != 0 {
			break
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(id int) {
			defer func() {
				<-sem
				wg.Done()
			}()
			var (
				host       string
				disconnect func() error
				err        error
			)
			for retry := 0; retry < retries; retry++ {
				host, disconnect, err = connector.ConnectDebug(ctx, id)
				if err == nil || err == ctx.Err() {
					break
				}
				time.Sleep(retryInterval)
			}
			if err != nil {
				errChan <- fmt.Errorf("error connecting to bee %d: %w", id, err)
				return
			}
			mut.Lock()
			ctl.bees[id] = &beeConnection{
				host:  host,
				close: disconnect,
			}
			mut.Unlock()
			pw.Increment()
		}(i)
	}

	doneChan := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneChan)
	}()

loop:
	for {
		select {
		case cerr := <-errChan:
			err = multierr.Append(err, cerr)
		case <-doneChan:
			break loop
		}
	}

	if err != nil {
		_ = ctl.Close()
		return nil, err
	}

	return ctl, nil
}

func (ctl *baseController) Connector() *Connector {
	return &ctl.conn
}

func (ctl *baseController) Close() (err error) {
	for i, conn := range ctl.bees {
		if conn == nil || conn.close == nil {
			continue
		}
		cerr := conn.close()
		if cerr != nil {
			err = multierr.Append(err, fmt.Errorf("error disconnecting from bee %d: %w", i, cerr))
		}
	}
	return err
}

func (ctl *baseController) IDs() (ids []int) {
	for i := range ctl.bees {
		ids = append(ids, i)
	}
	return ids
}

func (ctl *baseController) Len() int {
	return len(ctl.bees)
}

func (ctl *baseController) BaseLen() int {
	return ctl.Len()
}

type ResponseOrError struct {
	Response *http.Response
	Err      error
}

// Parallel runs the beeFunc in parallel over all the bees.
func (ctl *baseController) Parallel(ctx context.Context, beeFunc func(id int, host string) error) (err error) {
	return ctl.doParallel(ctx, nil, beeFunc)
}

// doParallel runs the beeFunc in parallel over the specified bees.
// If ids is nil, the beeFunc is run over all the bees.
func (ctl *baseController) doParallel(ctx context.Context, ids []int, beeFunc func(id int, host string) error) (err error) {
	var (
		wg  sync.WaitGroup
		sem = make(chan struct{}, ctl.parallel)
		id  int
		bee *beeConnection
	)

	errChan := make(chan error, 1)

	fn := func() {
		sem <- struct{}{}
		wg.Add(1)

		go func(id int, host string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			berr := beeFunc(id, host)
			if berr != nil {
				errChan <- berr
			}

		}(id, bee.host)
	}

	if ids == nil {
		for id, bee = range ctl.bees {
			if len(errChan) != 0 || ctx.Err() != nil {
				break
			}
			fn()
		}
	} else {
		for _, id = range ids {
			if len(errChan) != 0 || ctx.Err() != nil {
				break
			}
			if id < 0 || id >= len(ctl.bees) {
				return fmt.Errorf("bee %d not found", id)
			}
			bee = ctl.bees[id]
			fn()
		}
	}

	doneChan := make(chan struct{})

	go func() {
		wg.Wait()
		close(doneChan)
	}()

loop:
	for {
		select {
		case cerr := <-errChan:
			err = multierr.Append(err, cerr)
		case <-doneChan:
			break loop
		}
	}

	if err == nil {
		return ctx.Err()
	}

	return err
}

func (ctl *baseController) Hostname(beeID int) (host string, ok bool) {
	if beeID < 0 || beeID >= ctl.Len() {
		return "", false
	}
	return ctl.bees[beeID].host, true
}

// CheckHealth checks that bees are reachable, and reconnects to bees that are not reachable.
func (ctl *baseController) CheckHealth(ctx context.Context) (err error) {
	pw, _ := cli.NewProgress(os.Stdout, "Checking health ", ctl.Len())
	defer pw.Done()

	return ctl.Parallel(ctx, func(id int, host string) (err error) {
		defer pw.Increment()

		res, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/health"},
		})
		if err == nil {
			defer util.SafeClose(&err, res.Body)
			err = util.CheckResponse(res)
			if err != nil {
				return nil
			}
		}

		conn := ctl.bees[id]
		_ = conn.close()

		host, disconnect, err := ctl.conn.ConnectDebug(ctx, id)
		if err != nil {
			return err
		}

		conn.host = host
		conn.close = disconnect

		return nil
	})
}

type subController struct {
	*baseController
	ids []int
}

func (ctl *baseController) Sub(ids []int) Controller {
	return &subController{
		baseController: ctl,
		ids:            ids,
	}
}

func (sub *subController) IDs() []int {
	return sub.ids
}

func (sub *subController) Len() int {
	return len(sub.ids)
}

func (sub *subController) Parallel(ctx context.Context, beeFunc func(int, string) error) error {
	return sub.baseController.doParallel(ctx, sub.ids, beeFunc)
}

func (sub *subController) Close() error {
	return errors.New("close should not be called on a sub controller")
}
