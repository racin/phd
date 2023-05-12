package snips_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	accountingmock "github.com/ethersphere/bee/pkg/accounting/mock"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/localstore"
	"github.com/ethersphere/bee/pkg/log"
	"github.com/ethersphere/bee/pkg/p2p/streamtest"
	"github.com/ethersphere/bee/pkg/postage"
	pricermock "github.com/ethersphere/bee/pkg/pricer/mock"
	"github.com/ethersphere/bee/pkg/pushsync"
	"github.com/ethersphere/bee/pkg/snips"
	statestore "github.com/ethersphere/bee/pkg/statestore/mock"
	"github.com/ethersphere/bee/pkg/storage"
	testingc "github.com/ethersphere/bee/pkg/storage/testing"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/bee/pkg/tags"
	"github.com/ethersphere/bee/pkg/topology"
	mockTopology "github.com/ethersphere/bee/pkg/topology/mock"
	"github.com/ethersphere/bee/pkg/transaction"
	"github.com/ethersphere/bee/pkg/transaction/backendmock"
)

// Todo: test metrics "github.com/prometheus/client_golang/prometheus/testutil"
func Test_GetChunksInMyRadius(t *testing.T) {
	numChunks := 10000
	myChunks := make([]swarm.Chunk, numChunks)
	for i := 0; i < numChunks; i++ {
		myChunks[i] = testingc.GenerateTestRandomChunk()
	}
	// Storageradius 8 required for this test. (Depth now...)
	snipsService, db, _ := createMockService(t, swarm.MustParseHexAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), nil, 8, 5, nil, nil, nil, []mockTopology.Option{mockTopology.WithNeighborhoodDepth(8)})
	db.Put(context.Background(), storage.ModePutUpload, myChunks...)

	chunksChan, doneChan := make(chan swarm.Chunk), make(chan struct{})
	var minRange, maxRange []byte
	wantMinRange, wantMaxRange := swarm.MustParseHexAddress("aa00").Bytes(), swarm.MustParseHexAddress("ab00").Bytes()

	snipsService.GetChunksInMyRadius(context.TODO(), chunksChan, doneChan, &minRange, &maxRange)
	for {
		select {
		case chunk := <-chunksChan:
			if bytes.Compare(chunk.Address().Bytes(), wantMinRange) == -1 {
				t.Fatalf("Chunk address is under minrange. Got %v, want minrange %v", chunk.Address().Bytes(), wantMinRange)
			}
			if bytes.Compare(wantMaxRange, chunk.Address().Bytes()) == -1 {
				t.Fatalf("Chunk address is over maxrange. Got %v, want maxrange %v", chunk.Address().Bytes(), wantMaxRange)
			}
		case <-doneChan:
			goto PRINTOUT
		}
	}
PRINTOUT:
	if bytes.Compare(minRange, wantMinRange) == -1 {
		t.Fatalf("Minrange is under the wanted minrange. Got %v, want minrange %v", minRange, wantMinRange)
	}
	if bytes.Compare(wantMaxRange, maxRange) == -1 {
		t.Fatalf("Maxrange is over the wanted maxrange. Got %v, want maxrange %v", maxRange, wantMaxRange)
	}
}
func Test_WatchForNewBlock(t *testing.T) {
	blockNumber := uint64(1)
	be := backendmock.New(
		backendmock.WithBlockNumberFunc(func(c context.Context) (uint64, error) {
			blockNumber++
			return blockNumber, nil
		}),
		backendmock.WithHeaderbyNumberFunc(func(c context.Context, number *big.Int) (*types.Header, error) {
			return &types.Header{
				Root: common.BigToHash(number),
			}, nil
		}))

	sendChunkPivot := func(ctx context.Context, c swarm.Chunk) (*pushsync.Receipt, error) {
		panic("unexpected push")
	}
	otherPeer := swarm.MustParseHexAddress("0000000000000000000000000000000000000000000000000000000000000000") // base is 0000
	myPeer := swarm.MustParseHexAddress("6000000000000000000000000000000000000000000000000000000000000000")    // binary 0110 -> po 1
	recorder := streamtest.New(streamtest.WithBaseAddr(otherPeer))

	snipsService, _, _ := createMockService(t, myPeer, nil, 8, 5, recorder, sendChunkPivot, be, []mockTopology.Option{mockTopology.WithClosestPeerErr(topology.ErrWantSelf), mockTopology.WithPeers(otherPeer)})

	doneChan := make(chan struct{})
	blockchan, err := snipsService.WatchForNewBlock(context.Background(), doneChan)
	if err != nil {
		t.Fatal(err)
	}

	gotBlocks := 0
	lastBlockID := 0
	for {
		if gotBlocks == 5 {
			doneChan <- struct{}{}
			break
		}
		lastBlockID = int(common.TrimLeftZeroes(<-blockchan)[0])
		gotBlocks++
	}

	if gotBlocks != 5 {
		t.Errorf("incorrect number of blocks received. got %d blocks, want %d", gotBlocks, 5)
	}
	if lastBlockID != 25 {
		t.Errorf("incorrect last block id. got last block %d, want %d", lastBlockID, 25)
	}
}

