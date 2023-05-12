package snips

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee/pkg/p2p"
	"github.com/ethersphere/bee/pkg/p2p/protobuf"
	"github.com/ethersphere/bee/pkg/snips/pb"
	"github.com/ethersphere/bee/pkg/swarm"
)

func (s *Service) GetNeighborInfo(ctx context.Context) [][]byte {
	allResp := make([][]byte, 0)
	s.topology.EachNeighbor(func(addr swarm.Address, u uint8) (stop bool, jumpToNext bool, err error) {
		nstream, err := s.streamer.NewStream(ctx, addr, nil, protocolName, protocolVersion, streamNameSNIPSneighbors)
		if err != nil {
			s.logger.Error(err, "GetNeighborInfo: error creating stream")
			return false, false, err
		}
		defer func() { _ = nstream.Close() }()

		nwriter, nreader := protobuf.NewWriterAndReader(nstream)
		if err := nwriter.WriteMsgWithContext(ctx, &pb.ReqAddresses{}); err != nil {
			s.logger.Error(err, "GetNeighborInfo: error writing message")
			return false, false, err
		}
		var resp pb.Addresses
		if err := nreader.ReadMsgWithContext(ctx, &resp); err != nil {
			s.logger.Error(err, "GetNeighborInfo: error reading message")
		} else {
			allResp = append(allResp, resp.Marshalled)
		}

		return false, false, nil
	})
	return allResp
}

func (s *Service) handlerIncomingNeighbors(ctx context.Context, p p2p.Peer, stream p2p.Stream) error {
	s.logger.Debug("GetNeighborInfo: got request", "addr", p.Address)
	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: "localhost:1635", Path: "/addresses"},
	})
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	addresses, err := ioutil.ReadAll(res.Body)
	if err != nil {
		s.logger.Error(err, "handlerIncomingNeighbors: error reading response body")
		return err
	}
	awriter := protobuf.NewWriter(stream)
	s.logger.Debug("GetNeighborInfo: writing response", "addr", p.Address, "data", addresses, "dataS", string(addresses))
	err = awriter.WriteMsgWithContext(ctx, &pb.Addresses{
		Marshalled: addresses,
	})
	if err != nil {
		s.logger.Error(err, "handlerIncomingNeighbors: error writing message")
		return fmt.Errorf("handlerIncomingNeighbors: failed to write message: %w", err)
	}

	return nil
}
