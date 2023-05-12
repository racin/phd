package pullsyncfail_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethersphere/bee/pkg/localstore"
	"github.com/ethersphere/bee/pkg/log"
	"github.com/ethersphere/bee/pkg/p2p"
	"github.com/ethersphere/bee/pkg/p2p/streamtest"
	"github.com/ethersphere/bee/pkg/postage"
	postagetesting "github.com/ethersphere/bee/pkg/postage/testing"
	"github.com/ethersphere/bee/pkg/pullsync"
	"github.com/ethersphere/bee/pkg/pullsync/pullstorage"
	"github.com/ethersphere/bee/pkg/pullsync/pullstorage/mock"
	"github.com/ethersphere/bee/pkg/shed"
	"github.com/ethersphere/bee/pkg/storage"
	testingc "github.com/ethersphere/bee/pkg/storage/testing"
	"github.com/ethersphere/bee/pkg/swarm"
)

var (
	addrs  []swarm.Address
	chunks []swarm.Chunk
)

func PrintCursors(prefix string, Other, Mine *pullstorage.PullStorer) {
	otherCursor, _ := Other.Cursors(context.Background())
	myCursor, _ := Mine.Cursors(context.Background())
	fmt.Printf("%-28s. | Other cursor: %v, My cursor: %v\n", prefix, otherCursor[0], myCursor[0])
}

// This test demonstrates inconsistenties in pullsync:
// Peer A and peer B are synced with 5 chunks. Then peer A deletes 1 (or more) chunks, and peer A tries to sync with peer B.
// The deleted chunk should not be the last one synced. I.e. the deleted should have a lower BinID than the current maximum for that bin.
// --- In this case, the chunk will not be synced and the peer will remain inconsistent.
func TestPullsyncFail_Inconsistent(t *testing.T) {
	peerA := swarm.MustParseHexAddress("aaaaa00000000000000000000000000000000000000000000000000000000000")
	peerB := swarm.MustParseHexAddress("aaaaa11111111111111111111111111111111111111111111111111111111111")

	chunks := generateChunksWithPO(5, 0, peerA.Bytes())
	for i := 0; i < len(chunks); i++ {
		fmt.Printf("Chunk: %s, PO: %d\n", chunks[i].Address().String(), swarm.Proximity(chunks[i].Address().Bytes(), peerA.Bytes()))
	}
	ps, psOtherDB, _ := newPullSync(t, streamtest.New(streamtest.WithBaseAddr(peerA)), peerA.Bytes(), nil)
	recorder := streamtest.New(streamtest.WithBaseAddr(peerB), streamtest.WithProtocols(ps.Protocol()))
	psClient, clientDb, clientLDB := newPullSync(t, recorder, peerB.Bytes(), nil)

	psOtherDB.Put(context.Background(), storage.ModePutUpload, chunks[0:5]...)
	clientDb.Put(context.Background(), storage.ModePutUpload, chunks[0])
	PrintCursors("Initial upload", psOtherDB, clientDb)

	// Sync everything.
	topmost, _, err := psClient.SyncInterval(context.Background(), swarm.ZeroAddress, 0, 0, pullsync.MaxCursor)
	if err != nil {
		t.Fatal(err)
	}
	// We expect that the client have synced all chunks
	haveChunks(t, clientDb, chunks...)
	haveChunks(t, psOtherDB, chunks...)
	PrintCursors("After first syncing", psOtherDB, clientDb)

	fmt.Printf("Client deletes chunk: %s\n", chunks[3].Address().String())
	err = clientDb.Set(context.Background(), storage.ModeSetRemove, chunks[3].Address())
	if err != nil {
		t.Fatal(err)
	}

	PrintCursors("After deletion", psOtherDB, clientDb)

	// Then the client sync again, from the highest synced interval
	_ = topmost
	topmost, _, err = psClient.SyncInterval(context.Background(), swarm.ZeroAddress, 0, topmost, pullsync.MaxCursor)
	if err != nil {
		t.Fatal(err)
	}
	PrintCursors("Second syncing", psOtherDB, clientDb)

	fmt.Printf("--- The client are now storing these: ----\n")
	clientLDB.IterStoredChunks(context.TODO(), false, func(item shed.Item) error {
		fmt.Printf("Client's chunk: %x\n", item.Address)
		return err
	})
	// We expect that the client have synced all chunks
	haveChunks(t, psOtherDB, chunks...)
	haveChunks(t, clientDb, chunks...)
}

