package api

import (
	"net/http"

	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/gorilla/mux"
)

func (s *Service) pullerClearCursorHandler(w http.ResponseWriter, r *http.Request) {
	if peeraddr, ok := mux.Vars(r)["peeraddr"]; ok {
		peer, err := swarm.ParseHexAddress(peeraddr)
		if err != nil {
			s.logger.Error(err, "puller clear cursor: unable to parse peer address", "PeerAddr", peeraddr)
			jsonhttp.InternalServerError(w, nil)
			return
		}
		s.puller.ClearCursors(r.Context(), *s.overlay, peer)
	} else {
		s.puller.ClearCursors(r.Context(), *s.overlay, swarm.ZeroAddress)
	}
	jsonhttp.OK(w, nil)
}
