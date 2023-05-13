package experiments

import (
	"context"
	"fmt"
	"reflect"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/protocols"
	"github.com/racin/phd/code/evaluation/util"
	"golang.org/x/exp/slices"
)

// FailNodes will disable the specified protocols for n random bees, except for the bees in the white list.
// The restore function can be used to re-enable the protocols of the failed bees.
func FailNodes(ctx context.Context, ctl bee.Controller, n int, protocolNames []string, whiteList []int) (restore func() error, err error) {
	ids := ctl.IDs()
	for _, id := range whiteList {
		i := slices.Index(ids, id)
		if i != -1 {
			ids = slices.Delete(ids, i, i+1)
		}
	}

	util.Rand.Shuffle(len(ids), reflect.Swapper(ids))

	if n > len(ids) {
		return nil, fmt.Errorf("cannot fail %d of %d bees", n, len(ids))
	}

	failCtl := ctl.Sub(ids[:n])
	err = protocols.SetProtocols(ctx, failCtl, nil, protocolNames)
	if err != nil {
		return nil, err
	}

	restore = func() error {
		return protocols.SetProtocols(ctx, failCtl, protocolNames, nil)
	}

	return restore, nil
}
