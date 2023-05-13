// Package pos provides a proof-of-storage protocol as an alternative to the data stewardship protocol.
// Before reuploading chunks, a node can issue a proof-of-storage challenge to the neighborhoods of target chunks.
// The node then collects proofs from nodes that are storing the target chunks.
// If enough proofs are received, the node can skip reuploading chunks, potentially saving some bandwidth.
package pos

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/log"
	"github.com/ethersphere/bee/pkg/p2p"
	"github.com/ethersphere/bee/pkg/p2p/protobuf"
	"github.com/ethersphere/bee/pkg/pos/pb"
	"github.com/ethersphere/bee/pkg/pos/sender"
	"github.com/ethersphere/bee/pkg/pushsync"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/bee/pkg/topology"
	"github.com/ethersphere/bee/pkg/traversal"
)

const (
	protocolName    = "dsps"
	protocolVersion = "1.0.0"
	streamName      = "dsps"

	// how many parallel operations
	parallel = 5
	// how many peers to try
	maxPeers = 16
)

var (
	ErrNoChallenge     = errors.New("could not send challenge")
	ErrInvalidProof    = errors.New("invalid proof")
	ErrMissingChunk    = errors.New("missing chunk")
	ErrShallowResponse = errors.New("shallow response")
)

// ReuploadResult contains the addresses of all the chunks that were proved, reuploaded, or errored.
type ReuploadResult struct {
	Proved     map[string][]swarm.Address
	Reuploaded []swarm.Address
	Errored    []swarm.Address
}

type GetHasser interface {
	storage.Getter
	storage.Hasser
}

type Interface interface {
	Reupload(ctx context.Context, root swarm.Address) (ReuploadResult, error)
}

type Options struct {
	Enabled *bool
}

type Service struct {
	logger    log.Logger
	push      pushsync.PushSyncer
	sender    sender.Sender
	signer    crypto.Signer
	storer    GetHasser
	topology  topology.Driver
	traverser traversal.Traverser

	o Options

	metrics metrics

	address   swarm.Address
	blockHash []byte
	networkID uint64
}

func New(address swarm.Address, blockHash []byte, networkID uint64, logger log.Logger, push pushsync.PushSyncer, signer crypto.Signer, storer storage.Storer, streamer p2p.Streamer, topology topology.Driver, opts Options) *Service {
	if opts.Enabled == nil {
		opts.Enabled = new(bool)
	}

	s := &Service{
		logger:    logger,
		push:      push,
		signer:    signer,
		storer:    storer,
		topology:  topology,
		traverser: traversal.New(storer),

		o: opts,

		metrics: newMetrics(),

		address:   address,
		blockHash: blockHash,
		networkID: networkID,
	}

	s.sender = sender.New(s.Protocol(), s.Protocol().StreamSpecs[0], maxPeers, streamer, topology, logger)

	return s
}

func (s *Service) Protocol() p2p.ProtocolSpec {
	return p2p.ProtocolSpec{
		Name:    protocolName,
		Version: protocolVersion,
		StreamSpecs: []p2p.StreamSpec{
			{
				Name:    streamName,
				Handler: s.handler,
			},
		},
	}
}

type status uint8

const (
	statusErrored status = iota
	statusProved
	statusReuploaded
)

type chunkStatus struct {
	prover swarm.Address
	addr   swarm.Address
	status status
}

// Reupload is called when a client wants to re-upload the given root hash to the network.
// The service will automatically dereference and traverse all
// addresses and build a proof-of-storage challenge for every chunk.
// It assumes all chunks are available locally. It is therefore
// advisable to pin the content locally before trying to reupload it.
func (s *Service) Reupload(ctx context.Context, root swarm.Address) (ReuploadResult, error) {
	start := time.Now()
	result := ReuploadResult{
		Proved: make(map[string][]swarm.Address),
	}

	addrs, err := s.getChunkAddrs(ctx, root)
	if err != nil {
		return result, fmt.Errorf("failed to get chunk addresses: %w", err)
	}
	s.metrics.GetChunkAddrsTime.Observe(time.Since(start).Seconds())
	s.metrics.TotalChunks.Add(float64(len(addrs)))

	sem := make(chan struct{}, parallel)
	statusChan := make(chan chunkStatus, len(addrs))

	for _, addr := range addrs {

		sem <- struct{}{}
		go func(addr swarm.Address) {
			defer func() { <-sem }()
			statusChan <- s.reuploadChunk(ctx, addr)
		}(addr)
	}

	for range addrs {
		cs := <-statusChan
		switch cs.status {
		case statusErrored:
			result.Errored = append(result.Errored, cs.addr)
		case statusProved:
			proverStr := cs.prover.String()
			result.Proved[proverStr] = append(result.Proved[proverStr], cs.addr)
		case statusReuploaded:
			result.Reuploaded = append(result.Reuploaded, cs.addr)
		}
	}

	return result, nil
}