func TestProtocol(t *testing.T) {

	blockNumber := uint64(1)
	blockMutex := new(sync.Mutex)
	beA := backendmock.New(
		backendmock.WithBlockNumberFunc(func(c context.Context) (uint64, error) {
			blockMutex.Lock()
			defer blockMutex.Unlock()
			return blockNumber, nil
		}),
		backendmock.WithHeaderbyNumberFunc(func(c context.Context, number *big.Int) (*types.Header, error) {
			return &types.Header{
				Root: common.BigToHash(number),
			}, nil
		}))
	beB := backendmock.New(
		backendmock.WithBlockNumberFunc(func(c context.Context) (uint64, error) {
			blockMutex.Lock()
			defer blockMutex.Unlock()
			return blockNumber, nil
		}),
		backendmock.WithHeaderbyNumberFunc(func(c context.Context, number *big.Int) (*types.Header, error) {
			return &types.Header{
				Root: common.BigToHash(number),
			}, nil
		}))
	go func() {
		for {
			<-time.After(500 * time.Millisecond)
			blockMutex.Lock()
			blockNumber++
			blockMutex.Unlock()
		}
	}()
	peerA, peerAprivkey, err := generateSwarmAddress()
	if err != nil {
		t.Fatal(err)
	}
	// peerA := swarm.MustParseHexAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	// peerB := swarm.MustParseHexAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	peerB, peerBprivkey, err := generateSwarmAddress()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		numChunksA int
		numChunksB int
		similarity int // How many chunks are the same between A and B. 0 means all chunks are unique. 100 means all chunks are the same. To have have only 1 peer with unique chunks, put similarity to 100 and give set numChunksA larger than numChunksB.
		wait       int // The maximum amount of seconds to wait for this test to complete.
	}{
		{numChunksA: 100, numChunksB: 100, similarity: 0, wait: 9},
		{numChunksA: 100, numChunksB: 100, similarity: 100, wait: 6},
		{numChunksA: 100, numChunksB: 100, similarity: 50, wait: 6},
		{numChunksA: 250, numChunksB: 300, similarity: 90, wait: 6},
		{numChunksA: 2000, numChunksB: 0, similarity: 0, wait: 12},
		{numChunksA: 2000, numChunksB: 50, similarity: 80, wait: 12},
		{numChunksA: 2500, numChunksB: 3000, similarity: 90, wait: 12},
		{numChunksA: 5000, numChunksB: 4700, similarity: 0, wait: 12},
		{numChunksA: 5000, numChunksB: 4700, similarity: 100, wait: 12},
		{numChunksA: 300, numChunksB: 300, similarity: 0, wait: 12},
	}
	for i, test := range tests {
		t.Logf("\n\n-----------Starting test %d-----------\n", i)

		serviceA, dbA, _ := createMockService(t, peerA, peerAprivkey, 0, 10, nil, nil, beA, []mockTopology.Option{mockTopology.WithPeers(peerB)})
		pushsyncA := newPushSync(t, peerA, nil, dbA, nil, []mockTopology.Option{mockTopology.WithPeers(peerB)})

		serviceB, dbB, _ := createMockService(t, peerB, peerBprivkey, 0, 10, nil, nil, beB, []mockTopology.Option{mockTopology.WithPeers(peerA)})
		pushsyncB := newPushSync(t, peerB, nil, dbB, nil, []mockTopology.Option{mockTopology.WithPeers(peerA)})

		serviceA.AddStreamer(streamtest.New(streamtest.WithBaseAddr(peerA),
			streamtest.WithProtocols(serviceB.Protocol(), pushsyncB.Protocol())))
		serviceB.AddStreamer(streamtest.New(streamtest.WithBaseAddr(peerB),
			streamtest.WithProtocols(serviceA.Protocol(), pushsyncA.Protocol())))

		// Create chunks for A and B
		chunksA := make([]swarm.Chunk, test.numChunksA)
		chunksB := make([]swarm.Chunk, test.numChunksB)
		uniqueChunks := 0
		for i := 0; i < cap(chunksA)+cap(chunksB); i++ {
			uniquecheck := false
			rndChunk := testingc.GenerateTestRandomChunk()
			if i < test.numChunksA {
				chunksA[i] = rndChunk
				uniquecheck = true
			}
			if i < test.numChunksB {
				if rand.Intn(100) < test.similarity {
					chunksB[i] = rndChunk
					uniquecheck = true
				} else {
					chunksB[i] = testingc.GenerateTestRandomChunk()
					uniqueChunks++
				}
			}
			if uniquecheck {
				uniqueChunks++
			}
		}

		// Store chunks for A and B
		dbA.Put(context.Background(), storage.ModePutUpload, chunksA...)
		dbB.Put(context.Background(), storage.ModePutUpload, chunksB...)

		indicesA, _ := dbA.DebugIndices()
		indicesB, _ := dbB.DebugIndices()
		t.Logf("BEFORE: Peer A now has %d chunks\n", indicesA["retrievalDataIndex"])
		t.Logf("BEFORE: Peer B now has %d chunks\n", indicesB["retrievalDataIndex"])
		t.Logf("BEFORE: Similarity rate: %d, Unique chunks: %d\n", test.similarity, uniqueChunks)

		serviceA.Start(context.Background())
		serviceB.Start(context.Background())

		for i := 0; i < 2*test.wait; i++ {
			indicesA, _ = dbA.DebugIndices()
			indicesB, _ = dbB.DebugIndices()
			if indicesA["retrievalDataIndex"] == indicesB["retrievalDataIndex"] && indicesA["retrievalDataIndex"] == uniqueChunks {
				break
			}
			<-time.After(500 * time.Millisecond)
		}

		serviceA.Close()
		serviceB.Close()

		if indicesA["retrievalDataIndex"] != indicesB["retrievalDataIndex"] || indicesA["retrievalDataIndex"] != uniqueChunks {
			t.Fatalf("Peer A and B should have the same number of chunks. A has %d, B has %d. Unique: %d", indicesA["retrievalDataIndex"], indicesB["retrievalDataIndex"], uniqueChunks)
		} else {
			t.Logf("AFTER: Peer A now has %d chunks\n", indicesA["retrievalDataIndex"])
			t.Logf("AFTER: Peer B now has %d chunks\n", indicesB["retrievalDataIndex"])
		}

		fmt.Printf("Done")
	}
}

