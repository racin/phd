package snips

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/localstore"
	"github.com/ethersphere/bee/pkg/log"
	"github.com/ethersphere/bee/pkg/p2p"
	"github.com/ethersphere/bee/pkg/p2p/protobuf"
	"github.com/ethersphere/bee/pkg/postage"
	"github.com/ethersphere/bee/pkg/pushsync"
	pushpb "github.com/ethersphere/bee/pkg/pushsync/pb"
	"github.com/ethersphere/bee/pkg/shed"
	"github.com/ethersphere/bee/pkg/snips/pb"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/bee/pkg/topology"
	"github.com/ethersphere/bee/pkg/transaction"
	"github.com/hashicorp/go-multierror"
	"github.com/opencoff/go-bbhash"
)

const (
	protocolName             = "snips"
	protocolVersion          = "1.0.0"
	streamNameSNIPSproof     = "snips-proof"
	streamNameSNIPSmaintain  = "snips-maintain"
	streamNameSNIPSrequest   = "snips-request"
	streamNameSNIPSneighbors = "snips-neighbors"

	seed               = 1149799105110
	gamma              = 2
	concurrentPushes   = 50 // how many chunks to push simultaneously
	concurrentMaintain = 5  // how many maintain to process simultaneously
)

type GetHasser interface {
	storage.Getter
	storage.Hasser
}

type Interface interface {
	CreateAndSendProofForMyChunks(ctx context.Context, nonce uint64) error
	GetNeighborInfo(ctx context.Context) [][]byte
}

type Options struct {
	Enabled            *bool
	MinGamma           *float64
	MaxGamma           *float64
	BlockInterval      *uint64 // How many blocks to wait before sending a new SNIPS proof.
	BlockCheckInterval *time.Duration
}

func snipsDefaultOptions() Options {
	t, blockcheckinterval := true, 30*time.Second
	var mingamma, maxgamma float64 = 2, 3
	var blockinterval uint64 = 7200
	return Options{
		Enabled:            &t,
		MinGamma:           &mingamma,
		MaxGamma:           &maxgamma,
		BlockInterval:      &blockinterval,
		BlockCheckInterval: &blockcheckinterval,
	}
}

type Service struct {
	// components:
	backend      transaction.Backend
	db           *localstore.DB
	reserveState postage.ReserveStateGetter
	logger       log.Logger
	signer       crypto.Signer
	streamer     p2p.Streamer
	topology     topology.Driver

	o Options

	metrics Metrics

	// data:

	address   swarm.Address
	blockHash []byte // Needed for other peers to verify my address.
	networkID uint64

	// snips
	mapChunkIdToProof map[uint64]map[string]uint64
	mapCTPMu          sync.Mutex
	mapValToChunkId   map[uint64]map[uint64]swarm.Address
	mapVTCMu          sync.Mutex
	quit              chan struct{}
	sem               chan struct{}
	semmaintain       chan struct{}
}

func New(swarmaddress swarm.Address, blockHash []byte, networkID uint64, logger log.Logger, reserveState postage.ReserveStateGetter, signer crypto.Signer, db *localstore.DB, streamer p2p.Streamer, topology topology.Driver, backend transaction.Backend, opts Options) *Service {
	return &Service{
		logger:   logger,
		signer:   signer,
		db:       db,
		topology: topology,
		backend:  backend,

		metrics: newMetrics(),

		address:           swarmaddress, // Overlay
		blockHash:         blockHash,
		networkID:         networkID,
		mapChunkIdToProof: make(map[uint64]map[string]uint64),
		mapValToChunkId:   make(map[uint64]map[uint64]swarm.Address),
		streamer:          streamer,
		reserveState:      reserveState,
		quit:              make(chan struct{}),
		sem:               make(chan struct{}, concurrentPushes),
		semmaintain:       make(chan struct{}, concurrentMaintain),

		o: opts,
	}
}

func (s *Service) AddStreamer(streamer p2p.Streamer) {
	s.streamer = streamer
}

func (s *Service) Start(ctx context.Context) {
	// Worker to wait for new blocks and send SNIPS proofs.
	go s.snipsWorker(context.Background())
}