func (s *Service) reuploadChunk(ctx context.Context, chunkAddr swarm.Address) (status chunkStatus) {
	start := time.Now()

	defer func() {
		switch status.status {
		case statusErrored:
			s.metrics.TotalErrored.Inc()
			s.metrics.ErrorTime.Observe(time.Since(start).Seconds())
		case statusProved:
			s.metrics.TotalProved.Inc()
			s.metrics.ProvedTime.Observe(time.Since(start).Seconds())
		case statusReuploaded:
			s.metrics.TotalReuploaded.Inc()
			s.metrics.ReuploadTime.Observe(time.Since(start).Seconds())
		}
	}()

	nonce, err := s.generateNonce()
	if err != nil {
		s.logger.Error(err, "failed to generate nonce")
		return chunkStatus{addr: chunkAddr, status: statusErrored}
	}

	var (
		resp pb.Response
		req  = &pb.Challenge{Nonce: nonce, Address: chunkAddr.Bytes()}
	)
	s.metrics.CreateChallengeTime.Observe(time.Since(start).Seconds())

	err = s.sender.SendClosest(ctx, chunkAddr, req, &resp, false)
	if errors.Is(err, topology.ErrWantSelf) {
		has, err := s.storer.Has(ctx, chunkAddr)
		if err != nil {
			s.logger.Error(err, "pos: failed to check for chunk in local store: %v")
		} else if has {
			return chunkStatus{prover: s.address, addr: chunkAddr, status: statusProved}
		}
	} else if errors.Is(err, sender.ErrNotSent) {
		s.logger.Error(err, "failed to send challenge for chunk %v", chunkAddr)
	} else {
		s.metrics.TotalChallenges.Inc()
		if err != nil {
			s.logSendError(err)
		} else {
			prover, err := s.handleResponse(ctx, chunkAddr, &resp)
			if err == nil {
				s.logger.Info("pos: Obtained proof-of-storage for %v. Re-upload unnecessary.", chunkAddr)
				return chunkStatus{prover: prover, addr: chunkAddr, status: statusProved}
			} else {
				s.logger.Warning("pos: error handling response to challenge for chunk %v: %v", chunkAddr, err)
			}
		}
	}

	s.logger.Info("pos: Failed to obtain proof-of-storage for %v. Chunk will be re-uploaded.", chunkAddr)
	err = s.pushChunk(ctx, chunkAddr)
	if err != nil {
		s.logger.Warning("pos: Error re-uploading chunk %v: %v.", chunkAddr, err)
		return chunkStatus{prover: swarm.ZeroAddress, addr: chunkAddr, status: statusErrored}
	}

	s.logger.Info("pos: Chunk was re-uploaded: %v.", chunkAddr)
	return chunkStatus{prover: swarm.ZeroAddress, addr: chunkAddr, status: statusReuploaded}
}

func (s *Service) logSendError(err error) {
	var peerErr sender.Error
	// try to get the peer information from the error to give more descriptive logs
	if errors.As(err, &peerErr) {
		if err := peerErr.Unwrap(); errors.Is(err, io.EOF) {
			// EOF is expected when the recipient couldn't generate a proof.
			s.logger.Debug("pos: no response from %v", peerErr.Peer())
		} else {
			s.logger.Error(err, "pos: error receiving response from %v: %v", peerErr.Peer())
		}
	} else {
		s.logger.Error(err, "pos: error receiving response: %v")
	}
}