type sendChunkFunc func(context.Context, swarm.Chunk) (*pushsync.Receipt, error)

// newTestDB is a helper function that constructs a
// temporary database and returns a cleanup function that must
// be called to remove the data.
func newTestDB(t testing.TB, o *localstore.Options) *localstore.DB {
	t.Helper()

	baseKey := make([]byte, 32)
	if _, err := rand.Read(baseKey); err != nil {
		t.Fatal(err)
	}
	if o == nil {
		o = &localstore.Options{}
	}
	if o.UnreserveFunc == nil {
		o.UnreserveFunc = func(postage.UnreserveIteratorFn) error {
			return nil
		}
	}
	logger := log.Noop
	db, err := localstore.New("", baseKey, nil, o, logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := db.Close()
		if err != nil {
			t.Error(err)
		}
	})
	return db
}

func createMockService(t testing.TB, addr swarm.Address, privkey *ecdsa.PrivateKey, storageradius uint8, blockinterval uint64, recorder *streamtest.Recorder, sendChunk sendChunkFunc, backend transaction.Backend, topologyOpts []mockTopology.Option) (service *snips.Service, db *localstore.DB, metrics *snips.Metrics) {
	t.Helper()
	logger := log.NewLogger("mockservicelog").Build()

	mockTopology := mockTopology.NewTopologyDriver(topologyOpts...)
	rs := &reserveStateGetter{rs: postage.ReserveState{StorageRadius: storageradius}}
	db = newTestDB(t, nil)
	streamer := streamtest.NewRecorderDisconnecter(recorder)

	blockHash := common.HexToHash("0x1").Bytes()
	enabledisable, blockcheckinterval := true, 100*time.Millisecond
	var mingamma, maxgamma float64 = 2, 3
	service = snips.New(
		addr,
		blockHash,
		1,
		logger,
		rs,
		crypto.NewDefaultSigner(privkey),
		db,
		streamer,
		mockTopology,
		backend,
		snips.Options{
			Enabled:            &enabledisable,
			MinGamma:           &mingamma,
			MaxGamma:           &maxgamma,
			BlockInterval:      &blockinterval,
			BlockCheckInterval: &blockcheckinterval,
		},
	)
	metrics = service.MetricsRaw()
	return
}

