package api

import (
	"io"
	"net/http"

	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/gorilla/mux"
)

func (s *Service) traverseHandler(w http.ResponseWriter, r *http.Request) {
	rootAddr, err := swarm.ParseHexAddress(mux.Vars(r)["root"])
	if err != nil {
		jsonhttp.BadRequest(w, nil)
		return
	}
	// need to traverse first before writing response, because traversal might fail.
	var addrs []swarm.Address
	err = s.traversal.Traverse(r.Context(), rootAddr, func(addr swarm.Address) error {
		addrs = append(addrs, addr)
		return nil
	})
	if err != nil {
		s.logger.Error(err, "failed to traverse %s: %v", rootAddr)
		jsonhttp.InternalServerError(w, nil)
	}
	w.WriteHeader(http.StatusOK)
	for _, addr := range addrs {
		_, err = io.WriteString(w, addr.String()+"\n")
		if err != nil {
			s.logger.Error(err, "failed to write response: %v")
			return
		}
	}
}