func (s *Service) Protocol() p2p.ProtocolSpec {
	return p2p.ProtocolSpec{
		Name:    protocolName,
		Version: protocolVersion,
		StreamSpecs: []p2p.StreamSpec{
			{
				Name:    streamNameSNIPSproof,
				Handler: s.handlerIncomingProof,
			},
			{
				Name:    streamNameSNIPSmaintain,
				Handler: s.handlerIncomingMaintain,
			},
			{
				Name:    streamNameSNIPSrequest,
				Handler: s.handlerIncomingRequest,
			},
			{
				Name:    streamNameSNIPSneighbors,
				Handler: s.handlerIncomingNeighbors,
			},
		},
	}
}

// uploadMissingChunks handles a Maintain response and uploads the chunks that are missing (requested).
func (s *Service) uploadMissingChunks(ctx context.Context, addr swarm.Address, maintain *pb.Maintain) error {
	start := time.Now()
	buf := bytes.NewBuffer(maintain.BitVector)
	bv, err := MissingFromBytes(buf)
	if err != nil {
		return fmt.Errorf("snips: failed to unmarshal bitvector: %w", err)
	}

	chunks, err := s.GetMissingChunks(ctx, bv, maintain.Nonce)
	if err != nil {
		return fmt.Errorf("snips: failed to find missing chunks: %w", err)
	}

	var allerrors error
	for i := 0; i < len(chunks); i++ {
		s.sem <- struct{}{}
		chunk := chunks[i]
		stamp, err := chunk.Stamp().MarshalBinary()
		// s.logger.Debug(fmt.Sprintf("Sending chunk %d/%d to %s. Chunk: %s", i+1, len(chunks), addr, chunk.Address()))
		if err != nil {
			fmt.Printf("Failed to marshal stamp: %v\n", err)
			return err
		}
		msg := &pushpb.Delivery{
			Address: chunk.Address().Bytes(),
			Data:    chunk.Data(),
			Stamp:   stamp,
		}

		pushstream, err := s.streamer.NewStream(ctx, addr, nil, pushsync.GetProtocolName(), pushsync.GetProtocolVersion(), pushsync.StreamNamePushsyncSNIPS)
		if err != nil {
			allerrors = multierror.Append(allerrors, fmt.Errorf("uploadMissingChunks: new stream for peer with addr %s: %w", addr, err))
		}

		pushwriter, pushreader := protobuf.NewWriterAndReader(pushstream)

		if err = pushwriter.WriteMsgWithContext(ctx, msg); err != nil {
			fmt.Printf("Failed to WriteMsgWithContext: %v\n", err)
			allerrors = multierror.Append(allerrors, err)
		}

		var receipt pushpb.Receipt
		if err = pushreader.ReadMsgWithContext(ctx, &receipt); err != nil {
			fmt.Printf("Failed to ReadMsgWithContext: %v\n", err)
			allerrors = multierror.Append(allerrors, err)
		}

		_ = pushstream.Close()
		s.metrics.TotalChunksUploaded.Inc()
		<-s.sem
	}
	s.metrics.UploadMissingChunksTime.Observe(time.Since(start).Seconds())
	return allerrors
}