func generateSwarmAddress() (swarm.Address, *ecdsa.PrivateKey, error) {
	blockHash := common.HexToHash("0x1").Bytes()
	privkey, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}
	peer, err := crypto.NewOverlayAddress(privkey.PublicKey, 1, blockHash)
	if err != nil {
		return swarm.ZeroAddress, nil, err
	}

	return peer, privkey, nil
}

func newPushSync(t testing.TB, addr swarm.Address, blockHash []byte, db *localstore.DB, recorder *streamtest.Recorder, topologyOpts []mockTopology.Option) *pushsync.PushSync {
	logger := log.Noop

	mockTopology := mockTopology.NewTopologyDriver(topologyOpts...)
	mockStatestore := statestore.NewStateStore()
	mtag := tags.NewTags(mockStatestore, logger)

	mockAccounting := accountingmock.NewAccounting()
	mockPricer := pricermock.NewMockService(10, 10)
	recorderDisconnecter := streamtest.NewRecorderDisconnecter(recorder)
	unwrap := func(swarm.Chunk) {}
	validStamp := func(ch swarm.Chunk, stamp []byte) (swarm.Chunk, error) {
		return ch, nil
	}
	pk, _ := crypto.GenerateSecp256k1Key()

	//func New(address swarm.Address, nonce []byte, streamer p2p.StreamerDisconnecter, storer storage.Putter, topology topology.Driver, tagger *tags.Tags, isFullNode bool, unwrap func(swarm.Chunk), validStamp postage.ValidStampFn, logger log.Logger, accounting accounting.Interface, pricer pricer.Interface, signer crypto.Signer, tracer *tracing.Tracer, warmupTime time.Duration, o Options) *PushSync
	enabledisable := true
	return pushsync.New(addr, blockHash, recorderDisconnecter, db, mockTopology, mtag, true, unwrap, validStamp, logger, mockAccounting, mockPricer, crypto.NewDefaultSigner(pk), nil, -1, pushsync.Options{
		Enabled: &enabledisable,
	})
}

type reserveStateGetter struct {
	rs postage.ReserveState
}

func (r *reserveStateGetter) GetReserveState() *postage.ReserveState {
	return &r.rs
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
