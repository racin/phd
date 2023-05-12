// Copyright 2021 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"net/http"

	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/gorilla/mux"
)

// New function similar to Retreival (/home/racin/src/bee-snarl/pkg/retrieval/retrieval.go)
// Should be initialized with (ns := netstore.New(storer, noopValidStamp, nil, retrieve, logger))
// (Or similar)

// /home/racin/src/bee-snarl/pkg/node/node.go
// ns = netstore.New(storer, validStamp, nil, retrieve, logger)
// 	pullStorage := pullstorage.New(storer)
// debugAPIService.Configure(swarmAddress, p2ps, pingPong, kad, lightNodes, storer, tagService, acc, pseudosettleService, o.SwapEnable, o.ChequebookEnable, debugSwapService, chequebookService, batchStore, post, postageContractService, traversalService)

// pinRootHash pins root hash of given reference. This method is idempotent.
func (s *Service) posHandler(w http.ResponseWriter, r *http.Request) {
	ref, err := swarm.ParseHexAddress(mux.Vars(r)["reference"])
	if err != nil {
		s.logger.Debug("posHandler: unable to parse reference %q: %v", ref, err)
		s.logger.Error(err, "posHandler: unable to parse reference")
		jsonhttp.BadRequest(w, "bad reference")
		return
	}

	// body, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	s.logger.Debugf("posHandler: auth handler: read request body: %v", err)
	// 	s.logger.Error("posHandler: auth handler: read request body")
	// 	jsonhttp.BadRequest(w, "Read request body")
	// 	return
	// }
	// w.Write(append([]byte("Received body: "), body...))

	// var payload securityTokenReq
	// if err = json.Unmarshal(body, &payload); err != nil {
	// 	s.logger.Debugf("posHandler: auth handler: unmarshal request body: %v", err)
	// 	s.logger.Error("posHandler: auth handler: unmarshal request body")
	// 	jsonhttp.BadRequest(w, "Unmarshal json body. "+err.Error())
	// 	return
	// }

	status, err := s.pos.Reupload(r.Context(), ref)
	if err != nil {
		s.logger.Debug("pos put: re-upload %s: %v", ref, err)
		s.logger.Error(err, "pos put: re-upload")
		jsonhttp.InternalServerError(w, nil)
		return
	}

	if len(status.Errored) == 0 {
		jsonhttp.OK(w, status)
	} else {
		jsonhttp.InternalServerError(w, status)
	}
}