// handlerIncomingProof handles incoming SNIPS proofs.
func (s *Service) handlerIncomingProof(ctx context.Context, p p2p.Peer, stream p2p.Stream) error {
	defer func() { _ = stream.Close() }()
	if !*s.o.Enabled {
		_ = stream.Reset()
		return fmt.Errorf("SNIPS is disabled. Enable using /debug/snips/enable or set --snips-enable: true")
	}
	r := protobuf.NewReader(stream)

	var signedproof pb.SignedProof
	if err := r.ReadMsgWithContext(ctx, &signedproof); err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: failed to read SNIPS proof:")
		return fmt.Errorf("snips: handlerIncomingProof: failed to read message: %w", err)
	}

	// Verify the signature of the proof and ensure that the creator is within our neighborhood.
	creator, err := s.verifySignature(ctx, signedproof)
	if err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: failed to verify SNIPS proof signature:")
		return fmt.Errorf("snips: handlerIncomingProof: failed to verify signature: %w", err)
	}

	// We do not need to verify that the creator of the proof is the same peer that sent us the proof. (Peers might relay messages)
	if !bytes.Equal(creator.Bytes(), p.Address.Bytes()) {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: SNIPS proof creator and sender did not match.", "Sender", p.Address, "Creator", creator)
		return fmt.Errorf("snips: SNIPS proof creator and sender did not match :%w. Sender: %s, Creator: %s", err, p.Address, creator)
	}

	mphf, err := bbhash.UnmarshalBBHash(bytes.NewReader(signedproof.Proof.MPHF))
	if err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: failed to unmarshal mphf")
		return fmt.Errorf("snips: failed to unmarshal mphf: %w", err)
	}
	// Metrics
	s.metrics.TotalSNIPSproofRecv.Inc()
	s.metrics.TotalChunksInProof.Add(float64(signedproof.Proof.Length))
	s.metrics.SizeOfSNIPSsignedproof.WithLabelValues(fmt.Sprint(signedproof.Proof.Nonce)).Add(float64(signedproof.Proof.Size()))
	s.metrics.SizeOfSNIPSmphf.WithLabelValues(fmt.Sprint(signedproof.Proof.Nonce)).Add(float64(mphf.MarshalBinarySize()))

	newproof, err := s.RetrieveMissingChunks(ctx, p, mphf, signedproof.Proof)
	if err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: failed to retrieve missing chunks")
	} else if newproof {
		// Racin: Disabled for TestSimilarityPerformance
		request := &pb.Request{
			Nonce: signedproof.Proof.Nonce,
		}
		// fmt.Printf("Sending request for new proof to %s. req: %+v\n", p.Address, request)
		reqstream, err := s.streamer.NewStream(ctx, p.Address, nil, protocolName, protocolVersion, streamNameSNIPSrequest)
		// defer func() { _ = reqstream.Close() }()
		if err != nil {
			s.metrics.TotalErrored.Inc()
			s.logger.Error(err, "handlerIncomingProof: send request: new stream for peer with addr %s", p.Address)
			return fmt.Errorf("handlerIncomingProof: send request: new stream for peer with addr %s: %w", p.Address, err)
		}
		mwriter := protobuf.NewWriter(reqstream)
		s.metrics.TotalSNIPSreqSent.Inc()
		err = mwriter.WriteMsgWithContext(ctx, request)
		if err != nil {
			s.metrics.TotalErrored.Inc()
			s.logger.Error(err, "handlerIncomingProof: failed to write message")
			return fmt.Errorf("handlerIncomingProof: failed to write message: %w", err)
		}
	}

	return err
}

func (s *Service) verifySignature(ctx context.Context, proof pb.SignedProof) (swarm.Address, error) {
	start := time.Now()

	// verify the signature
	data, err := proof.Proof.Marshal()
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("verifySignature: failed to marshal proof for signature verification: %w", err)
	}
	pk, err := crypto.Recover(proof.Signature, data)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("verifySignature: signature invalid: %w", err)
	}

	// verify the source address
	src, err := crypto.NewOverlayAddress(*pk, s.networkID, proof.BlockHash)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("verifySignature failed to obtain source address: %w", err)
	}

	s.metrics.verifySignatureTime.Observe(time.Since(start).Seconds())

	// check that the response comes from a sufficient depth
	depth := s.topology.NeighborhoodDepth()
	po := swarm.Proximity(src.Bytes(), s.address.Bytes())
	if po < depth {
		s.logger.Error(err, "verifySignature: source address is not within the neighborhood depth %d", "Source", src, "Dest", s.address, "Depth", depth, "PO", po)
		s.metrics.ShallowProofDepth.WithLabelValues(strconv.Itoa(int(po))).Inc()
		return swarm.ZeroAddress, fmt.Errorf("verifySignature: Proving peer is outside our neighborhood. Ignore it. PO: %d, Depth: %d, Peer: %s", po, depth, src)
	}

	return src, nil
}