// This test demonstrates deadlock in pullsync:
// Peer A and peer B are synced with 5 chunks. Then peer A deletes a chunk, and peer B tries to sync with peer A.
// --- In this case, the SyncInterval() function will livelock, as it is trying to sync from topmost index (5), but that doesn't exist.
func TestPullsyncFail_Deadlock(t *testing.T) {
	peerA := swarm.MustParseHexAddress("aaaaa00000000000000000000000000000000000000000000000000000000000")
	peerB := swarm.MustParseHexAddress("aaaaa11111111111111111111111111111111111111111111111111111111111")

	chunks := generateChunksWithPO(5, 0, peerA.Bytes())
	for i := 0; i < len(chunks); i++ {
		fmt.Printf("Chunk: %s, PO: %d\n", chunks[i].Address().String(), swarm.Proximity(chunks[i].Address().Bytes(), peerA.Bytes()))
	}
	ps, psOtherDB, _ := newPullSync(t, streamtest.New(streamtest.WithBaseAddr(peerA)), peerA.Bytes(), nil)
	recorder := streamtest.New(streamtest.WithBaseAddr(peerB), streamtest.WithProtocols(ps.Protocol()))
	psClient, clientDb, _ := newPullSync(t, recorder, peerB.Bytes(), nil)

	psOtherDB.Put(context.Background(), storage.ModePutUpload, chunks[0:5]...)
	clientDb.Put(context.Background(), storage.ModePutUpload, chunks[0])
	PrintCursors("Initial upload", psOtherDB, clientDb)

	// Sync everything.
	topmost, _, err := psClient.SyncInterval(context.Background(), swarm.ZeroAddress, 0, 0, pullsync.MaxCursor)
	if err != nil {
		t.Fatal(err)
	}
	// We expect that the client have synced all chunks (5)
	haveChunks(t, clientDb, chunks...)
	haveChunks(t, psOtherDB, chunks...)
	PrintCursors("After first syncing", psOtherDB, clientDb)

	fmt.Printf("Other peer deletes chunk: %s\n", chunks[4].Address().String())
	err = psOtherDB.Set(context.Background(), storage.ModeSetRemove, chunks[4].Address())
	if err != nil {
		t.Fatal(err)
	}
	PrintCursors("Other deletes a chunk", psOtherDB, clientDb)

	fmt.Printf("!!! This will never finish. I think... !!!\n")
	// Then the client sync again, from the highest synced interval
	_ = topmost
	topmost, _, err = psClient.SyncInterval(context.Background(), swarm.ZeroAddress, 0, topmost, pullsync.MaxCursor)
	if err != nil {
		t.Fatal(err)
	}
	PrintCursors("Second syncing", psOtherDB, clientDb)
	// We expect that the client have synced all chunks
	haveChunks(t, psOtherDB, chunks...)
	haveChunks(t, clientDb, chunks...)
}

func someChunks(i ...int) (c []swarm.Chunk) {
	for _, v := range i {
		c = append(c, chunks[v])
	}
	return c
}

func generateChunksWithPO(numChunks int, po uint8, addr []byte) []swarm.Chunk {
	chunks := make([]swarm.Chunk, numChunks)
	stamp := postagetesting.MustNewStamp()
	for i := 0; i < numChunks; i++ {
		theChunk := testingc.GenerateTestRandomChunk().WithStamp(stamp)
		if swarm.Proximity(theChunk.Address().Bytes(), addr) != po {
			// fmt.Printf("Failed. Chunk: %s, PO: %d\n", theChunk.Address(), swarm.Proximity(theChunk.Address().Bytes(), addr))
			i--
			continue
		}
		chunks[i] = theChunk
	}
	return chunks
}

