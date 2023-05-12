package snips_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethersphere/bee/pkg/snips"
	"github.com/ethersphere/bee/pkg/storage"
	testingc "github.com/ethersphere/bee/pkg/storage/testing"
	"github.com/ethersphere/bee/pkg/swarm"
)

func TestMissingChunks(t *testing.T) {
	tests := []struct {
		numChunks     int
		nonce         uint64
		missingChunks []int // Which chunks indices do we simulate as missing.
	}{
		{numChunks: 10, nonce: 9999, missingChunks: []int{0, 1, 2, 3}},
		{numChunks: 100, nonce: 9999, missingChunks: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{numChunks: 1000, nonce: 9999, missingChunks: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}},
		{numChunks: 10000, nonce: 9999, missingChunks: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}},
	}

	for _, test := range tests {
		// create a random set of chunks
		myChunks := make([]swarm.Chunk, test.numChunks)
		snipsService, db, _ := createMockService(t, swarm.MustParseHexAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), nil, 0, 5, nil, nil, nil, nil)
		for i := 0; i < test.numChunks; i++ {
			myChunks[i] = testingc.GenerateTestRandomChunk()
		}

		db.Put(context.Background(), storage.ModePutUpload, myChunks...)
		valToChunkId, mapChunkIdToProof := snipsService.GetProofMapping()
		chunksChan := make(chan swarm.Chunk)
		doneChan := make(chan struct{})
		var minRange, maxRange []byte
		snipsService.GetChunksInMyRadius(context.TODO(), chunksChan, doneChan, &minRange, &maxRange)

		baseProof, chunkProofToId, err := snipsService.BuildBaseProof(context.TODO(), chunksChan, doneChan, test.nonce)
		if err != nil {
			t.Fatal(err)
		}

		proof, err := snipsService.CreateMPHF(context.Background(), baseProof, chunkProofToId, test.nonce)
		if err != nil {
			t.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
		}

		var missing []uint64
		deletedChunkId := make([]swarm.Address, 0)

		// Start losing chunks
		for _, numMissChunks := range test.missingChunks {
			db.Set(context.Background(), storage.ModeSetRemove, myChunks[numMissChunks].Address())

			t.Run("Test_FindMissingChunks", func(t *testing.T) {
				deletedChunkId = append(deletedChunkId, myChunks[numMissChunks].Address())
				foundMissing := make(map[string]struct{})

				miss, err := snipsService.FindMissingChunks(context.Background(), proof, test.nonce, uint32(len(myChunks)), minRange, maxRange)

				if err != nil {
					t.Fatalf("Error finding missing chunks. Numchunks: %d, Err: %v", test.numChunks, err)
				}
				missing = miss.GetMissingChunks()

				for _, m := range missing {
					foundMissing[valToChunkId[test.nonce][m].ByteString()] = struct{}{}
				}
				for j := 0; j < len(deletedChunkId); j++ {
					if _, ok := foundMissing[deletedChunkId[j].ByteString()]; !ok {
						t.Fatalf("Missing chunk not found. Numchunks: %d, Missing chunk: %v", test.numChunks, deletedChunkId[j])
					}
				}
			})
		}
		db.Put(context.Background(), storage.ModePutUpload, myChunks...)
		t.Run("Test_GetMissingChunks", func(t *testing.T) {
			gottenChunks, err := snipsService.GetMissingChunks(context.TODO(), missing, test.nonce)
			if err != nil {
				t.Fatalf("Error getting missing chunks. Numchunks: %d, Err: %v", test.numChunks, err)
			}

			if len(gottenChunks) != len(missing) {
				t.Fatalf("Unexpected gottenChunks. Got: %d, Want: %d", len(gottenChunks), len(missing))
			}
			mapGotten := make(map[string]struct{})
			for i := 0; i < len(gottenChunks); i++ {
				mapGotten[gottenChunks[i].Address().ByteString()] = struct{}{}
				_, ok := mapChunkIdToProof[test.nonce][gottenChunks[i].Address().ByteString()]
				if !ok {
					t.Fatalf("Chunk not found mapChunkIdToProof. Addr: %v", gottenChunks[i].Address())
				}
			}
			for i := 0; i < len(missing); i++ {
				chunkid := valToChunkId[test.nonce][missing[i]]
				if _, ok := mapGotten[chunkid.ByteString()]; !ok {
					t.Fatalf("Chunk not found in mapGotten. Addr: %v", chunkid)
				}
			}
		})
	}

}

func TestExpectedValsSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		numChunks int
		fillRate  []float64 // How many percentage of the chunks are we already storing.
	}{
		{numChunks: 100, fillRate: []float64{10, 20, 30, 40, 50, 60, 70, 80}},
		{numChunks: 1000, fillRate: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 95}},
		{numChunks: 10000, fillRate: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 99}},
		{numChunks: 100000, fillRate: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 99}},
		{numChunks: 1000000, fillRate: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 99}},
	}

	for _, test := range tests {
		for _, fillRate := range test.fillRate {
			eVal := snips.NewExpectedVals(test.numChunks)
			numHits := int(float64(test.numChunks) / 100 * fillRate)
			knownRndVals := make(map[uint64]struct{})
			for i := 0; i < numHits; i++ {
				hashVal := rand.Uint64()%uint64(test.numChunks) + 1
				if _, ok := knownRndVals[hashVal]; ok {
					i--
					continue
				}
				knownRndVals[hashVal] = struct{}{}
				eVal.Hit(hashVal)
			}
			missing := eVal.GetMissingChunks()
			missingBitVector := missing.AsBitVector()
			if uint64(len(missing)*4) < missingBitVector.GetSerializedSizeInBytes() {
				t.Errorf("Expected missing chunks bit vector to be smaller than missing chunks slice. Missing chunks slice size: %d, missing chunks bit vector size: %d", len(missing)*4, missingBitVector.GetSerializedSizeInBytes())
			}
			t.Logf("Missing: %v, UintArray size: %v, BitVector size: %v, numChunks: %v, fillRate: %v\n", len(missing), len(missing)*4, missingBitVector.GetSerializedSizeInBytes(), test.numChunks, fillRate)
		}
	}
}

func TestCreateProof(t *testing.T) {
	chunks := []swarm.Chunk{
		testingc.FixtureChunk("0025"),
		testingc.FixtureChunk("0033"),
		testingc.FixtureChunk("02c2"),
		testingc.FixtureChunk("7000"),
	}
	baseproof := []uint64{
		11350884615230032978,
		14247628847674666623,
		15155240913233625475,
		13290450046525334366,
	}
	var nonce uint64 = 0x1234567890abcdef

	snipsService, _, _ := createMockService(t, swarm.ZeroAddress, nil, 8, 5, nil, nil, nil, nil)
	_, mapChunkIdToProof := snipsService.GetProofMapping()
	t.Run("GetChunkProof", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			if i == 0 && mapChunkIdToProof[nonce] != nil {
				t.Fatal("Expected mapChunkIdToProof to be uninitalized. Was initialized")
			} else if i != 0 && len(mapChunkIdToProof[nonce]) != len(baseproof) {
				t.Fatalf("Expected mapChunkIdToProof to have %d elements. Had %d. i: %d", len(baseproof), len(mapChunkIdToProof[nonce]), i)
			}
			for j, c := range chunks {
				proof, err := snipsService.GetChunkProof(context.TODO(), nonce, c)
				if err != nil {
					t.Fatal(err)
				}
				if proof != baseproof[j] {
					t.Fatalf("expected proof got %d, want %d", baseproof[j], proof)
				}
			}
		}
	})
	chunkchan := make(chan swarm.Chunk)
	donechan := make(chan struct{})
	t.Run("BuildBaseProof", func(t *testing.T) {
		go func() {
			defer close(donechan)
			for _, c := range chunks {
				chunkchan <- c
			}
			donechan <- struct{}{}
			close(chunkchan)
		}()
		newbaseproof, chunkProofToId, err := snipsService.BuildBaseProof(context.TODO(), chunkchan, donechan, nonce)
		if err != nil {
			t.Fatal(err)
		}
		if len(baseproof) != len(newbaseproof) {
			t.Fatalf("unexpected length of baseproof got %d, want %d", len(newbaseproof), len(baseproof))
		}
		for i, p := range newbaseproof {
			if p != baseproof[i] {
				t.Fatalf("unexpected value in baseproof got %d, want %d", p, baseproof[i])
			}
		}
		if len(chunkProofToId) != len(mapChunkIdToProof[nonce]) {
			t.Fatalf("unexpected length of chunkProofToId got %d, want %d", len(chunkProofToId), len(mapChunkIdToProof[nonce]))
		}
		for cp, id := range chunkProofToId {
			if cp != mapChunkIdToProof[nonce][id.ByteString()] {
				t.Fatalf("unexpected value in chunkProofToId got %d, want %d", cp, mapChunkIdToProof[nonce][id.ByteString()])
			}
		}
	})
}