// handlerIncomingMaintain
// Does the following:
// 1) Collects the Maintain response.
// 2) Upload the missing chunks to the peer.
// 3) Sends an acknowledgement when the upload is complete.
func (s *Service) handlerIncomingMaintain(ctx context.Context, p p2p.Peer, stream p2p.Stream) error {
	defer func() { _ = stream.Close() }()
	if !*s.o.Enabled {
		_ = stream.Reset()
		return fmt.Errorf("SNIPS is disabled. Enable using /debug/snips/enable or set --snips-enable: true")
	}

	s.metrics.TotalSNIPSmaintainRecv.Inc()
	w, r := protobuf.NewWriterAndReader(stream)

	var maintain pb.Maintain
	if err := r.ReadMsgWithContext(ctx, &maintain); err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: StartSNIPS: failed to read maintain response from peer %v", p.Address.ByteString())
		return fmt.Errorf("snips: handlerIncomingMaintain: failed to read message: %w", err)
	}

	s.semmaintain <- struct{}{}
	if err := s.uploadMissingChunks(ctx, p.Address, &maintain); err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: handlerIncomingMaintain: failed to upload chunks", "to_peer", p.Address.ByteString())
	}

	if err := w.WriteMsgWithContext(ctx, &pb.UploadDone{}); err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: handlerIncomingMaintain: failed to write upload status to peer %v", p.Address.ByteString())
	}
	<-s.semmaintain
	s.metrics.TotalSNIPSuploaddoneSent.Inc()

	return nil
}

// handlerIncomingRequest returns a SNIPS proof my chunks for the given nonce.
// Used when a peer discovers duplicates and requests a new proof to clear them.
func (s *Service) handlerIncomingRequest(ctx context.Context, p p2p.Peer, stream p2p.Stream) error {
	defer func() { _ = stream.Close() }()
	if !*s.o.Enabled {
		_ = stream.Reset()
		return fmt.Errorf("SNIPS is disabled. Enable using /debug/snips/enable or set --snips-enable: true")
	}

	s.metrics.TotalSNIPSreqRecv.Inc()
	r := protobuf.NewReader(stream)

	var request pb.Request

	if err := r.ReadMsgWithContext(ctx, &request); err != nil {
		s.logger.Error(err, "snips: handlerIncomingRequest: failed to read message")
		return fmt.Errorf("snips: handlerIncomingRequest: failed to read message: %w", err)
	}
	proof, err := s.CreateProofForMyChunks(ctx, request.Nonce)
	if err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: handlerIncomingRequest: failed to create proof")
		return fmt.Errorf("snips: handlerIncomingRequest: failed to create proof: %w", err)
	}
	signedproof, err := s.createResponse(proof)
	if err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: handlerIncomingRequest : failed to sign proof")
		return fmt.Errorf("snips: handlerIncomingRequest : failed to sign proof: %w", err)
	}
	if err := s.SendProof(ctx, p.Address, signedproof); err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: handlerIncomingRequest: failed to send proof")
		return fmt.Errorf("snips: handlerIncomingRequest: failed to send proof: %w", err)
	}
	return nil
}

type procProofState struct {
	Sender       string
	Recipient    string
	StoredChunks int
	Missing      int
	Collisions   int
}

// RetrieveMissingChunks enters a loop which will terminate when there are no missing chunks.
// The loop does the following:
// 1) Finds missing chunks. 2) Send a Maintain request to the proving peer.
// 3) After receiving an acknowledgement, it will repeat from step 1)
func (s *Service) RetrieveMissingChunks(ctx context.Context, p p2p.Peer, mphf *bbhash.BBHash, proof *pb.Proof) (bool, error) {
	start := time.Now()
	var expected expectedVals
	var err error

	expected, err = s.FindMissingChunks(ctx, mphf, proof.Nonce, proof.Length, proof.Begin, proof.End)
	if err != nil {
		return false, fmt.Errorf("snips: failed to find missing chunks: %w", err)
	}
	missing := expected.GetMissingChunks()
	if len(missing) == 0 {
		return false, nil // Nothing missing - nothing to request.
	}

	missingBitVector := missing.AsBitVector()
	mbvbytes, err := missingBitVector.ToBytes()
	if err != nil {
		return false, fmt.Errorf("snips: failed to convert missing bit vector to bytes: %w", err)
	}

	maintain := &pb.Maintain{
		Nonce:     proof.Nonce,
		BitVector: mbvbytes,
	}
	mstream, err := s.streamer.NewStream(ctx, p.Address, nil, protocolName, protocolVersion, streamNameSNIPSmaintain)
	defer func() { _ = mstream.Close() }()
	if err != nil {
		return false, fmt.Errorf("RetrieveMissingChunks: new stream for peer with addr %s: %w", p.Address, err)
	}
	mwriter, mreader := protobuf.NewWriterAndReader(mstream)
	err = mwriter.WriteMsgWithContext(ctx, maintain)
	if err != nil {
		return false, fmt.Errorf("RetrieveMissingChunks: failed to write message: %w", err)
	}

	s.metrics.TotalSNIPSmaintainSent.Inc()
	s.metrics.TotalChunksInMaintain.Add(float64(len(missing)))
	s.metrics.SizeOfSNIPSsentMaintainBV.Add(float64(len(mbvbytes)))
	var uploadstatus pb.UploadDone
	if err := mreader.ReadMsgWithContext(ctx, &uploadstatus); err != nil {
		s.logger.Error(err, "RetrieveMissingChunks: StartSNIPS: failed to read uploadstatus %v", p.Address.ByteString())
	}
	s.metrics.TotalSNIPSuploaddoneRecv.Inc()

	s.metrics.RetriveMissingChunksTime.Observe(time.Since(start).Seconds())
	return expected.GotMissingAndDuplicate(), nil
}