func (s *Service) pushChunk(ctx context.Context, chunkAddr swarm.Address) error {
	c, err := s.storer.Get(ctx, storage.ModeGetSync, chunkAddr)
	if err != nil {
		return err
	}

	_, err = s.push.PushChunkToClosest(ctx, c)
	if err != nil {
		if !errors.Is(err, topology.ErrWantSelf) {
			return err
		}
		// swallow the error in case we are the closest node
	}

	return nil
}

func (s *Service) generateNonce() (nonce []byte, err error) {
	var k [32]byte
	_, err = io.ReadFull(rand.Reader, k[:])
	if err != nil {
		return nil, err
	}
	nonce, err = crypto.LegacyKeccak256(k[:])
	if err != nil {
		return nil, fmt.Errorf("error generating hash of nonce key: %w", err)
	}
	return nonce, nil
}

func (s *Service) getChunkAddrs(ctx context.Context, root swarm.Address) (addrs []swarm.Address, err error) {
	err = s.traverser.Traverse(ctx, root, func(addr swarm.Address) error {
		has, err := s.storer.Has(ctx, addr)
		if err != nil {
			return fmt.Errorf("error checking store for chunk %v: %w", addr, err)
		}
		if !has {
			return fmt.Errorf("%w: %v", ErrMissingChunk, addr)
		}
		addrs = append(addrs, addr)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return addrs, nil
}

func (s *Service) createResponse(proof *pb.Proof) (resp *pb.Response, err error) {
	b, err := proof.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof: %w", err)
	}

	sig, err := s.signer.Sign(b)
	if err != nil {
		return nil, fmt.Errorf("failed to sign proof: %w", err)
	}

	return &pb.Response{
		Proof:     proof,
		Signature: sig,
		BlockHash: s.blockHash,
	}, nil
}

// handler handles incoming challenges.
func (s *Service) handler(ctx context.Context, p p2p.Peer, stream p2p.Stream) error {
	defer func() { _ = stream.FullClose() }()

	w, r := protobuf.NewWriterAndReader(stream)

	var (
		chall pb.Challenge
		resp  = new(pb.Response)
	)

	err := r.ReadMsgWithContext(ctx, &chall)
	if err != nil {
		return fmt.Errorf("pos: failed to read message: %w", err)
	}

	s.metrics.TotalChallengesRecv.Inc()

	chunkAddr := swarm.NewAddress(chall.GetAddress())
	s.logger.Debug("pos: received challenge for chunk %v from %v", chunkAddr, p.Address)

	// try to forward the challenge and receive a proof in response
	err = s.sender.SendClosest(ctx, chunkAddr, &chall, resp, true, p.Address)

	// sendChallenge may return ErrWantSelf, in which case we need to process the challenge.
	if errors.Is(err, topology.ErrWantSelf) {
		s.logger.Debug("pos: handling challenge for chunk %v from %v", chunkAddr, p.Address)

		if !*s.o.Enabled {
			_ = stream.Reset()
			return fmt.Errorf("POS is disabled. Enable using /debug/pos/enable or set --pos-enable: true.")
		}

		resp, err = s.processChallenge(ctx, &chall)
		if err != nil {
			return fmt.Errorf("pos: failed to process challenge: %w", err)
		}
	} else if errors.Is(err, sender.ErrNotSent) {
		return err
	} else {
		s.metrics.TotalForwarded.Inc()

		if err != nil {
			s.logSendError(err)
			return nil
		}
	}

	// finally, we return the response
	err = w.WriteMsgWithContext(ctx, resp)
	if err != nil {
		return fmt.Errorf("pos: failed to write message: %w", err)
	}

	return nil
}

func (s *Service) processChallenge(ctx context.Context, chall *pb.Challenge) (response *pb.Response, err error) {
	start := time.Now()

	var (
		hasher    = swarm.NewHasher()
		proofHash = make([]byte, hasher.Size())
	)

	addr := swarm.NewAddress(chall.GetAddress())

	hash, err := s.genChunkProof(ctx, addr, chall.GetNonce())
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			err = fmt.Errorf("pos: failed to create chunk proof: %w", err)
		}
		return nil, err
	}

	pubKey, err := s.signer.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("pos: failed to get public key: %w", err)
	}

	hasher.Reset()
	_, err = hasher.Write(crypto.EncodeSecp256k1PublicKey(pubKey))
	if err != nil {
		return nil, fmt.Errorf("pos: failed to hash public key: %w", err)
	}

	_, err = hasher.Write(hash)
	if err != nil {
		return nil, fmt.Errorf("pos: failed to hash proof: %w", err)
	}

	proofHash = hasher.Sum(proofHash[:0])

	proof := &pb.Proof{
		Challenge: chall,
		Hash:      proofHash,
	}
	s.metrics.ProveChunkTimeNoSignature.Observe(time.Since(start).Seconds())

	s.metrics.TotalProofsGenerated.Inc()

	msg, err := s.createResponse(proof)
	if err != nil {
		return nil, fmt.Errorf("pos: failed to sign proof: %w", err)
	}
	s.metrics.ProveChunkTime.Observe(time.Since(start).Seconds())

	return msg, nil
}

