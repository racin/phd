package pullsync

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/util"
)

func StartPullSync(ctx context.Context, ctl bee.Controller, beeID int) (duration time.Duration, err error) {
	host, disconnect, err := ctl.Connector().ConnectDebug(ctx, beeID)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)

	start := time.Now()

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/notifyPeerSig"},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to notify peer to start pullsync: %w", err)
	}
	defer util.SafeClose(&err, res.Body)
	if err := util.CheckResponse(res); err != nil {
		return 0, fmt.Errorf("unexpected response: %w", err)
	}

	return time.Since(start), nil
}

func ClearPullSync(ctx context.Context, ctl bee.Controller, beeID int, peer string) (duration time.Duration, err error) {
	host, disconnect, err := ctl.Connector().ConnectDebug(ctx, beeID)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)

	var path string
	if peer == "all" {
		path = "/debug/puller/cursors"
	} else {
		path = fmt.Sprintf("/debug/puller/cursors/%s", peer)
	}
	start := time.Now()

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodDelete,
		URL:    &url.URL{Scheme: "http", Host: host, Path: path},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to clear pullsync cursors: %w", err)
	}
	defer util.SafeClose(&err, res.Body)
	if err := util.CheckResponse(res); err != nil {
		return 0, fmt.Errorf("unexpected response: %w", err)
	}

	return time.Since(start), nil
}
