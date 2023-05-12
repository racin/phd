package pb_test

import (
	"testing"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/pos/pb"
	pushpb "github.com/ethersphere/bee/pkg/pushsync/pb"
	"github.com/ethersphere/bee/pkg/swarm"
)

func TestSizes(t *testing.T) {
	chall := &pb.Challenge{
		Nonce:   make([]byte, 32),
		Address: make([]byte, 32),
	}

	challBytes, err := chall.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	key, err := crypto.GenerateSecp256k1Key()
	if err != nil {
		t.Fatal(err)
	}

	signer := crypto.NewDefaultSigner(key)

	sig, err := signer.Sign(challBytes)
	if err != nil {
		t.Fatal(err)
	}

	resp := &pb.Response{
		Proof: &pb.Proof{
			Challenge: chall,
			Hash:      make([]byte, 32),
		},
		Signature: sig,
		BlockHash: make([]byte, 32),
	}

	respBytes, err := resp.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	delivery := &pushpb.Delivery{
		Address: make([]byte, 32),
		Data:    make([]byte, swarm.ChunkWithSpanSize),
		Stamp:   make([]byte, 113),
	}

	deliveryBytes, err := delivery.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	receipt := &pushpb.Receipt{
		Address:   make([]byte, 32),
		Signature: sig,
		Nonce:     make([]byte, 32),
	}

	receiptBytes, err := receipt.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	challLen := float64(len(challBytes))
	deliveryLen := float64(len(deliveryBytes))
	respLen := float64(len(respBytes))
	receiptLen := float64(len(receiptBytes))

	t.Logf("pos challenge size: %d", len(challBytes))
	t.Logf("pos response size: %d", len(respBytes))
	t.Logf("pushsync delivery size: %d", len(deliveryBytes))
	t.Logf("pushsync receipt size: %d", len(receiptBytes))

	t.Logf("theoretical savings: %f%%", 100-(challLen+respLen)/(deliveryLen+receiptLen)*100)
}
