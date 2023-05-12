package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/ethersphere/bee/pkg/shed"
	"github.com/ethersphere/bee/pkg/swarm"
)

func (s *Service) debugDBIndicesHandler(w http.ResponseWriter, r *http.Request) {
	di, err := s.localStore.DebugIndices()
	if err != nil {
		s.logger.Error(err, "Failed to run DebugIndices: %v")
		jsonhttp.InternalServerError(w, nil)
		return
	}
	jsonhttp.OK(w, di)
}

func (s *Service) debugDBContentsHandler(w http.ResponseWriter, r *http.Request) {
	err := s.localStore.IterStoredChunks(r.Context(), false, func(item shed.Item) error {
		a := swarm.NewAddress(item.Address)
		_, err := io.WriteString(w, a.String()+"\n")
		return err
	})
	if err != nil {
		s.logger.Error(err, "Failed to run ListStoredChunks: %v")
		jsonhttp.InternalServerError(w, nil)
		return
	}
}

func (s *Service) debugDBLastIndexHandler(w http.ResponseWriter, r *http.Request) {
	lastChunks := s.localStore.LastPullChunks()
	for _, c := range lastChunks {
		str := fmt.Sprintf("Addr: %x, BatchID: %x BucketDepth: %d Index: %d, Depth: %d\n", c.Address, c.BatchID, c.BucketDepth, c.Index, c.Depth)
		_, err := io.WriteString(w, str)
		if err != nil {
			s.logger.Error(err, "Failed to print last index: %v")
			jsonhttp.InternalServerError(w, nil)
			return
		}
	}
}
