package snips

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethersphere/bee/pkg/shed"
	"github.com/ethersphere/bee/pkg/snips/pb"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/opencoff/go-bbhash"
	"github.com/opencoff/go-fasthash"
)

// GetChunkProof returns the chunk proof for the given chunk and nonce.
// If the chunk proof is not cached, it will be calculated and cached.
func (s *Service) GetChunkProof(ctx context.Context, nonce uint64, chunk swarm.Chunk) (uint64, error) {
	s.mapCTPMu.Lock()
	defer s.mapCTPMu.Unlock()
	start := time.Now()
	if _, ok := s.mapChunkIdToProof[nonce]; !ok {
		s.mapChunkIdToProof[nonce] = make(map[string]uint64)
	}
	cpuint64, ok := s.mapChunkIdToProof[nonce][chunk.Address().ByteString()]
	if !ok {
		nonceByte := make([]byte, 8)
		binary.LittleEndian.PutUint64(nonceByte, nonce)
		startgenchunkproof := time.Now()
		chunkproof, err := genChunkProof(ctx, chunk.Data(), nonceByte)

		if err != nil {
			return cpuint64, err
		}
		s.metrics.GenChunkProofTime.Observe(time.Since(startgenchunkproof).Seconds())
		cpuint64 = HashByteToUint64(chunkproof)
		s.mapChunkIdToProof[nonce][chunk.Address().ByteString()] = cpuint64
	}
	s.metrics.GetChunkProofTime.Observe(time.Since(start).Seconds())
	return cpuint64, nil
}

// genChunkProof generates the chunk proof for the given data and nonce.
// A chunk proof is the hash of the concatenation of the data and the nonce.
// To avoid recalculating the same chunk proof, you can use s.GetChunkProof which will cache the chunk proofs.
func genChunkProof(ctx context.Context, chunkdata, nonce []byte) ([]byte, error) {
	hasher := swarm.NewHasher()
	_, err := hasher.Write(nonce)
	if err != nil {
		return nil, fmt.Errorf("snips proof: failed to hash nonce: %w", err)
	}

	_, err = hasher.Write(chunkdata)
	if err != nil {
		return nil, fmt.Errorf("snips proof: failed to hash chunk data: %w", err)
	}

	hash := hasher.Sum(nil)
	return hash, nil
}

// HashByteToUint64 hashes the byte array into uint64
func HashByteToUint64(cp []byte) uint64 {
	return fasthash.Hash64(seed, cp)
}

// BuildBaseProof takes a channel that feed in chunks.
// For each chunk it will calculate its chunk proof for the given nonce.
// Each chunk proof is cumulated into a list called the base proof. (Todo: find a better name)
func (s *Service) BuildBaseProof(ctx context.Context, chunkschan <-chan swarm.Chunk, doneChan <-chan struct{}, nonce uint64) ([]uint64, map[uint64]swarm.Address, error) {
	start := time.Now()
	baseProof := make([]uint64, 0)
	chunkProofToId := make(map[uint64]swarm.Address)
	for {
		select {
		case chunk := <-chunkschan:

			cpuint64, err := s.GetChunkProof(ctx, nonce, chunk)
			if err != nil {
				return nil, nil, err
			}
			baseProof = append(baseProof, cpuint64)
			chunkProofToId[cpuint64] = chunk.Address()

		case <-doneChan:
			s.metrics.BuildBaseProofTime.Observe(time.Since(start).Seconds())
			return baseProof, chunkProofToId, nil
		}
	}
}

// CreateMPHF attempts to create a valid MPHF for the given baseProof.
// For each attempt, the gamma value is incremented by 0.5, until the maximum.
// chunkProofToId is a temporary map that maps chunk proof to chunk id. This map is needed to construct mapValToChunkId.
func (s *Service) CreateMPHF(ctx context.Context, baseProof []uint64, chunkProofToId map[uint64]swarm.Address, nonce uint64) (*bbhash.BBHash, error) {
	start := time.Now()
	var bb *bbhash.BBHash
	var err error
	s.mapVTCMu.Lock()
	defer s.mapVTCMu.Unlock()
	for i := *s.o.MinGamma; i <= *s.o.MaxGamma; i++ {
	CONSTRUCTPROOF:
		bb, err = bbhash.NewSerial(i, baseProof)
		if err != nil {
			fmt.Printf("Error constructing BBHash: %v\n", err)
			continue
		}
		s.mapValToChunkId[nonce] = make(map[uint64]swarm.Address)

		for _, cp := range baseProof {
			val := bb.Find(cp)
			if _, ok := s.mapValToChunkId[nonce][val]; ok {
				i += 0.5
				goto CONSTRUCTPROOF
			}
			s.mapValToChunkId[nonce][val] = chunkProofToId[cp]
		}

		break // if we get here, we have a valid bbhash
	}
	if err != nil {
		return nil, err
	}
	s.metrics.CreateMPHFTime.Observe(time.Since(start).Seconds())

	return bb, nil
}