// handleResponse validates the response. Returns the address of the prover if the response is valid.
func (s *Service) handleResponse(ctx context.Context, chunkAddr swarm.Address, resp *pb.Response) (prover swarm.Address, err error) {
	start := time.Now()

	if !chunkAddr.Equal(swarm.NewAddress(resp.GetProof().GetChallenge().GetAddress())) {
		return swarm.ZeroAddress, fmt.Errorf("pos: proof address does not match challenge address")
	}

	// verify the signature
	data, err := resp.GetProof().Marshal()
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("failed to marshal proof for signature verification: %w", err)
	}
	pk, err := crypto.Recover(resp.GetSignature(), data)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("signature invalid: %w", err)
	}

	// verify the source address
	src, err := crypto.NewOverlayAddress(*pk, s.networkID, resp.GetBlockHash())
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("failed to obtain source address: %w", err)
	}

	// verify the proof itself
	err = s.verifyProof(ctx, chunkAddr, pk, resp.GetProof())
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("failed to verify proof: %w", err)
	}

	s.metrics.VerifyProofTimeNoSignature.Observe(time.Since(start).Seconds())

	// check that the response comes from a sufficient depth
	depth := s.topology.NeighborhoodDepth()
	po := swarm.Proximity(src.Bytes(), chunkAddr.Bytes())

	if po < depth {
		s.logger.Info(
			"handleResponse: response from %v for chunk %v too shallow",
			src,
			swarm.NewAddress(resp.Proof.GetChallenge().GetAddress()),
		)
		s.metrics.ShallowProofDepth.WithLabelValues(strconv.Itoa(int(po))).Inc()
		return swarm.ZeroAddress, ErrShallowResponse
	}

	s.metrics.VerifyProofTime.Observe(time.Since(start).Seconds())
	s.metrics.ProofDepth.WithLabelValues(strconv.Itoa(int(po))).Inc()

	return src, nil
}

// verifyProof verifies a proof.
func (s *Service) verifyProof(ctx context.Context, addr swarm.Address, pk *ecdsa.PublicKey, proof *pb.Proof) (err error) {
	myProof := make([]byte, swarm.HashSize)

	cp, err := s.genChunkProof(ctx, addr, proof.GetChallenge().GetNonce())
	if err != nil {
		return fmt.Errorf("failed to generate chunk proof: %w", err)
	}

	hasher := swarm.NewHasher()
	_, err = hasher.Write(crypto.EncodeSecp256k1PublicKey(pk))
	if err != nil {
		return fmt.Errorf("failed to hash public key: %w", err)
	}
	_, err = hasher.Write(cp)
	if err != nil {
		return fmt.Errorf("failed to hash proof: %w", err)
	}

	myProof = hasher.Sum(myProof[:0])
	if !bytes.Equal(myProof, proof.GetHash()) {
		return ErrInvalidProof
	}

	return nil
}

// genChunkProof generates the proof for the chunk at the given address.
func (s *Service) genChunkProof(ctx context.Context, addr swarm.Address, nonce []byte) ([]byte, error) {
	chunk, err := s.storer.Get(ctx, storage.ModeGetRequest, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	hasher := swarm.NewHasher()

	_, err = hasher.Write(nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to hash nonce: %w", err)
	}

	_, err = hasher.Write(chunk.Data())
	if err != nil {
		return nil, fmt.Errorf("failed to hash chunk data: %w", err)
	}

	hash := hasher.Sum(nil)

	return hash, nil
}