// nolint:gochecknoinits
func init() {
	n := 5
	chunks = make([]swarm.Chunk, n)
	addrs = make([]swarm.Address, n)

	stamp := postagetesting.MustNewStamp()
	for i := 0; i < n; i++ {
		// return swarm.NewChunk(addr, data)
		chunks[i] = testingc.GenerateTestRandomChunk().WithStamp(stamp)
		addrs[i] = chunks[i].Address()
	}
}
func haveChunks(t *testing.T, s *pullstorage.PullStorer, chunks ...swarm.Chunk) {
	t.Helper()
	for _, a := range chunks {
		have, err := s.Has(context.Background(), a.Address())
		if err != nil {
			t.Fatal(err)
		}
		if !have {
			t.Errorf("storage does not have chunk %s", a)
		}
	}
}

func newPullSync(t testing.TB, s p2p.Streamer, baseKey []byte, o ...mock.Option) (*pullsync.Syncer, *pullstorage.PullStorer, *localstore.DB) {
	// storage := mock.NewPullStorage(o...)
	// storage := newPullLocalstore(t, nil, o...)

	enabled := true
	db := newTestDB(t, baseKey, nil)
	storage := pullstorage.New(db, pullstorage.Options{
		Enabled: &enabled,
	})
	logger := log.Noop
	unwrap := func(swarm.Chunk) {}
	validStamp := func(ch swarm.Chunk, _ []byte) (swarm.Chunk, error) {
		stamp := postagetesting.MustNewStamp()
		return ch.WithStamp(stamp), nil
	}
	return pullsync.New(s, storage, unwrap, validStamp, logger, pullsync.Options{
		Enabled: &enabled,
	}), storage, db
}

type pullLocalstore struct {
	db       *localstore.DB
	ps       *mock.PullStorage
	putCalls int
	setCalls int
}

func newPullLocalstore(t testing.TB, o *localstore.Options, opts ...mock.Option) *pullLocalstore {
	t.Helper()
	return &pullLocalstore{}
}

func (pl *pullLocalstore) Cursors(ctx context.Context) (curs []uint64, err error) {
	return pl.ps.Cursors(ctx)
}
func (pl *pullLocalstore) IntervalChunks(_ context.Context, bin uint8, from, to uint64, limit int) (chunks []swarm.Address, topmost uint64, err error) {
	return pl.ps.IntervalChunks(context.TODO(), bin, from, to, limit)
}
func (pl *pullLocalstore) PutCalls() int {
	return pl.putCalls
}
func (pl *pullLocalstore) SetCalls() int {
	return pl.ps.SetCalls()
}

// Get chunks.
func (pl *pullLocalstore) Get(_ context.Context, _ storage.ModeGet, addrs ...swarm.Address) (chs []swarm.Chunk, err error) {
	return pl.db.GetMulti(context.Background(), storage.ModeGetRequest, addrs...)
}

// Put chunks.
func (pl *pullLocalstore) Put(_ context.Context, _ storage.ModePut, chs ...swarm.Chunk) error {
	_, err := pl.db.Put(context.Background(), storage.ModePutRequest, chs...)
	pl.putCalls++
	return err
}

// Set chunks.
func (pl *pullLocalstore) Set(ctx context.Context, mode storage.ModeSet, addrs ...swarm.Address) error {
	err := pl.db.Set(ctx, mode, addrs...)
	pl.setCalls++
	return err
}

// Has chunks.
func (pl *pullLocalstore) Has(_ context.Context, addr swarm.Address) (bool, error) {
	return pl.db.Has(context.Background(), addr)
}

// newTestDB is a helper function that constructs a
// temporary database and returns a cleanup function that must
// be called to remove the data.
func newTestDB(t testing.TB, baseKey []byte, o *localstore.Options) *localstore.DB {
	t.Helper()

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
