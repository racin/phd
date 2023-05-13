package metrics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/cli"
	"github.com/racin/phd/code/evaluation/util"
)

type MetricsSlice map[string][]float64
type Metrics map[string]float64

func GetMetrics(ctx context.Context, ctl bee.Controller) (metrics MetricsSlice, err error) {
	var mut sync.Mutex
	metrics = make(MetricsSlice)

	pw, _ := cli.NewProgress(os.Stdout, "Gathering metrics ", ctl.Len())
	defer pw.Done()

	err = ctl.Parallel(ctx, func(id int, host string) (err error) {
		defer pw.Increment()

		m := make(Metrics)

		res, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/metrics"},
		})
		if err != nil {
			return fmt.Errorf("failed to get metrics: %w", err)
		}
		defer util.SafeClose(&err, res.Body)
		err = util.CheckResponse(res)
		if err != nil {
			return fmt.Errorf("failed to get metrics: %w", err)
		}

		r := bufio.NewReader(res.Body)

		for {
			line, err := r.ReadString('\n')
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return fmt.Errorf("failed to read metrics: %w", err)
			}

			if strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.Split(strings.TrimSpace(line), " ")
			if len(parts) < 2 {
				continue
			}

			val, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				continue
			}

			m[parts[0]] = val
		}

		mut.Lock()
		insertSlice(metrics, m, id, ctl.BaseLen())
		mut.Unlock()

		return nil
	})
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func (a MetricsSlice) Add(b MetricsSlice) {
	for k, bvals := range b {
		avals := a[k]
		if avals == nil {
			avals = make([]float64, len(bvals))
		}
		for i := range bvals {
			avals[i] += bvals[i]
		}
		a[k] = avals
	}
}

func (a MetricsSlice) Sub(b MetricsSlice) {
	for k, bvals := range b {
		avals := a[k]
		if avals == nil {
			avals = make([]float64, len(bvals))
		}
		for i := range bvals {
			avals[i] -= bvals[i]
		}
		a[k] = avals
	}
}

func (a Metrics) Add(b Metrics) {
	for k, v := range b {
		a[k] += v
	}
}

func (a Metrics) Sub(b Metrics) {
	for k, v := range b {
		a[k] -= v
	}
}

func Filter(metrics Metrics, keys []string) Metrics {
	m := make(Metrics, len(keys))

	for _, key := range keys {
		if val, ok := metrics[key]; ok {
			m[key] = val
		}
	}

	return m
}

func insertSlice(s MetricsSlice, m Metrics, i, n int) {
	for key, val := range m {
		vals := s[key]
		if vals == nil {
			vals = make([]float64, n)
		}
		vals[i] = val
		s[key] = vals
	}
}

func Sum(s MetricsSlice) Metrics {
	m := make(Metrics, len(s))

	for key, vals := range s {
		sum := .0
		for _, val := range vals {
			sum += val
		}
		m[key] = sum
	}
	return m
}
