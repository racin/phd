// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/ethersphere/bee/pkg/swarm"
)

type addrResp struct {
	Overlay      *swarm.Address `json:"overlay"`
	Underlay     []string       `json:"underlay"`
	Ethereum     common.Address `json:"ethereum"`
	PublicKey    string         `json:"publicKey"`
	PSSPublicKey string         `json:"pssPublicKey"`
	HostName     string         `json:"hostName"`
}

func (s *Service) neighborsHandler(w http.ResponseWriter, r *http.Request) {
	unmarshalledNeighbors := s.snips.GetNeighborInfo(r.Context())
	marshalledNeighbors := make([]addrResp, 0)
	for i, neighbor := range unmarshalledNeighbors {
		var ar addrResp
		if err := json.Unmarshal(neighbor, &ar); err != nil {
			s.logger.Error(err, "neighborsHandler: error unmarshalling", "index", i)
		}
		marshalledNeighbors = append(marshalledNeighbors, ar)
	}
	jsonhttp.OK(w, marshalledNeighbors)
}

func (s *Service) topologyHandler(w http.ResponseWriter, r *http.Request) {
	params := s.topologyDriver.Snapshot()

	params.LightNodes = s.lightNodes.PeerInfo()

	b, err := json.Marshal(params)
	if err != nil {
		s.logger.Error(err, "topology get: marshal to json failed")
		jsonhttp.InternalServerError(w, err)
		return
	}
	w.Header().Set("Content-Type", jsonhttp.DefaultContentTypeHeader)
	_, _ = io.Copy(w, bytes.NewBuffer(b))
}

func (s *Service) notifyPeerSigHandler(w http.ResponseWriter, r *http.Request) {
	s.topologyDriver.NotifyPeerSig()
	jsonhttp.OK(w, nil)
}

func (s *Service) broadcastKnownPeers(w http.ResponseWriter, r *http.Request) {
	err := s.topologyDriver.BroadcastKnownPeers(r.Context())
	if err != nil {
		s.logger.Error(err, "topology marshal to json: %v")
		jsonhttp.InternalServerError(w, err)
	} else {
		jsonhttp.OK(w, nil)
	}
}