// GetProofMapping is a helper function used for the tests.
// returns s.mapValToChunkId, s.mapChunkIdToProof
func (s *Service) GetProofMapping() (map[uint64]map[uint64]swarm.Address, map[uint64]map[string]uint64) {
	return s.mapValToChunkId, s.mapChunkIdToProof
}

// WatchForNewBlock is a worker that watches for new blocks published on the blockchain.
// For each s.o.BlockInterval-th block, it will write its block hash to the returned channel.
func (s *Service) WatchForNewBlock(ctx context.Context, done chan struct{}) (chan []byte, error) {
	var lastBlock uint64
	blockNumber := make(chan []byte)
	go func() {
		defer close(done)
		for {
			select {
			case <-done:
				return
			case <-time.After(*s.o.BlockCheckInterval):
			}
			curBlock, err := s.backend.BlockNumber(ctx)
			if err != nil {
				s.logger.Error(err, "snips: failed to get current block number")
				continue
			}

			if curBlock >= lastBlock+*s.o.BlockInterval {
				lastBlock = curBlock
				provingBlockNr := curBlock - (curBlock % *s.o.BlockInterval) // Each s.o.BlockInterval-th block.
				header, err := s.backend.HeaderByNumber(ctx, big.NewInt(int64(provingBlockNr)))
				if err != nil {
					s.logger.Error(err, "snips: failed to get header for block %v", provingBlockNr)
				}
				blockNumber <- header.Root.Bytes()
			}
		}
	}()
	return blockNumber, nil
}

// snipsWorker watches for newly published blocks on the blockchain.
// For a given interval of blocks, given by s.o.BlockInterval, it will generate a SNIPS proof.
// The SNIPS proof is sent to all neighours (In order deepest (closest) to shallowest (farthest)).
func (s *Service) snipsWorker(ctx context.Context) error {
	doneChan := make(chan struct{})
	defer func() { doneChan <- struct{}{} }()
	blockChan, err := s.WatchForNewBlock(ctx, doneChan)
	if err != nil {
		s.logger.Error(err, "snips: snipsWorker : failed to start block watcher")
	}
	for {
		select {
		case <-s.quit:
			return nil
		case blockNum := <-blockChan:
			blockNumU64 := HashByteToUint64(blockNum)
			if err := s.CreateAndSendProofForMyChunks(ctx, blockNumU64); err != nil {
				s.logger.Error(err, "snips: snipsWorker: failed to create and send proof for my chunks")
			}
		}
	}
}

// CreateAndSendProofForMyChunks creates a SNIPS proof for all chunks that are stored on this node.
// Then it sends the proof to all neighbours.
func (s *Service) CreateAndSendProofForMyChunks(ctx context.Context, nonce uint64) error {
	s.metrics.TotalCreateAndSendProofForMyChunks.Inc()
	proof, err := s.CreateProofForMyChunks(ctx, nonce)
	if err != nil {
		s.logger.Error(err, "snips: snipsWorker : failed to create proof")
		return err
	}
	signedproof, err := s.createResponse(proof)
	if err != nil {
		s.logger.Error(err, "snips: snipsWorker : failed to sign proof")
		return err
	}
	s.metrics.SizeOfSNIPSsentbaseproof.WithLabelValues(fmt.Sprint(nonce)).Add(float64(proof.Length))
	peerFunc := s.PeerFunc(ctx, signedproof)
	return s.topology.EachNeighbor(peerFunc)
}

