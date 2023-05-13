package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/cli"
	"github.com/racin/phd/code/evaluation/util"
)

type State uint8

const (
	Unknown State = iota
	Enabled
	Disabled
)

func SetProtocol(ctx context.Context, ctl bee.Controller, protocolName string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}

	pw, _ := cli.NewProgress(os.Stdout, fmt.Sprintf("%s %s ", action, protocolName), ctl.Len())
	defer pw.Done()

	return ctl.Parallel(ctx, func(id int, host string) (err error) {
		defer pw.Increment()

		res, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Scheme: "http", Host: host, Path: fmt.Sprintf("/debug/%s/%s", protocolName, action)},
		})
		if err != nil {
			return fmt.Errorf("failed to %s %s for bee %d: %w", action, protocolName, id, err)
		}
		defer util.SafeClose(&err, res.Body)
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected response when %s %s for bee %d: %s", action, protocolName, id, res.Status)
		}

		return nil
	})
}

func GetProtocolState(ctx context.Context, ctl bee.Controller, protocolName string) ([]State, error) {
	states := make([]State, ctl.BaseLen())

	pw, _ := cli.NewProgress(os.Stdout, fmt.Sprintf("querying %s state ", protocolName), ctl.Len())
	defer pw.Done()

	err := ctl.Parallel(ctx, func(id int, host string) (err error) {
		defer pw.Increment()

		res, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Scheme: "http", Host: host, Path: fmt.Sprintf("/debug/%s", protocolName)},
		})
		if err != nil {
			return fmt.Errorf("failed to query %s state for bee %d: %w", protocolName, id, err)
		}
		defer util.SafeClose(&err, res.Body)
		err = util.CheckResponse(res)
		if err != nil {
			return fmt.Errorf("unexpected response from bee %d: %w", id, err)
		}

		var enabled bool
		_, err = fmt.Fscanf(res.Body, "%t", &enabled)
		if err != nil {
			return fmt.Errorf("failed to read response body for bee %d: %w", id, err)
		}

		if enabled {
			states[id] = Enabled
		} else {
			states[id] = Disabled
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return states, nil
}

func SetPushSync(ctx context.Context, ctl bee.Controller, enabled bool) error {
	return SetProtocol(ctx, ctl, "pushsync", enabled)
}

func SetPullSync(ctx context.Context, ctl bee.Controller, enabled bool) error {
	return SetProtocol(ctx, ctl, "pullsync", enabled)
}

func SetRetrievalSync(ctx context.Context, ctl bee.Controller, enabled bool) error {
	return SetProtocol(ctx, ctl, "retrievalsync", enabled)
}

func SetChainSync(ctx context.Context, ctl bee.Controller, enabled bool) error {
	return SetProtocol(ctx, ctl, "chainsync", enabled)
}

func SetPOS(ctx context.Context, ctl bee.Controller, enabled bool) error {
	return SetProtocol(ctx, ctl, "pos", enabled)
}

func SetSNIPS(ctx context.Context, ctl bee.Controller, enabled bool) error {
	return SetProtocol(ctx, ctl, "snips", enabled)
}

func SetProtocols(ctx context.Context, ctl bee.Controller, enableProtocols []string, disableProtocols []string) (err error) {
	for _, protocol := range enableProtocols {
		err = SetProtocol(ctx, ctl, protocol, true)
		if err != nil {
			return fmt.Errorf("failed to enable protocol %s: %w", protocol, err)
		}
	}
	for _, protocol := range disableProtocols {
		err = SetProtocol(ctx, ctl, protocol, false)
		if err != nil {
			return fmt.Errorf("failed to disable protocol %s: %w", protocol, err)
		}
	}
	return nil
}
