package pos_test

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/log"
	"github.com/ethersphere/bee/pkg/p2p/streamtest"
	"github.com/ethersphere/bee/pkg/pos"
	"github.com/ethersphere/bee/pkg/pushsync"
	mockps "github.com/ethersphere/bee/pkg/pushsync/mock"
	"github.com/ethersphere/bee/pkg/storage"
	mocks "github.com/ethersphere/bee/pkg/storage/mock"
	testingc "github.com/ethersphere/bee/pkg/storage/testing"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/ethersphere/bee/pkg/topology"
	mockt "github.com/ethersphere/bee/pkg/topology/mock"
)

var noOfRetries = 20

type sendChunkFunc func(context.Context, swarm.Chunk) (*pushsync.Receipt, error)

func TestPOSProof(t *testing.T) {
	// chunk data to reupload
	chunk := testingc.FixtureChunk("7000")

	// create a pivot node and a mocked closest node
	pivotNode := swarm.MustParseHexAddress("0000000000000000000000000000000000000000000000000000000000000000")   // base is 0000
	closestPeer := swarm.MustParseHexAddress("6000000000000000000000000000000000000000000000000000000000000000") // binary 0110 -> po 1

	sendChunkPivot := func(ctx context.Context, c swarm.Chunk) (*pushsync.Receipt, error) {
		panic("unexpected push")
	}

	closest, closestStorer, realClosestAddr := createPOSNode(
		closestPeer,
		nil,
		nil,
		[]mocks.Option{mocks.WithBaseAddress(closestPeer)},
		[]mockt.Option{mockt.WithClosestPeerErr(topology.ErrWantSelf), mockt.WithPeers(pivotNode)},
	)

	_, _ = closestStorer.Put(context.TODO(), storage.ModePutRequest, chunk)

	recorder := streamtest.New(streamtest.WithBaseAddr(pivotNode), streamtest.WithProtocols(closest.Protocol()))

	pivot, storer, _ := createPOSNode(
		pivotNode,
		recorder,
		sendChunkPivot,
		[]mocks.Option{mocks.WithBaseAddress(pivotNode)},
		[]mockt.Option{mockt.WithClosestPeer(closestPeer), mockt.WithPeers(closestPeer)},
	)

	_, _ = storer.Put(context.TODO(), storage.ModePutRequest, chunk)

	status, err := pivot.Reupload(context.TODO(), chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	// HACK: the address used for signing the message is not the same as the one used for topology
	proved := status.Proved[realClosestAddr.String()]

	if len(proved) != 1 || !proved[0].Equal(chunk.Address()) {
		t.Error("expected proved status for chunk")
	}
}

func TestPOSPush(t *testing.T) {
	// chunk data to reupload
	chunk := testingc.FixtureChunk("7000")

	// create a pivot node and a mocked closest node
	pivotNode := swarm.MustParseHexAddress("0000000000000000000000000000000000000000000000000000000000000000")   // base is 0000
	closestPeer := swarm.MustParseHexAddress("6000000000000000000000000000000000000000000000000000000000000000") // binary 0110 -> po 1

	var pushChunk = swarm.ZeroAddress

	sendChunkPivot := func(ctx context.Context, c swarm.Chunk) (*pushsync.Receipt, error) {
		pushChunk = c.Address()
		return nil, nil
	}

	closest, _, _ := createPOSNode(
		closestPeer,
		nil,
		nil,
		[]mocks.Option{mocks.WithBaseAddress(closestPeer)},
		[]mockt.Option{mockt.WithClosestPeerErr(topology.ErrWantSelf), mockt.WithPeers(pivotNode)},
	)

	recorder := streamtest.New(streamtest.WithBaseAddr(pivotNode), streamtest.WithProtocols(closest.Protocol()))

	pivot, storer, _ := createPOSNode(
		pivotNode,
		recorder,
		sendChunkPivot,
		[]mocks.Option{mocks.WithBaseAddress(pivotNode)},
		[]mockt.Option{mockt.WithClosestPeer(closestPeer)},
	)

	_, _ = storer.Put(context.TODO(), storage.ModePutRequest, chunk)

	status, err := pivot.Reupload(context.TODO(), chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Reuploaded) != 1 || !status.Reuploaded[0].Equal(chunk.Address()) {
		t.Error("expected reuploaded status for chunk")
	}

	// Check if the chunk is set as synced in the DB.
	for i := 0; i < noOfRetries; i++ {
		// Give some time for chunk to be pushed and receipt to be received
		time.Sleep(10 * time.Millisecond)

		if pushChunk.Equal(chunk.Address()) {
			return
		}
	}

	t.Error("no chunk was pushed")
}

func TestPOSForwardProof(t *testing.T) {
	// chunk data to reupload
	chunk := testingc.FixtureChunk("7000")

	// create a pivot node and a mocked closest node
	pivotNode := swarm.MustParseHexAddress("0000000000000000000000000000000000000000000000000000000000000000") // base is 0000
	forwardNode := swarm.MustParseHexAddress("2000000000000000000000000000000000000000000000000000000000000000")
	closestPeer := swarm.MustParseHexAddress("6000000000000000000000000000000000000000000000000000000000000000") // binary 0110 -> po 1

	sendChunkPivot := func(ctx context.Context, c swarm.Chunk) (*pushsync.Receipt, error) {
		panic("unexpected push")
	}

	closest, closestStorer, realClosestAddr := createPOSNode(
		closestPeer,
		nil,
		nil,
		[]mocks.Option{mocks.WithBaseAddress(closestPeer)},
		[]mockt.Option{mockt.WithClosestPeerErr(topology.ErrWantSelf), mockt.WithPeers(forwardNode)},
	)

	_, _ = closestStorer.Put(context.TODO(), storage.ModePutRequest, chunk)

	forwardRecorder := streamtest.New(streamtest.WithBaseAddr(forwardNode), streamtest.WithProtocols(closest.Protocol()))

	forward, _, _ := createPOSNode(
		forwardNode,
		forwardRecorder,
		nil,
		[]mocks.Option{mocks.WithBaseAddress(forwardNode)},
		[]mockt.Option{mockt.WithClosestPeer(closestPeer), mockt.WithPeers(pivotNode, closestPeer)},
	)

	pivotRecorder := streamtest.New(streamtest.WithBaseAddr(pivotNode), streamtest.WithProtocols(forward.Protocol()))

	pivot, storer, _ := createPOSNode(
		pivotNode,
		pivotRecorder,
		sendChunkPivot,
		[]mocks.Option{mocks.WithBaseAddress(pivotNode)},
		[]mockt.Option{mockt.WithClosestPeer(forwardNode), mockt.WithPeers(forwardNode)},
	)

	_, _ = storer.Put(context.TODO(), storage.ModePutRequest, chunk)

	status, err := pivot.Reupload(context.TODO(), chunk.Address())
	if err != nil {
		t.Fatal(err)
	}

	proved := status.Proved[realClosestAddr.String()]

	if len(proved) != 1 || !proved[0].Equal(chunk.Address()) {
		t.Error("expected proved status for chunk")
	}
}

func createPOSNode(addr swarm.Address, recorder *streamtest.Recorder, sendChunk sendChunkFunc, storerOpts []mocks.Option, topologyOpts []mockt.Option) (service *pos.Service, storer *mocks.MockStorer, realAddr swarm.Address) {
	logger := log.NewLogger("mockservicelog").Build()

	mockTopology := mockt.NewTopologyDriver(topologyOpts...)
	mockPushsync := mockps.New(sendChunk)
	storer = mocks.NewStorer(storerOpts...)
	streamer := streamtest.NewRecorderDisconnecter(recorder)

	pk, _ := crypto.GenerateSecp256k1Key()
	blockHash := common.HexToHash("0x1").Bytes()

	enabled := new(bool)
	*enabled = true

	service = pos.New(
		addr,
		blockHash,
		1,
		logger,
		mockPushsync,
		crypto.NewDefaultSigner(pk),
		storer,
		streamer,
		mockTopology,
		pos.Options{
			Enabled: enabled,
		},
	)

	realAddr, err := crypto.NewOverlayAddress(pk.PublicKey, 1, blockHash)
	if err != nil {
		panic(err)
	}

	return service, storer, realAddr
}
