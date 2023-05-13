package distribution

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/util"
	"go.uber.org/multierr"
)

type ChunkDistribution map[string][]string
type NodeDistribution map[string][]string

func GetChunkDistribution(ctx context.Context, ctl bee.Controller, nameformat string, chunkfilter func(string) bool) (cd ChunkDistribution, nd NodeDistribution, err error) {
	var mut sync.Mutex
	cd = make(ChunkDistribution)
	nd = make(NodeDistribution)

	err = ctl.Parallel(ctx, func(id int, host string) (err error) {
		res, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/dbcontents"},
		})
		if err != nil {
			return fmt.Errorf("failed to request database contents from %d: %w", id, err)
		}
		defer util.SafeClose(&err, res.Body)
		err = util.CheckResponse(res)
		if err != nil {
			return err
		}

		mut.Lock()
		defer mut.Unlock()

		pod := fmt.Sprintf(nameformat, id)
		nd[pod] = make([]string, 0)
		rd := bufio.NewReader(res.Body)

		for {
			line, rerr := rd.ReadString('\n')
			if errors.Is(rerr, io.EOF) {
				break
			} else if rerr != nil {
				err = multierr.Append(rerr, fmt.Errorf("failed to read from %d's response body: %w", id, err))
				return
			}

			chunkid := strings.TrimSpace(line)
			if chunkfilter == nil || chunkfilter(chunkid) {
				if len(cd[chunkid]) == 0 {
					cd[chunkid] = make([]string, 0)
				}
				cd[chunkid] = append(cd[chunkid], pod)
				nd[pod] = append(nd[pod], chunkid)
			}
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return cd, nd, nil
}
