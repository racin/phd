// Copyright 2021 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/gorilla/mux"
)

func (s *Service) snipsHandler(w http.ResponseWriter, r *http.Request) {
	nonce, err := strconv.ParseUint(mux.Vars(r)["nonce"], 10, 64)

	if err != nil {
		s.logger.Debug("snipsHandler: unable to parse nonce %q: %v", mux.Vars(r)["nonce"], err)
		s.logger.Error(err, "snipsHandler: unable to parse nonce")
		jsonhttp.BadRequest(w, "bad nonce")
		return
	}
	fmt.Printf("Starting CreateAndSendProof with nonce %d\n", nonce)
	if err := s.snips.CreateAndSendProofForMyChunks(r.Context(), nonce); err != nil {
		s.logger.Error(err, "snips CreateAndSendProofForMyChunks", "Nonce", nonce)
		jsonhttp.InternalServerError(w, nil)
		return
	}

	jsonhttp.OK(w, nonce)
}

func (s *Service) snipsOptionHandler(w http.ResponseWriter, r *http.Request) {
	for key, val := range mux.Vars(r) {
		if err := s.handleSnipsOption(strings.ToLower(key), val); err != nil {
			s.logger.Error(err, "snips option handler")
		}
	}
}

func (s *Service) handleSnipsOption(key, val string) error {
	switch key {
	case "mingamma":
		if newval, err := strconv.ParseFloat(val, 64); err != nil {
			*s.snipsOptions.MinGamma = newval
		} else {
			return err
		}
	case "maxgamma":
		if newval, err := strconv.ParseFloat(val, 64); err != nil {
			*s.snipsOptions.MaxGamma = newval
		} else {
			return err
		}
	case "blockinterval":
		if newval, err := strconv.ParseUint(val, 10, 64); err != nil {
			*s.snipsOptions.BlockInterval = newval
		} else {
			return err
		}
	case "blockcheckinterval":
		if newval, err := strconv.Atoi(val); err != nil {
			*s.snipsOptions.BlockCheckInterval = time.Duration(newval) * time.Second
		} else {
			return err
		}
	}
	return nil
}
