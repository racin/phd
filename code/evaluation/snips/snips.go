package snips

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/util"
)

func StartSNIPS(ctx context.Context, ctl bee.Controller, beeID int, nonce uint64) (duration time.Duration, err error) {
	host, disconnect, err := ctl.Connector().ConnectAPI(ctx, beeID)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)

	start := time.Now()

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: host, Path: fmt.Sprintf("/snips/%d", nonce)},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to perform reupload: %w", err)
	}
	defer util.SafeClose(&err, res.Body)
	if err := util.CheckResponse(res); err != nil {
		return 0, fmt.Errorf("unexpected response: %w", err)
	}

	duration = time.Since(start)

	return duration, nil
}