func Test_CreateMPHF(t *testing.T) {
	t.Parallel()
	tests := []struct {
		numChunks int
		nonce     uint64
	}{
		{numChunks: 1, nonce: 9999},
		{numChunks: 10, nonce: 9999},
		{numChunks: 100, nonce: 9999},
		{numChunks: 1000, nonce: 9999},
		{numChunks: 10000, nonce: 9999},
	}
	if !testing.Short() {
		// tests = append(tests, struct {
		// 	numChunks int
		// 	nonce     uint64
		// }{numChunks: 100000, nonce: 9999})
		// tests = append(tests, struct {
		// 	numChunks int
		// 	nonce     uint64
		// }{numChunks: 262144, nonce: 9999}) // 1 GB
		// tests = append(tests, struct {
		// 	numChunks int
		// 	nonce     uint64
		// }{numChunks: 2621440, nonce: 9999}) // 10 GB
		// tests = append(tests, struct {
		// 	numChunks int
		// 	nonce     uint64
		// }{numChunks: 26214400, nonce: 9999}) // 100 GB)
	}

	chunksChan := make(chan swarm.Chunk)
	doneChan := make(chan struct{})

	for i, test := range tests {
		if testing.Verbose() {
			fmt.Printf("Start test %d, numChunks: %d\n", i, test.numChunks)
		}

		go func() {
			for i := 0; i < test.numChunks; i++ {
				if test.numChunks <= 100000 {
					chunksChan <- testingc.GenerateTestRandomChunk()
				} else {
					addr := make([]byte, 8)
					binary.LittleEndian.PutUint64(addr, uint64(i))
					chunk := swarm.NewChunk(swarm.NewAddress(addr), addr)
					chunksChan <- chunk
				}
			}
			doneChan <- struct{}{}
		}()

		snipsService, _, _ := createMockService(t, swarm.ZeroAddress, nil, 8, 5, nil, nil, nil, nil)

		baseProof, chunkProofToId, err := snipsService.BuildBaseProof(context.Background(), chunksChan, doneChan, test.nonce)

		if err != nil {
			t.Fatal(err)
		}

		if len(baseProof) != test.numChunks {
			t.Errorf("len(baseProof) returns incorrect value got %v, want %v", len(baseProof), test.numChunks)
		}
		gotAddrs := make(map[string]struct{})
		for _, chunkProof := range baseProof {
			if addr, ok := chunkProofToId[chunkProof]; !ok {
				t.Errorf("chunkProofToId(chunkProof) returns address. input %v, numchunks: %d", chunkProof, test.numChunks)
			} else if _, ok := gotAddrs[addr.String()]; ok {
				t.Errorf("gotAddrs(addr.String()) duplicate found. input %v, numchunks: %d", addr.String(), test.numChunks)
			} else {
				gotAddrs[addr.String()] = struct{}{}
			}
		}

		mapValToChunkId, mapChunkIdToProof := snipsService.GetProofMapping()

		bbhash, err := snipsService.CreateMPHF(context.Background(), baseProof, chunkProofToId, test.nonce)
		if err != nil {
			t.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
		}

		foundHashVals := make(map[uint64]struct{})
		for _, chunkProof := range baseProof {
			hashVal := bbhash.Find(chunkProof)
			if _, ok := foundHashVals[hashVal]; ok {
				t.Errorf("foundHashVals([hashVal] duplicate found. input %v, numchunks: %d", hashVal, test.numChunks)
			}
			foundHashVals[hashVal] = struct{}{}
			if !chunkProofToId[chunkProof].Equal(mapValToChunkId[test.nonce][hashVal]) {
				t.Errorf("mapValToChunkId[test.nonce][hashVal] returns incorrect address with input %d, got %v, want %v, numChunks: %d", hashVal, mapValToChunkId[test.nonce][hashVal], chunkProofToId[chunkProof], test.numChunks)
			}
			if chunkProof != mapChunkIdToProof[test.nonce][chunkProofToId[chunkProof].ByteString()] {
				t.Errorf("cpuint64 and mapChunkIdToProof[test.nonce][chunks[i].Address().ByteString()] do not match with input %x, got %v, want %v, numChunks: %d", chunkProofToId[chunkProof].ByteString(), mapChunkIdToProof[test.nonce][chunkProofToId[chunkProof].ByteString()], chunkProof, test.numChunks)
			}
		}
	}
}
