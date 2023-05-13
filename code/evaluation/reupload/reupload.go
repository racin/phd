package reupload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/util"
)

func Steward(ctx context.Context, ctl bee.Controller, beeID int, rootHash string) (duration time.Duration, err error) {
	host, disconnect, err := ctl.Connector().ConnectAPI(ctx, beeID)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)

	start := time.Now()

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPut,
		URL:    &url.URL{Scheme: "http", Host: host, Path: fmt.Sprintf("/stewardship/%s", rootHash)},
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

type POSResult struct {
	Proved     map[string][]string
	Reuploaded []string
	Errored    []string
}

func POS(ctx context.Context, ctl bee.Controller, beeID int, rootHash string) (result POSResult, duration time.Duration, err error) {
	host, disconnect, err := ctl.Connector().ConnectAPI(ctx, beeID)
	if err != nil {
		return result, 0, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)

	start := time.Now()

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: host, Path: fmt.Sprintf("/pos/%s", rootHash)},
	})
	if err != nil {
		return result, 0, fmt.Errorf("failed to perform reupload: %w", err)
	}
	defer util.SafeClose(&err, res.Body)
	if err := util.CheckResponse(res); err != nil {
		return result, 0, fmt.Errorf("unexpected response: %w", err)
	}

	duration = time.Since(start)

	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&result)
	if err != nil {
		return result, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, duration, nil
}
