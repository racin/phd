package sender

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethersphere/bee/pkg/log"
	"github.com/ethersphere/bee/pkg/p2p"
	"github.com/ethersphere/bee/pkg/p2p/protobuf"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/bee/pkg/topology"
	"github.com/gogo/protobuf/proto"
)

// Sender is able to send messages.
type Sender interface {
	// Send sends req to the peer closest to addr, excluding any peers in skipPeers.
	// The response is stored in resp.
	// If the closest peer is this node itself, the error returned will be topology.ErrWantSelf.
	// If an error occurs when receiving a response, the error returned will be of type Error.
	// If the request was not sent successfully, the error returned will be ErrNotSent.
	// Any other errors that occurred can be retrieved by calling GetLastErrors().
	SendClosest(ctx context.Context, addr swarm.Address, req, resp proto.Message, includeSelf bool, skipPeers ...swarm.Address) error

	// GetLastErrors returns the errors that occurred during the last call to SendClosest.
	GetLastErrors() []Error
}

type sender struct {
	mut        sync.Mutex
	lastErrors []Error

	protocol      p2p.ProtocolSpec
	stream        p2p.StreamSpec
	retries       int
	streamer      p2p.Streamer
	closestPeerer topology.ClosestPeerer
	logger        log.Logger
}

// New returns a new Sender.
func New(protocol p2p.ProtocolSpec, stream p2p.StreamSpec, retries int, streamer p2p.Streamer, closestPeerer topology.ClosestPeerer, logger log.Logger) Sender {
	return &sender{
		protocol:      protocol,
		stream:        stream,
		retries:       retries,
		streamer:      streamer,
		closestPeerer: closestPeerer,
		logger:        logger,
	}
}

// Send sends req to the peer closest to addr, excluding any peers in skipPeers.
// The response is stored in resp.
// If the closest peer is this node itself, the error returned will be topology.ErrWantSelf.
// If an error occurs when receiving a response, the error returned will be of type Error.
// If the request was not sent successfully, the error returned will be ErrNotSent.
// Any other errors that occurred can be retrieved by calling GetLastErrors().
func (s *sender) SendClosest(ctx context.Context, addr swarm.Address, req, resp proto.Message, includeSelf bool, skipPeers ...swarm.Address) (err error) {
	var (
		sent    = false
		retries = s.retries
		errs    []Error
	)

	defer func() {
		s.mut.Lock()
		s.lastErrors = errs
		s.mut.Unlock()
	}()

	send := func(peer swarm.Address) error {
		stream, err := s.streamer.NewStream(ctx, peer, nil, s.protocol.Name, s.protocol.Version, s.stream.Name)
		if err != nil {
			return err
		}
		defer func() { _ = stream.FullClose() }()

		w, r := protobuf.NewWriterAndReader(stream)

		err = w.WriteMsgWithContext(ctx, req)
		if err != nil {
			return err
		}
		sent = true

		err = r.ReadMsgWithContext(ctx, resp)
		if err != nil {
			return err
		}

		return nil
	}

	for {
		var closest swarm.Address
		closest, err = s.closestPeerer.ClosestPeer(addr, includeSelf, topology.Filter{Reachable: true}, skipPeers...)
		if err != nil {
			return err
		}

		skipPeers = append(skipPeers, closest)

		err = send(closest)
		if err == nil {
			return nil
		} else if sent {
			return Error{peer: closest, err: err}
		} else {
			s.logger.Debug("%s: error sending request to %v: %v", s.streamName(), closest, err)
			errs = append(errs, Error{peer: closest, err: err})
		}

		retries--
		if retries < 0 {
			return ErrNotSent
		}
	}
}

// GetLastErrors returns the errors that occurred during the last call to SendClosest.
func (s *sender) GetLastErrors() []Error {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.lastErrors
}

func (s *sender) streamName() string {
	return fmt.Sprintf("%s.%s", s.protocol.Name, s.stream.Name)
}