// CreateProofForMyChunks creates a SNIPS proof for all chunks in the storage radius.
func (s *Service) CreateProofForMyChunks(ctx context.Context, nonce uint64) (*pb.Proof, error) {
	start := time.Now()
	nonceByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(nonceByte, nonce)
	chunksChan, doneChan := make(chan swarm.Chunk), make(chan struct{})
	var minRange, maxRange []byte
	s.GetChunksInMyRadius(ctx, chunksChan, doneChan, &minRange, &maxRange)

	baseproof, chunkProofToId, err := s.BuildBaseProof(ctx, chunksChan, doneChan, nonce)
	if err != nil {
		return nil, err
	}
	mphf, err := s.CreateMPHF(ctx, baseproof, chunkProofToId, nonce)
	if err != nil {
		return nil, err
	}
	s.metrics.CreateProofTime.Observe(time.Since(start).Seconds())

	buf := new(bytes.Buffer)
	err = mphf.MarshalBinary(buf)
	if err != nil {
		return nil, err
	}

	s.metrics.TotalProofsGenerated.Inc()
	proof := &pb.Proof{
		Begin:  minRange,
		End:    maxRange,
		Nonce:  nonce,
		MPHF:   buf.Bytes(),
		Length: uint32(len(baseproof)),
	}

	return proof, nil
}

// FindMissingChunks takes a SNIPS proof that another peer sent to us and queries with all chunks that we have.
// To limit the chunks queried, we employ a begin and end range filter.
// The function returns the list of values that we are missing from the proof.
// The creator of the proof will be able to use this list to return missing chunks to us.
func (s *Service) FindMissingChunks(ctx context.Context, proof *bbhash.BBHash, nonce uint64, length uint32, begin, end []byte) (expectedVals, error) {
	start := time.Now()
	myExpectedChunks := NewExpectedVals(int(length))
	err := s.db.IterStoredChunks(ctx, true, func(item shed.Item) error {
		chunk := swarm.NewChunk(swarm.NewAddress(item.Address), item.Data)
		distBegin := bytes.Compare(begin, chunk.Address().Bytes()) // Must be -1 or 0
		distEnd := bytes.Compare(chunk.Address().Bytes(), end)     // Must be -1 or 0
		if distBegin == 1 || distEnd == 1 {
			return nil // The chunk address is outside the range.
		}

		chunkproof, err := s.GetChunkProof(ctx, nonce, chunk)
		if err != nil {
			return err
		}

		hashVal := proof.Find(chunkproof)
		if hashVal != 0 {
			myExpectedChunks.Hit(hashVal)
		}

		return nil
	})
	s.metrics.FindMissingChunksTime.Observe(time.Since(start).Seconds())
	return myExpectedChunks, err
}

// GetMissingChunks is used when another peer replies with a list of missing values for a SNIPS proof.
// This function will take the missing values as input and return the corresponding chunks.
// Note that we take the nonce that was attached to the SNIPS proof as input to verify that we created the proof.
func (s *Service) GetMissingChunks(ctx context.Context, missingVals []uint64, nonce uint64) ([]swarm.Chunk, error) {
	s.mapVTCMu.Lock()
	defer s.mapVTCMu.Unlock()
	start := time.Now()
	if _, ok := s.mapValToChunkId[nonce]; !ok {
		s.metrics.TotalErrored.Inc()
		return nil, fmt.Errorf("snips: GetMissingChunks: No proof found for nonce (I was not the creator of the proof!)")
	}

	chunks := make([]swarm.Chunk, len(missingVals))
	for i, val := range missingVals {
		chunkid, ok := s.mapValToChunkId[nonce][val]
		if !ok {
			return nil, fmt.Errorf("snips: GetMissingChunks: Missing chunk not found")
		}
		chunk, err := s.db.Get(ctx, storage.ModeGetRequest, chunkid)
		if err != nil {
			return nil, err
		}
		chunks[i] = chunk
	}
	s.metrics.GetMissingChunksTime.Observe(time.Since(start).Seconds())
	return chunks, nil
}