// SendProof sends a SNIPS proof to another peer.
// The stream is only closed on the receiver side, as closing here caused issues when testing.
func (s *Service) SendProof(ctx context.Context, addr swarm.Address, proof *pb.SignedProof) error {
	if proof.Proof.Length == 0 {
		s.logger.Debug("snips: SendProof: nothing to send.")
		return nil // Nothing to send. Not an error.
	}
	snipsstream, err := s.streamer.NewStream(ctx, addr, nil, protocolName, protocolVersion, streamNameSNIPSproof)
	// defer func() { _ = snipsstream.FullClose() }() // The stream is closed on the other side.

	if err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: failed to open stream to peer %v", addr)
		return fmt.Errorf("snips: failed to create stream: %w", err)
	}
	snipswriter := protobuf.NewWriter(snipsstream)

	s.metrics.TotalSNIPSproofSent.Inc()
	s.metrics.SizeOfSNIPSsentMPHF.Add(float64(len(proof.Proof.MPHF)))
	s.metrics.SizeOfSNIPSsentBlockhash.Add(float64(len(proof.BlockHash)))
	s.metrics.SizeOfSNIPSsentSignature.Add(float64(len(proof.Signature)))

	if err := snipswriter.WriteMsgWithContext(ctx, proof); err != nil {
		s.metrics.TotalErrored.Inc()
		s.logger.Error(err, "snips: SendProof: failed to write proof to peer %v", addr)
	}

	return nil
}

// PeerFunc returns a function that should be called once per peer in your neighborhood.
// The function does the following:
func (s *Service) PeerFunc(ctx context.Context, proof *pb.SignedProof) func(swarm.Address, uint8) (stop bool, jumpToNext bool, err error) {
	return func(addr swarm.Address, po uint8) (stop bool, jumpToNext bool, err error) {
		if err := s.SendProof(ctx, addr, proof); err != nil {
			s.logger.Error(err, "snips: PeerFunc: failed to send proof to peer %v", addr)
		}
		return false, false, nil
	}
}

// GetChunksInMyRadius will write all chunks that are in my storage radius to the given channel.
// The minimum and maximum range are returned to the called.
func (s *Service) GetChunksInMyRadius(ctx context.Context, myChunksChan chan<- swarm.Chunk, doneChan chan<- struct{}, minRange, maxRange *[]byte) {
	go func() {
		err := s.db.IterStoredChunks(ctx, true, func(item shed.Item) error {
			chunk := swarm.NewChunk(swarm.NewAddress(item.Address), item.Data)
			chunkAddr := chunk.Address().Bytes()

			// If the Proximity Order is below the storage radius, we can ignore the chunk.
			if swarm.Proximity(s.address.Bytes(), chunkAddr) < s.topology.NeighborhoodDepth() { // s.reserveState.GetReserveState().StorageRadius {
				return nil
			}
			if len(*minRange) == 0 || bytes.Compare(chunkAddr, *minRange) == -1 {
				*minRange = chunkAddr
			}
			if len(*maxRange) == 0 || bytes.Compare(*maxRange, chunkAddr) == -1 {
				*maxRange = chunkAddr
			}

			myChunksChan <- chunk
			return nil
		})
		if err != nil {
			ctx.Err()
		}
		doneChan <- struct{}{}
	}()
}

func (s *Service) createResponse(proof *pb.Proof) (resp *pb.SignedProof, err error) {
	b, err := proof.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proof: %w", err)
	}

	sig, err := s.signer.Sign(b)
	if err != nil {
		return nil, fmt.Errorf("failed to sign proof: %w", err)
	}

	return &pb.SignedProof{
		Proof:     proof,
		Signature: sig,
		BlockHash: s.blockHash,
	}, nil
}

func (s *Service) Close() {
	s.quit <- struct{}{}
	close(s.quit)
}
