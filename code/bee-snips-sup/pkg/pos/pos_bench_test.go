package pos

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/log"
	"github.com/ethersphere/bee/pkg/p2p/streamtest"
	"github.com/ethersphere/bee/pkg/pos/pb"
	"github.com/ethersphere/bee/pkg/pushsync"
	mockps "github.com/ethersphere/bee/pkg/pushsync/mock"
	"github.com/ethersphere/bee/pkg/storage"
	mocks "github.com/ethersphere/bee/pkg/storage/mock"
	testingc "github.com/ethersphere/bee/pkg/storage/testing"
	"github.com/ethersphere/bee/pkg/swarm"
	mockt "github.com/ethersphere/bee/pkg/topology/mock"
)

func newPOS(b *testing.B) (pos *Service, storer storage.Storer) {
	b.Helper()

	logger := log.NewLogger("mockservicelog").Build()
	pk, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		b.Fatal(err)
	}
	storer = mocks.NewStorer()
	topology := mockt.NewTopologyDriver(mockt.WithNeighborhoodDepth(0))
	streamer := streamtest.New()
	signer := crypto.NewDefaultSigner(pk)
	ps := mockps.New(func(ctx context.Context, chunk swarm.Chunk) (*pushsync.Receipt, error) { panic("unexpected push") })

	pos = New(swarm.ZeroAddress, make([]byte, 32), 0, logger, ps, signer, storer, streamer, topology, Options{})

	return pos, storer
}

func BenchmarkProcessChallenge(b *testing.B) {
	ch := testingc.FixtureChunk("7000")

	pos, storer := newPOS(b)

	storer.Put(context.Background(), storage.ModePutRequest, ch)

	nonce, err := pos.generateNonce()
	if err != nil {
		b.Fatal(err)
	}

	var resp *pb.Response

	chall := &pb.Challenge{
		Nonce:   nonce,
		Address: ch.Address().Bytes(),
	}

	for i := 0; i < b.N; i++ {
		resp, err = pos.processChallenge(context.Background(), chall)
		if err != nil {
			b.Fatal(err)
		}
	}
	_ = resp
}

func BenchmarkHandleResponse(b *testing.B) {
	ch := testingc.FixtureChunk("7000")

	pos, storer := newPOS(b)

	storer.Put(context.Background(), storage.ModePutRequest, ch)

	nonce, err := pos.generateNonce()
	if err != nil {
		b.Fatal(err)
	}

	resp, err := pos.processChallenge(context.Background(), &pb.Challenge{
		Nonce:   nonce,
		Address: ch.Address().Bytes(),
	})

	if err != nil {
		b.Fatal(err)
	}

	var prover swarm.Address

	for i := 0; i < b.N; i++ {
		prover, err = pos.handleResponse(context.Background(), ch.Address(), resp)
		if err != nil {
			b.Fatal(err)
		}
	}

	_ = prover
}

func BenchmarkVerifyProof(b *testing.B) {
	ch := testingc.FixtureChunk("7000")

	pos, storer := newPOS(b)

	storer.Put(context.Background(), storage.ModePutRequest, ch)

	nonce, err := pos.generateNonce()
	if err != nil {
		b.Fatal(err)
	}

	resp, err := pos.processChallenge(context.Background(), &pb.Challenge{
		Nonce:   nonce,
		Address: ch.Address().Bytes(),
	})

	if err != nil {
		b.Fatal(err)
	}

	pk, err := pos.signer.PublicKey()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		err = pos.verifyProof(context.Background(), ch.Address(), pk, resp.Proof)
		if err != nil {
			b.Fatal(err)
		}
	}

	_ = err
}

func BenchmarkGenerateNonceRand(b *testing.B) {
	var (
		nonce []byte
		err   error
	)
	for i := 0; i < b.N; i++ {
		nonce, err = generateNonce_rand()
		if err != nil {
			b.Fatal(err)
		}
	}
	_ = nonce
}

func BenchmarkGenerateNonceSecp256k(b *testing.B) {
	var (
		nonce []byte
		err   error
	)
	for i := 0; i < b.N; i++ {
		nonce, err = generateNonce_secp256k()
		if err != nil {
			b.Fatal(err)
		}
	}
	_ = nonce
}

func generateNonce_rand() (nonce []byte, err error) {
	// TODO(johningve) store the key somewhere?
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

func generateNonce_secp256k() (nonce []byte, err error) {
	// TODO(johningve) store the key somewhere?
	k, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		return nil, fmt.Errorf("error generating key for nonce: %w", err)
	}
	nonce, err = crypto.LegacyKeccak256(crypto.EncodeSecp256k1PrivateKey(k))
	if err != nil {
		return nil, fmt.Errorf("error generating hash of nonce key: %w", err)
	}
	return nonce, nil
}
