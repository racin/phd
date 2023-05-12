package api

import (
	"net/http"

	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/gorilla/mux"
)

func updateSyncState(w http.ResponseWriter, r *http.Request, state *bool) {
	newstate := mux.Vars(r)["newstate"]
	if newstate == "enable" || newstate == "1" {
		*state = true
		jsonhttp.OK(w, nil)
	} else if newstate == "disable" || newstate == "0" {
		*state = false
		jsonhttp.OK(w, nil)
	} else {
		jsonhttp.NotFound(w, "Invalid new state. To enable pass (enable or 1), to disable pass (disable or 0).")
	}
}

func (s *Service) pushsyncUpdateStateHandler(w http.ResponseWriter, r *http.Request) {
	updateSyncState(w, r, s.pushSyncEnabled)
}

func (s *Service) pullsyncUpdateStateHandler(w http.ResponseWriter, r *http.Request) {
	updateSyncState(w, r, s.pullSyncEnabled)
}

func (s *Service) retrievalsyncUpdateStateHandler(w http.ResponseWriter, r *http.Request) {
	updateSyncState(w, r, s.retrievalSyncEnabled)
}

func (s *Service) snipsUpdateStateHandler(w http.ResponseWriter, r *http.Request) {
	updateSyncState(w, r, s.snipsOptions.Enabled)
}

func (s *Service) chainsyncUpdateStateHandler(w http.ResponseWriter, r *http.Request) {
	updateSyncState(w, r, s.chainSyncEnabled)
}

func (s *Service) posUpdateStateHandler(w http.ResponseWriter, r *http.Request) {
	updateSyncState(w, r, s.posEnabled)
}

func (s *Service) pushsyncGetStateHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, *s.pushSyncEnabled)
}

func (s *Service) pullsyncGetStateHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, *s.pullSyncEnabled)
}

func (s *Service) retrievalsyncGetStateHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, *s.retrievalSyncEnabled)
}

func (s *Service) snipsGetStateHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, *s.snipsOptions.Enabled)
}

func (s *Service) chainsyncGetStateHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, *s.chainSyncEnabled)
}

func (s *Service) posGetStateHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, *s.posEnabled)
}

func (s *Service) pushsyncClearSkiplistHandler(w http.ResponseWriter, r *http.Request) {
	state, err := s.pushsync.ResetSkiplist(r.Context())
	if err != nil {
		s.logger.Error(err, "Failed to run reset skiplist: %v")
		jsonhttp.InternalServerError(w, nil)
	} else {
		jsonhttp.OK(w, state)
	}
}
