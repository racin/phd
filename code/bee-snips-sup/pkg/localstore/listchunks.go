package localstore

import (
	"context"

	"github.com/ethersphere/bee/pkg/sharky"
	"github.com/ethersphere/bee/pkg/shed"
)

// IterStoredChunks will iterate over all stored chunks and call iterFunc for each of them.
func (db *DB) IterStoredChunks(ctx context.Context, withdata bool, iterFunc func(shed.Item) error) (err error) {
	return db.retrievalDataIndex.Iterate(func(item shed.Item) (stop bool, err error) {
		if withdata {
			l, err := sharky.LocationFromBinary(item.Location)
			if err != nil {
				return false, err
			}

			item.Data = make([]byte, l.Length)
			err = db.sharky.Read(ctx, l, item.Data)
			if err != nil {
				return false, err
			}
		}
		err = iterFunc(item)
		if err != nil {
			return true, err
		}
		if err = ctx.Err(); err != nil {
			return true, err
		}
		return false, nil
	}, nil)
}
