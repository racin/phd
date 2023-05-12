package snips

import (
	"fmt"
	"io"

	"github.com/RoaringBitmap/roaring/roaring64"
)

type expectedVals []int32
type missingVals []uint64

func NewExpectedVals(length int) expectedVals {
	expectedChunks := make(expectedVals, length)
	for i := range expectedChunks {
		expectedChunks[i] = 1
	}

	return expectedChunks
}

func MissingFromBytes(bitvectorbuf io.Reader) (missingVals, error) {
	if bitvectorbuf == nil {
		return nil, fmt.Errorf("bitvectorbuf is nil")
	}
	bv := roaring64.New()
	_, err := bv.ReadFrom(bitvectorbuf)
	if err != nil {
		return nil, err
	}

	return bv.ToArray(), nil
}

func MissingFromBitVector(bitvector *roaring64.Bitmap) missingVals {
	if bitvector == nil {
		return nil
	}

	return bitvector.ToArray()
}

func (m *missingVals) AsBitVector() *roaring64.Bitmap {
	return roaring64.BitmapOf(*m...)
}

func (e *expectedVals) Hit(hashVal uint64) {
	(*e)[hashVal-1]--
}

func (e *expectedVals) GetMissingOrDuplicate() missingVals {
	missordup := make([]uint64, 0)

	for i := range *e {
		if (*e)[i] != 0 {
			missordup = append(missordup, uint64(i+1))
		}
	}
	return missordup
}

func (e *expectedVals) GetMissingChunks() missingVals {
	missing := make([]uint64, 0)

	for i := range *e {
		if (*e)[i] == 1 {
			missing = append(missing, uint64(i+1))
		}
	}
	return missing
}

func (e *expectedVals) GotMissingAndDuplicate() bool {
	missing, duplicate := false, false

	for i := range *e {
		if missing && duplicate {
			return true
		}
		val := (*e)[i]
		if val == 1 {
			missing = true
		} else if val == -1 {
			duplicate = true
		}
	}
	return missing && duplicate
}

func (e *expectedVals) DuplicateProof() []uint32 {
	duplicate := make([]uint32, 0)

	for i := range *e {
		if (*e)[i] < 0 {
			duplicate = append(duplicate, uint32(i+1))
		}
	}
	return duplicate
}
