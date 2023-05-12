package snips_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethersphere/bee/pkg/localstore"
	"github.com/ethersphere/bee/pkg/p2p/streamtest"
	"github.com/ethersphere/bee/pkg/snips"
	"github.com/ethersphere/bee/pkg/storage"
	testingc "github.com/ethersphere/bee/pkg/storage/testing"
	"github.com/ethersphere/bee/pkg/swarm"
	mockTopology "github.com/ethersphere/bee/pkg/topology/mock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Testing accuracy without range checks by verifying the proof with as many chunks as it contains.
// The verifier has some chunks that is equal to the prover, and the other chunks are random.
func TestProofAccuracyWithoutRange(t *testing.T) {
	return // FIXME: This is only used for simulations.
	t.Parallel()
	tests := []struct {
		numChunks    int
		nonce        uint64
		availability []int // Availability represeents how many chunks another peer has. Availability = 0 means no chunks, Availability = 100 means all chunks.
	}{
		// {numChunks: 1, nonce: 9999, availability: []int{100}},
		// {numChunks: 10, nonce: 9999, availability: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}},
		{numChunks: 100, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}},
		{numChunks: 1000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}},
		{numChunks: 10000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}},
	}
	if !testing.Short() {
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 100000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}})
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 262144, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}}) // 1 GB
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 2621440, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 10 GB
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 26214400, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 100 GB)
	}

	for i, test := range tests {
		if testing.Verbose() {
			fmt.Printf("Start test %d, numChunks: %d\n", i, test.numChunks)
		}
		myChunks := make([]swarm.Chunk, test.numChunks)
		otherChunks := make([]swarm.Chunk, test.numChunks)
		otherPeersChunks := make(map[int][]swarm.Chunk)
		otherPeersProof := make(map[int][]uint64)
		for i := 0; i < test.numChunks; i++ {
			var myChunk swarm.Chunk
			var otherChunk swarm.Chunk
			if test.numChunks <= 100000 {
				myChunk = testingc.GenerateTestRandomChunk()
				otherChunk = testingc.GenerateTestRandomChunk()
			} else {
				addr := make([]byte, 16)
				binary.LittleEndian.PutUint64(addr[:8], uint64(rand.Uint64()))
				binary.LittleEndian.PutUint64(addr[8:], uint64(rand.Uint64()))
				myChunk = swarm.NewChunk(swarm.NewAddress(addr[:8]), addr[:8])
				otherChunk = swarm.NewChunk(swarm.NewAddress(addr[8:]), addr[8:])
			}
			myChunks[i] = myChunk
			otherChunks[i] = otherChunk

		}
		if testing.Verbose() {
			fmt.Printf("Mychunks: %v\n", myChunks)
			fmt.Printf("OtherChunks: %v\n", otherChunks)
		}

		myChunksChan := make(chan swarm.Chunk)
		myDoneChan := make(chan struct{})
		otherChunksChan := make(chan swarm.Chunk)
		otherDoneChan := make(chan struct{})
		go func() {
			for i := 0; i < test.numChunks; i++ {
				myChunksChan <- myChunks[i]
			}
			myDoneChan <- struct{}{}
		}()
		go func() {
			for i := 0; i < test.numChunks; i++ {
				otherChunksChan <- otherChunks[i]
			}
			otherDoneChan <- struct{}{}
		}()

		snipsService, _, _ := createMockService(
			t,
			swarm.ZeroAddress, nil,
			8, 5,
			nil,
			nil,
			nil,
			nil,
		)

		otherBaseProof, _, err := snipsService.BuildBaseProof(context.Background(), otherChunksChan, otherDoneChan, test.nonce)
		myBaseProof, myChunkProofToId, err := snipsService.BuildBaseProof(context.Background(), myChunksChan, myDoneChan, test.nonce)

		if err != nil {
			t.Fatal(err)
		}

		for _, avail := range test.availability {
			numChunks := avail * test.numChunks / 100
			otherPeersChunks[avail] = make([]swarm.Chunk, test.numChunks)
			otherPeersProof[avail] = make([]uint64, test.numChunks)

			copy(otherPeersChunks[avail][:numChunks], myChunks[:numChunks])
			copy(otherPeersChunks[avail][numChunks:], otherChunks[numChunks:])
			copy(otherPeersProof[avail][:numChunks], myBaseProof[:numChunks])
			copy(otherPeersProof[avail][numChunks:], otherBaseProof[numChunks:])

			if testing.Verbose() {
				fmt.Printf("myChunks: %v, otherChunks[%v]: %v, Numchunks: %d\n\n", myChunks[0:numChunks], avail, otherChunks[numChunks:], numChunks)
				fmt.Printf("otherPeersChunks: %v\n\n\n", otherPeersChunks[avail])
			}
		}

		mapValToChunkId, mapChunkIdToProof := snipsService.GetProofMapping()
		_ = mapChunkIdToProof

		bbhash, err := snipsService.CreateMPHF(context.Background(), myBaseProof, myChunkProofToId, test.nonce)
		_ = bbhash
		_ = mapValToChunkId
		if err != nil {
			t.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
		}

		// Generate false chunk proofs and check how many matches we get
		foundHashVals := make(map[uint64]struct{})
		nonceByte := make([]byte, 8)
		binary.LittleEndian.PutUint64(nonceByte, test.nonce)

		for _, avail := range test.availability {
			var zeroValues int
			expectedChunks := snips.NewExpectedVals(test.numChunks)
			for i := 0; i < test.numChunks; i++ {
				hashVal := bbhash.Find(otherPeersProof[avail][i])
				if hashVal == 0 {
					zeroValues++
				} else {
					expectedChunks.Hit(hashVal)
				}
			}

			// Only requesting missing
			fmt.Printf("To Request: %d, Numchunks: %d, Chunks Missing After: %d, Excess Bandwidth: %d, Duplicate proofs (not request) %v, Zeroes: %v, Chunk Loss: %d\n", len(expectedChunks.GetMissingChunks()), test.numChunks, max(0, (test.numChunks-(avail*test.numChunks/100))-len(expectedChunks.GetMissingChunks())), max(0, -1*((test.numChunks-(avail*test.numChunks/100))-len(expectedChunks.GetMissingChunks()))), len(expectedChunks.DuplicateProof()), zeroValues, 100-avail)
		}
		var zeroValues int
		for i := 0; i < test.numChunks; i++ {
			falseProof := rand.Uint64()

			hashVal := bbhash.Find(falseProof)
			if hashVal == 0 {
				zeroValues++
			}
			if _, ok := foundHashVals[hashVal]; ok {
				t.Errorf("foundHashVals([hashVal] duplicate found. input %v, numchunks: %d", hashVal, test.numChunks)
			}
		}

		fmt.Printf("Size bbhash: %v\n", bbhash.MarshalBinarySize())
		fmt.Printf("Zero values: %d, numChunks: %d\n", zeroValues, test.numChunks)
	}
}

// Testing accuracy with range checks by verifying the proof with as many chunks as it contains.
// The verifier only has chunks that are equal to the prover's.'
func TestProofAccuracyWithPerfectRange(t *testing.T) {
	return // FIXME: This is only used for simulations.
	t.Parallel()
	tests := []struct {
		numChunks    int
		nonce        uint64
		availability []int // Availability represeents how many chunks another peer has. availability = 0 means no chunks, availability = 100 means all chunks.
	}{
		// {numChunks: 1, nonce: 9999, availability: []int{100}},
		// {numChunks: 10, nonce: 9999, availability: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}},
		{numChunks: 100, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}},
		{numChunks: 1000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}},
		{numChunks: 10000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}},
	}
	if !testing.Short() {
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 100000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}})
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 262144, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}}) // 1 GB
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 2621440, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 10 GB
		tests = append(tests, struct {
			numChunks    int
			nonce        uint64
			availability []int
		}{numChunks: 26214400, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 100 GB)
	}

	for i, test := range tests {
		if testing.Verbose() {
			fmt.Printf("Start test %d, numChunks: %d\n", i, test.numChunks)
		}
		myChunks := make([]swarm.Chunk, test.numChunks)
		otherPeersChunks := make(map[int][]swarm.Chunk)
		otherPeersProof := make(map[int][]uint64)
		for i := 0; i < test.numChunks; i++ {
			var myChunk swarm.Chunk
			if test.numChunks <= 100000 {
				myChunk = testingc.GenerateTestRandomChunk()
			} else {
				addr := make([]byte, 16)
				binary.LittleEndian.PutUint64(addr[:8], uint64(rand.Uint64()))
				binary.LittleEndian.PutUint64(addr[8:], uint64(rand.Uint64()))
				myChunk = swarm.NewChunk(swarm.NewAddress(addr[:8]), addr[:8])
			}
			myChunks[i] = myChunk

		}
		if testing.Verbose() {
			fmt.Printf("Mychunks: %v\n", myChunks)
		}

		myChunksChan := make(chan swarm.Chunk)
		myDoneChan := make(chan struct{})
		go func() {
			for i := 0; i < test.numChunks; i++ {
				myChunksChan <- myChunks[i]
			}
			myDoneChan <- struct{}{}
		}()

		snipsService, closestStorer, _ := createMockService(
			t,
			swarm.ZeroAddress, nil,
			8, 5,
			nil,
			nil,
			nil,
			nil,
		)

		_ = closestStorer
		myBaseProof, myChunkProofToId, err := snipsService.BuildBaseProof(context.Background(), myChunksChan, myDoneChan, test.nonce)

		if err != nil {
			t.Fatal(err)
		}

		for _, avail := range test.availability {
			numChunks := avail * test.numChunks / 100
			otherPeersChunks[avail] = make([]swarm.Chunk, numChunks)
			otherPeersProof[avail] = make([]uint64, numChunks)

			copy(otherPeersChunks[avail][:numChunks], myChunks[:numChunks])
			copy(otherPeersProof[avail][:numChunks], myBaseProof[:numChunks])

			if testing.Verbose() {
				fmt.Printf("myChunks: %v,  Numchunks: %d\n\n", myChunks[0:numChunks], numChunks)
				fmt.Printf("otherPeersChunks: %v\n\n\n", otherPeersChunks[avail])
			}
		}

		mapValToChunkId, mapChunkIdToProof := snipsService.GetProofMapping()
		_ = mapChunkIdToProof

		bbhash, err := snipsService.CreateMPHF(context.Background(), myBaseProof, myChunkProofToId, test.nonce)
		_ = bbhash
		_ = mapValToChunkId
		if err != nil {
			t.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
		}

		// Generate false chunk proofs and check how many matches we get
		foundHashVals := make(map[uint64]struct{})
		nonceByte := make([]byte, 8)
		binary.LittleEndian.PutUint64(nonceByte, test.nonce)

		for _, avail := range test.availability {
			var zeroValues int
			expectedChunks := snips.NewExpectedVals(test.numChunks)
			numChunks := avail * test.numChunks / 100
			for i := 0; i < numChunks; i++ {
				hashVal := bbhash.Find(otherPeersProof[avail][i])
				if hashVal == 0 {
					zeroValues++
				} else {
					expectedChunks.Hit(hashVal)
				}
			}

			// Only requesting missing
			fmt.Printf("To Request: %d, Numchunks: %d, Chunks Missing After: %d, Excess Bandwidth: %d, Duplicate proofs (not request) %v, Zeroes: %v, Chunk Loss: %d\n", len(expectedChunks.GetMissingChunks()), test.numChunks, max(0, (test.numChunks-(avail*test.numChunks/100))-len(expectedChunks.GetMissingChunks())), max(0, -1*((test.numChunks-(avail*test.numChunks/100))-len(expectedChunks.GetMissingChunks()))), len(expectedChunks.DuplicateProof()), zeroValues, 100-avail)
		}
		var zeroValues int
		for i := 0; i < test.numChunks; i++ {
			falseProof := rand.Uint64()

			hashVal := bbhash.Find(falseProof)
			if hashVal == 0 {
				zeroValues++
			}
			if _, ok := foundHashVals[hashVal]; ok {
				t.Errorf("foundHashVals([hashVal] duplicate found. input %v, numchunks: %d", hashVal, test.numChunks)
			}
		}

		fmt.Printf("Size bbhash: %v\n", bbhash.MarshalBinarySize())
		fmt.Printf("Zero values: %d, numChunks: %d\n", zeroValues, test.numChunks)
	}
}

// Testing accuracy without range checks by verifying the proof with as many chunks as it contains.
// The verifier has some chunks that is equal to the prover, and the other chunks are random.
func TestProofAccuracyWithRange(t *testing.T) {
	return // FIXME: This is only used for simulations.
	t.Parallel()
	tests := []struct {
		numChunks     int
		nonce         uint64
		availability  []int // Availability represeents how many chunks another peer has. Availability = 0 means no chunks, Availability = 100 means all chunks.
		rangeAddition []int // Range Addition represents how many chunks the other peer has added within the range.
	}{
		{numChunks: 1, nonce: 9999, availability: []int{100}},
		{numChunks: 10, nonce: 9999, availability: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, rangeAddition: []int{0, 5, 10}},
		{numChunks: 100, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{numChunks: 1000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{numChunks: 10000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	if !testing.Short() {
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 100000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}})
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 262144, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}) // 1 GB
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 2621440, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 10 GB
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 26214400, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 100 GB)
	}

	for i, test := range tests {
		if testing.Verbose() {
			fmt.Printf("Start test %d, numChunks: %d\n", i, test.numChunks)
		}
		myChunks := make([]swarm.Chunk, test.numChunks)
		otherChunks := make([]swarm.Chunk, test.numChunks)
		otherPeersChunks := make(map[int]map[int][]swarm.Chunk)
		otherPeersProof := make(map[int]map[int][]uint64)
		for i := 0; i < test.numChunks; i++ {
			var myChunk swarm.Chunk
			var otherChunk swarm.Chunk
			if test.numChunks <= 100000 {
				myChunk = testingc.GenerateTestRandomChunk()
				otherChunk = testingc.GenerateTestRandomChunk()
			} else {
				addr := make([]byte, 16)
				binary.LittleEndian.PutUint64(addr[:8], uint64(rand.Uint64()))
				binary.LittleEndian.PutUint64(addr[8:], uint64(rand.Uint64()))
				myChunk = swarm.NewChunk(swarm.NewAddress(addr[:8]), addr[:8])
				otherChunk = swarm.NewChunk(swarm.NewAddress(addr[8:]), addr[8:])
			}
			myChunks[i] = myChunk
			otherChunks[i] = otherChunk

		}
		if testing.Verbose() {
			fmt.Printf("MyChunks: %v\nOtherChunks: %v\n\n", myChunks, otherChunks)
		}

		myChunksChan := make(chan swarm.Chunk)
		myDoneChan := make(chan struct{})
		otherChunksChan := make(chan swarm.Chunk)
		otherDoneChan := make(chan struct{})
		go func() {
			for i := 0; i < test.numChunks; i++ {
				myChunksChan <- myChunks[i]
			}
			myDoneChan <- struct{}{}
		}()
		go func() {
			for i := 0; i < test.numChunks; i++ {
				otherChunksChan <- otherChunks[i]
			}
			otherDoneChan <- struct{}{}
		}()

		snipsService, _, _ := createMockService(
			t,
			swarm.ZeroAddress, nil,
			8, 5,
			nil,
			nil,
			nil,
			nil,
		)
		otherBaseProof, _, err := snipsService.BuildBaseProof(context.Background(), otherChunksChan, otherDoneChan, test.nonce)
		myBaseProof, myChunkProofToId, err := snipsService.BuildBaseProof(context.Background(), myChunksChan, myDoneChan, test.nonce)

		if err != nil {
			t.Fatal(err)
		}

		for _, avail := range test.availability {
			availChunks := avail * test.numChunks / 100
			otherPeersChunks[avail] = make(map[int][]swarm.Chunk)
			otherPeersProof[avail] = make(map[int][]uint64)

			for _, rangeadd := range test.rangeAddition {
				rangeAddChunks := rangeadd * test.numChunks / 100
				otherPeersChunks[avail][rangeadd] = make([]swarm.Chunk, availChunks+rangeAddChunks)
				otherPeersProof[avail][rangeadd] = make([]uint64, availChunks+rangeAddChunks)

				copy(otherPeersChunks[avail][rangeadd][:availChunks], myChunks[:availChunks])
				copy(otherPeersChunks[avail][rangeadd][availChunks:], otherChunks[rangeAddChunks:])
				copy(otherPeersProof[avail][rangeadd][:availChunks], myBaseProof[:availChunks])
				copy(otherPeersProof[avail][rangeadd][availChunks:], otherBaseProof[rangeAddChunks:])

				if testing.Verbose() {
					fmt.Printf("otherPeersChunks[%v][%v]: %v\n\n\n", avail, rangeadd, otherPeersChunks[avail][rangeadd])
				}
			}
		}

		mapValToChunkId, mapChunkIdToProof := snipsService.GetProofMapping()
		_ = mapChunkIdToProof

		bbhash, err := snipsService.CreateMPHF(context.Background(), myBaseProof, myChunkProofToId, test.nonce)
		_ = bbhash
		_ = mapValToChunkId
		if err != nil {
			t.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
		}

		// Generate false chunk proofs and check how many matches we get
		nonceByte := make([]byte, 8)
		binary.LittleEndian.PutUint64(nonceByte, test.nonce)

		for _, avail := range test.availability {
			availChunks := avail * test.numChunks / 100
			fmt.Printf("\n\n------------------Availability: %d------------------\n", avail)
			for _, rangeadd := range test.rangeAddition {
				rangeAddChunks := rangeadd * test.numChunks / 100
				var zeroValues int
				expectedChunks := snips.NewExpectedVals(test.numChunks)
				for i := 0; i < availChunks+rangeAddChunks; i++ {
					hashVal := bbhash.Find(otherPeersProof[avail][rangeadd][i])
					if hashVal == 0 {
						zeroValues++
					} else {
						expectedChunks.Hit(hashVal)
					}
				}

				// Only requesting missing
				fmt.Printf("To Request: %d, Numchunks: %d, Chunks Missing After: %d, Excess Bandwidth: %d, Duplicate proofs (not request) %v, Zeroes: %v, Chunk Loss: %d, Range Add: %d\n", len(expectedChunks.GetMissingChunks()), test.numChunks, max(0, (test.numChunks-(avail*test.numChunks/100))-len(expectedChunks.GetMissingChunks())), max(0, -1*((test.numChunks-(avail*test.numChunks/100))-len(expectedChunks.GetMissingChunks()))), len(expectedChunks.DuplicateProof()), zeroValues, 100-avail, rangeAddChunks)
			}
		}

		fmt.Printf("Size bbhash: %v\n", bbhash.MarshalBinarySize())
	}
}

// Testing SNIPS interaction by seeing how many times the prover and verifier must exchange proofs before both have the same set of chunks.
func TestSNIPSinteraction(t *testing.T) {
	return // FIXME: This is only used for simulations.
	t.Parallel()
	tests := []struct {
		numChunks     int
		nonce         uint64
		availability  []int // Availability represeents how many chunks another peer has. Availability = 0 means no chunks, Availability = 100 means all chunks.
		rangeAddition []int // Range Addition represents how many chunks the other peer has added within the range.
	}{
		{numChunks: 1, nonce: 9999, availability: []int{100}},
		{numChunks: 10, nonce: 9999, availability: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, rangeAddition: []int{0, 5, 10}},
		{numChunks: 100, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{numChunks: 1000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{numChunks: 10000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	if !testing.Short() {
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 100000, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}})
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 262, nonce: 9999, availability: []int{1, 5, 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 5, 10, 50, 100}}) // 1 MB
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 2621, nonce: 9999, availability: []int{1, 5, 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 5, 10, 50, 100}}) // 10 MB
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 26214, nonce: 9999, availability: []int{1, 5, 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 96, 97, 98, 99, 100}, rangeAddition: []int{0, 5, 10, 50, 100}}) // 100 MB
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 262144, nonce: 9999, availability: []int{98, 99, 100}, rangeAddition: []int{0, 5, 10, 50, 100}}) // 1 GB
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 2621440, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 10 GB
		tests = append(tests, struct {
			numChunks     int
			nonce         uint64
			availability  []int
			rangeAddition []int
		}{numChunks: 26214400, nonce: 9999, availability: []int{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}) // 100 GB)
	}

	for i, test := range tests {

		// BEGIN initial setup
		if testing.Verbose() {
			fmt.Printf("Start test %d, numChunks: %d\n", i, test.numChunks)
		}
		myChunks := make([]swarm.Chunk, test.numChunks)
		otherChunks := make([]swarm.Chunk, test.numChunks)
		otherPeersChunks := make(map[int]map[int][]swarm.Chunk)
		otherPeersProof := make(map[int]map[int][]uint64)
		for i := 0; i < test.numChunks; i++ {
			var myChunk swarm.Chunk
			var otherChunk swarm.Chunk
			if test.numChunks <= 100000 {
				myChunk = testingc.GenerateTestRandomChunk()
				otherChunk = testingc.GenerateTestRandomChunk()
			} else {
				addr := make([]byte, 16)
				binary.LittleEndian.PutUint64(addr[:8], uint64(rand.Uint64()))
				binary.LittleEndian.PutUint64(addr[8:], uint64(rand.Uint64()))
				myChunk = swarm.NewChunk(swarm.NewAddress(addr[:8]), addr[:8])
				otherChunk = swarm.NewChunk(swarm.NewAddress(addr[8:]), addr[8:])
			}
			myChunks[i] = myChunk
			otherChunks[i] = otherChunk

		}
		if testing.Verbose() {
			fmt.Printf("MyChunks: %v\nOtherChunks: %v\n\n", myChunks, otherChunks)
		}

		myChunksChan := make(chan swarm.Chunk)
		myDoneChan := make(chan struct{})
		otherChunksChan := make(chan swarm.Chunk)
		otherDoneChan := make(chan struct{})
		go func() {
			for i := 0; i < test.numChunks; i++ {
				myChunksChan <- myChunks[i]
			}
			myDoneChan <- struct{}{}
		}()
		go func() {
			for i := 0; i < test.numChunks; i++ {
				otherChunksChan <- otherChunks[i]
			}
			otherDoneChan <- struct{}{}
		}()

		mySnipsService, _, _ := createMockService(
			t,
			swarm.ZeroAddress, nil,
			8, 5,
			nil,
			nil,
			nil,
			nil,
		)
		otherSnipsService, _, _ := createMockService(
			t,
			swarm.ZeroAddress, nil,
			8, 5,
			nil,
			nil,
			nil,
			nil,
		)
		otherBaseProof, otherChunkProofToId, err := otherSnipsService.BuildBaseProof(context.Background(), otherChunksChan, otherDoneChan, test.nonce)
		myBaseProof, myChunkProofToId, err := mySnipsService.BuildBaseProof(context.Background(), myChunksChan, myDoneChan, test.nonce)

		if err != nil {
			t.Fatal(err)
		}

		for _, avail := range test.availability {
			availChunks := avail * test.numChunks / 100
			otherPeersChunks[avail] = make(map[int][]swarm.Chunk)
			otherPeersProof[avail] = make(map[int][]uint64)

			for _, rangeadd := range test.rangeAddition {
				rangeAddChunks := rangeadd * test.numChunks / 100
				otherPeersChunks[avail][rangeadd] = make([]swarm.Chunk, availChunks+rangeAddChunks)
				otherPeersProof[avail][rangeadd] = make([]uint64, availChunks+rangeAddChunks)

				copy(otherPeersChunks[avail][rangeadd][:availChunks], myChunks[:availChunks])
				copy(otherPeersChunks[avail][rangeadd][availChunks:], otherChunks[:rangeAddChunks])
				copy(otherPeersProof[avail][rangeadd][:availChunks], myBaseProof[:availChunks])
				copy(otherPeersProof[avail][rangeadd][availChunks:], otherBaseProof[:rangeAddChunks])

				if testing.Verbose() {
					fmt.Printf("otherPeersChunks[%v][%v]: %v\n\n\n", avail, rangeadd, otherPeersChunks[avail][rangeadd])
				}
			}
		}

		myMapValToChunkId, myMapChunkIdToProof := mySnipsService.GetProofMapping()
		otherMapValToChunkId, otherMapChunkIdToProof := otherSnipsService.GetProofMapping()
		_, _, _, _ = myMapValToChunkId, myMapChunkIdToProof, otherMapValToChunkId, otherMapChunkIdToProof

		if err != nil {
			t.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
		}

		// END initial setup
		nonceByte := make([]byte, 8)
		binary.LittleEndian.PutUint64(nonceByte, test.nonce)
		for _, avail := range test.availability {
			if testing.Verbose() {
				fmt.Printf("\n\n------------------Availability: %d------------------\n", avail)
			}
			for _, rangeadd := range test.rangeAddition {
				rangeAddChunks := rangeadd * test.numChunks / 100
				if testing.Verbose() {
					fmt.Printf("\n\n------------------Added chunks: %d------------------\n", rangeAddChunks)
				}
				interactions, myBandwidth, otherBandwidth, myBandwidthPullsync, otherBandwidthPullsync := 0, 0, 0, 0, 0

				myBaseProof = myBaseProof[:test.numChunks] // Reset base proof since we are adding to it when the other peer has new chunks.
				chunksExist := len(myBaseProof) + rangeAddChunks

				myBandwidthPullsync += 32 * 4    // The cursor array is : 32 Proximity Order bins, one integer (4 bytes) per bin.
				otherBandwidthPullsync += 32 * 4 // The cursor array is : 32 Proximity Order bins, one integer (4 bytes) per bin.

				myBandwidthPullsync += 6 * (20)    // Assume we send 6 range requests. 20 = 4 bytes for binID, 8 bytes for From, 8 bytes for To.
				otherBandwidthPullsync += 6 * (20) // Assume we send 6 range requests. 20 = 4 bytes for binID, 8 bytes for From, 8 bytes for To.

				// myBandwidthPullsync += test.numChunks * swarm.HashSize                           // Size of proof
				// otherBandwidthPullsync += len(otherPeersProof[avail][rangeadd]) * swarm.HashSize // Size of proof

				myBandwidthPullsync += (chunksExist - len(otherPeersProof[avail][rangeadd])) * swarm.HashSize // Missing chunk identifiers
				otherBandwidthPullsync += (chunksExist - test.numChunks) * swarm.HashSize                     // Missing chunk identifiers

				myBandwidthPullsync += (chunksExist - test.numChunks) / 8                           // Missing chunk identifiers bitmap
				otherBandwidthPullsync += (chunksExist - len(otherPeersProof[avail][rangeadd])) / 8 // Missing chunk identifiers bitmap

				for {
					if interactions >= 10 {
						t.Fatal("Too many interactions")
					}
					interactions++

					myBBhash, err := mySnipsService.CreateMPHF(context.Background(), myBaseProof, myChunkProofToId, test.nonce)
					if err != nil {
						t.Fatal(err)
					}
					fmt.Printf("My BBhash: %v\n", myBBhash.String())
					otherBBhash, err := otherSnipsService.CreateMPHF(context.Background(), otherPeersProof[avail][rangeadd], otherChunkProofToId, test.nonce)
					if err != nil {
						t.Fatal(err)
					}

					// I verify using others' proof (otherBBhash) and requests some chunks.
					myExpectedChunks := snips.NewExpectedVals(len(otherPeersProof[avail][rangeadd]))
					for i := 0; i < len(myBaseProof); i++ {
						hashVal := otherBBhash.Find(myBaseProof[i])
						if hashVal != 0 {
							myExpectedChunks.Hit(hashVal)
						}
					}

					// Other verifies using my proof (myBBhash), and requests some chunks.
					// Add these chunks to my database.
					otherExpectedChunks := snips.NewExpectedVals(len(myBaseProof))
					for i := 0; i < len(otherPeersProof[avail][rangeadd]); i++ {
						hashVal := myBBhash.Find(otherPeersProof[avail][rangeadd][i])
						if hashVal != 0 {
							otherExpectedChunks.Hit(hashVal)
						}
					}
					myNumMissingChunks := len(myExpectedChunks.GetMissingChunks())
					otherNumMissingChunks := len(otherExpectedChunks.GetMissingChunks())

					if otherNumMissingChunks == 0 && myNumMissingChunks == 0 {

						if testing.Verbose() {
							fmt.Printf("|MINE| To Request: %d, Chunk Exist: %d, Chunks Missing After: %d, Excess Bandwidth: %d, Duplicate proofs (not request) %v, Chunk Loss: %d, Range Add: %d, Interactions: %d, Bandwidth: %d, PullsyncBW: %d\n", len(myExpectedChunks.GetMissingChunks()), chunksExist, chunksExist-len(myBaseProof)-len(myExpectedChunks.GetMissingChunks()), max(0, -1*((test.numChunks-(avail*test.numChunks/100))-len(myExpectedChunks.GetMissingChunks()))), len(myExpectedChunks.DuplicateProof()), 100-avail, rangeAddChunks, interactions, myBandwidth, myBandwidthPullsync)

							fmt.Printf("|OTHER| To Request: %d, Chunks Exist: %d, Chunks Missing After: %d, Excess Bandwidth: %d, Duplicate proofs (not request) %v, Chunk Loss: %d, Range Add: %d, Interactions: %d, Bandwidth: %d, PullsyncBW: %d\n", len(otherExpectedChunks.GetMissingChunks()), chunksExist, chunksExist-len(otherPeersProof[avail][rangeadd])-len(otherExpectedChunks.GetMissingChunks()), max(0, -1*((test.numChunks-(avail*test.numChunks/100))-len(otherExpectedChunks.GetMissingChunks()))), len(otherExpectedChunks.DuplicateProof()), 100-avail, rangeAddChunks, interactions, otherBandwidth, otherBandwidthPullsync)
						}

						// The combined stats
						// fmt.Printf("%v,%v,%v,%v,%v,%v,%v,%v,%v\n", chunksExist, test.numChunks, avail, rangeadd, interactions, myBandwidth, otherBandwidth, myBandwidthPullsync, otherBandwidthPullsync)
						break // We are done. No missing chunks.
					}

					// Both sides continue sending proofs until there are no missing chunks.
					myBandwidth += len(myExpectedChunks.GetMissingChunks()) / 8 // Request using bit vector. Assume 1 bit per request. (Though not completely accurate)
					myBandwidth += int(myBBhash.MarshalBinarySize())            // Size of proof
					myBandwidth += 2 * swarm.HashSize                           // Size of the range (two identifiers)

					otherBandwidth += len(otherExpectedChunks.GetMissingChunks()) / 8 // Request using bit vector. Assume 1 bit per request. (Though not completely accurate)
					otherBandwidth += int(otherBBhash.MarshalBinarySize())            // Size of proof
					otherBandwidth += 2 * swarm.HashSize                              // Size of the range (two identifiers)

					for _, val := range myExpectedChunks.GetMissingChunks() {
						downloadAddr := otherMapValToChunkId[test.nonce][uint64(val)]
						downloadProof := otherMapChunkIdToProof[test.nonce][downloadAddr.ByteString()]
						myBaseProof = append(myBaseProof, downloadProof)
						myChunkProofToId[downloadProof] = downloadAddr
					}

					// For each interation, calculate the new bbhashes
					for _, val := range otherExpectedChunks.GetMissingChunks() {
						downloadAddr := myMapValToChunkId[test.nonce][uint64(val)]
						downloadProof := myMapChunkIdToProof[test.nonce][downloadAddr.ByteString()]
						otherPeersProof[avail][rangeadd] = append(otherPeersProof[avail][rangeadd], downloadProof)
						otherChunkProofToId[downloadProof] = downloadAddr
					}
				}
			}
		}

		//fmt.Printf("Size bbhash: %v\n", bbhash.MarshalBinarySize())
	}
}

// Test_CollisionRate finds specific collision rate.
// Specific collision is when a randomly generated chunk maps to a specific value of the proof.
// This is to simulate the probability of "False Consistencies" in SNIPS.
// For a false consistency to occur, the verifier needs to be convinced it posesses all chunks in the proof, while it actually does not.
// False Positive: When a chunk that was not part of the original set gets a "might be in the set" response when querying the proof.
// Collision:  More than one chunk evaluates to the same value (output) of the proof. Having a collision means that you have at least one false positive.
// False Consistency: When a peer believes it has no missing chunks after querying a proof. However, at least one chunk used to query the proof gave a false positive. AND the value of the false positive exactly matches the last remaining expected value. (This is what figure 7 shows) (Now incorrectly labeled "Rate of false positives")
// Duplicate: Same as collision. All occurrences should be replaced.
// This can occur, if another chunk (not in the proof) maps to the same value as the actual chunk (that the verifier does not have).
func Test_CollisionRate(t *testing.T) {
	t.Parallel()
	var iterations float64 = 1_000_000_000
	var printInterval int = int(iterations) / 100
	tests := []struct {
		numChunks  int
		nonce      uint64
		iterations float64
	}{
		{numChunks: 1, nonce: 9999, iterations: iterations},             // 10 ** 0
		{numChunks: 10, nonce: 9999, iterations: iterations},            // 10 ** 1
		{numChunks: 100, nonce: 9999, iterations: iterations},           // 10 ** 2
		{numChunks: 1_000, nonce: 9999, iterations: iterations},         // 10 ** 3
		{numChunks: 10_000, nonce: 9999, iterations: iterations},        // 10 ** 4
		{numChunks: 100_000, nonce: 9999, iterations: iterations},       // 10 ** 5
		{numChunks: 1_000_000, nonce: 9999, iterations: iterations},     // 10 ** 6
		{numChunks: 10_000_000, nonce: 9999, iterations: iterations},    // 10 ** 7
		{numChunks: 100_000_000, nonce: 9999, iterations: iterations},   // 10 ** 8
		{numChunks: 1_000_000_000, nonce: 9999, iterations: iterations}, // 10 ** 9
		// {numChunks: 10_000_000_000, nonce: 9999, iterations: 100_000_000_000},
		// {numChunks: 100_000_000_000, nonce: 9999, iterations: 10_000_000_000_000},
		// {numChunks: 100_000_000_000, nonce: 9999, iterations: 10_000_000_000_000},
		// {numChunks: 1_000_000_000_000, nonce: 9999, iterations: 10_000_000_000_000},
	}

	zeroAddr := swarm.NewAddress(make([]byte, swarm.HashSize))

	for i, test := range tests {
		if testing.Verbose() {
			fmt.Printf("Start test %d, numChunks: %d\n", i, test.numChunks)
		}

		chunkProofMapAddr := make(map[uint64]swarm.Address)
		chunkProofMap := make(map[uint64]int)
		chunkProofArr := make([]uint64, test.numChunks)
		for i := 0; i < test.numChunks; i++ {
			chunkproof := rand.Uint64()
			if _, ok := chunkProofMap[chunkproof]; ok {
				i--
				continue
			}
			chunkProofArr[i] = chunkproof
			chunkProofMap[chunkproof] = i
			chunkProofMapAddr[chunkproof] = zeroAddr
		}

		snipsService, _, _ := createMockService(t, swarm.ZeroAddress, nil, 8, 5, nil, nil, nil, nil)

		bbhash, err := snipsService.CreateMPHF(context.Background(), chunkProofArr, chunkProofMapAddr, test.nonce)
		if err != nil {
			t.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
		}

		var falsepostives, falseconsistency float64 = 0, 0
		for i := float64(0); i < test.iterations; i++ {
			rndChunkProof := rand.Uint64()
			rndValueInProof := rand.Uint64()%uint64(test.numChunks) + 1
			if _, ok := chunkProofMap[rndValueInProof]; ok {
				i-- // Testing against a duplicate chunk. Skip this iteration as this cannot happen.
				continue
			}
			testHashVal := bbhash.Find(rndChunkProof)
			if testHashVal != 0 {
				falsepostives++
			}
			if testHashVal == rndValueInProof {
				falseconsistency++
			}
			if testing.Verbose() && int(i)%printInterval == 0 {
				fmt.Printf("Time: %s. Iteration: %d/%d\n", time.Now(), int(i)/printInterval, int(test.iterations)/printInterval)
			}
		}

		fmt.Printf("Numchunks: %v, False Positives: %8.0f, False Positive rate: %0.5f, False Consistency: %v, False Consistency rate: %v\n", test.numChunks, falsepostives, (falsepostives/test.iterations)*100, falseconsistency, (falseconsistency/test.iterations)*100)
	}
}

func BenchmarkProofCreateVerify(b *testing.B) {
	tests := []struct {
		numChunks int
	}{
		{numChunks: 257},
		{numChunks: 2561},
		{numChunks: 25604},
		{numChunks: 256039},
		{numChunks: 2560390},
		// {numChunks: 25603900},
		// {numChunks: 100_000_000},
		// {numChunks: 1_000_000_000},
	}
	chunksChan := make(chan swarm.Chunk)
	doneChan := make(chan struct{})
	go func() {
		for i := 0; i < 2560390; i++ {
			addr := make([]byte, 8)
			binary.LittleEndian.PutUint64(addr, uint64(i))
			chunk := swarm.NewChunk(swarm.NewAddress(addr), addr)
			chunksChan <- chunk
		}
		doneChan <- struct{}{}
	}()

	snipsService, _, _ := createMockService(b, swarm.ZeroAddress, nil, 8, 50, nil, nil, nil, nil)

	baseProof, chunkProofToId, err := snipsService.BuildBaseProof(context.Background(), chunksChan, doneChan, 123)
	if err != nil {
		b.Fatal(err)
	}
	for i, test := range tests {
		b.Run(fmt.Sprintf("numChunks=%d", test.numChunks), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				bbhash, err := snipsService.CreateMPHF(context.Background(), baseProof[:test.numChunks], chunkProofToId, uint64(i))
				if err != nil {
					b.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
				}
				if bbhash.MarshalBinarySize() < 100 {
					// b.Fatalf("Error generating SNIPS proof. Numchunks: %d, Err: %v", test.numChunks, err)
				}
			}
		})
	}
}

type testSimilPerf struct {
	SimilarityRate  int
	UniqueChunks    int
	TotalChunks     int
	PeerChunks      []int
	Bandwidth       []int64
	ProofCounter    []int
	MaintainCounter []int
}

func waitUntilSyncFinishByTiming(dbA, dbB *localstore.DB, minWait, maxWait time.Duration) error {
	indicesA, _ := dbA.DebugIndices()
	indicesB, _ := dbB.DebugIndices()
	begin := time.Now()
	for {
		<-time.After(minWait)
		if time.Since(begin) > maxWait {
			return fmt.Errorf("timeout waiting for sync")
		}
		newIndicesA, _ := dbA.DebugIndices()
		newIndicesB, _ := dbB.DebugIndices()
		fmt.Printf("Old A: %d, New A: %d, Old B: %d, New B: %d\n", indicesA["retrievalDataIndex"], newIndicesA["retrievalDataIndex"], indicesB["retrievalDataIndex"], newIndicesB["retrievalDataIndex"])
		if indicesA["retrievalDataIndex"] == newIndicesA["retrievalDataIndex"] && indicesB["retrievalDataIndex"] == newIndicesB["retrievalDataIndex"] {
			return nil
		}
		indicesA, indicesB = newIndicesA, newIndicesB
	}
	return fmt.Errorf("should not reach here")
}

func waitUntilProofSent(peerMetricsA, peerMetricsB *snips.Metrics, wantA, wantB float64, minWait time.Duration, dbA, dbB *localstore.DB) error {
	for {
		<-time.After(minWait)
		newIndicesA, _ := dbA.DebugIndices()
		newIndicesB, _ := dbB.DebugIndices()
		fmt.Printf("Time: %s || size A: %d, size B: %d || proofSent-> want A: %f, got A: %f, want B: %f, got B: %f\n", time.Now().Format("2006-01-02 15:04:05"), newIndicesA["retrievalDataIndex"], newIndicesB["retrievalDataIndex"], wantA, testutil.ToFloat64(peerMetricsA.TotalSNIPSproofRecv), wantB, testutil.ToFloat64(peerMetricsB.TotalSNIPSproofRecv))

		if testutil.ToFloat64(peerMetricsA.TotalSNIPSproofRecv) >= wantA && testutil.ToFloat64(peerMetricsB.TotalSNIPSproofRecv) >= wantB {
			return nil
		}
	}
	return fmt.Errorf("should not reach here")
}

func waitUntilSyncFinish(peerMetricsA, peerMetricsB *snips.Metrics, wantA, wantB float64, minWait time.Duration, dbA, dbB *localstore.DB) error {
	for {
		<-time.After(minWait)
		newIndicesA, _ := dbA.DebugIndices()
		newIndicesB, _ := dbB.DebugIndices()
		fmt.Printf("Time: %s || size A: %d, size B: %d || uplRecv-> want A: %f, got A: %f, want B: %f, got B: %f\n", time.Now().Format("2006-01-02 15:04:05"), newIndicesA["retrievalDataIndex"], newIndicesB["retrievalDataIndex"], wantA, testutil.ToFloat64(peerMetricsA.TotalSNIPSuploaddoneRecv), wantB, testutil.ToFloat64(peerMetricsB.TotalSNIPSuploaddoneRecv))

		if testutil.ToFloat64(peerMetricsA.TotalSNIPSuploaddoneRecv) >= wantA && testutil.ToFloat64(peerMetricsB.TotalSNIPSuploaddoneRecv) >= wantB {
			return nil
		}
	}
	return fmt.Errorf("should not reach here")
}

func TestSimilarityPerformance(t *testing.T) {
	peerA, peerAprivkey, err := generateSwarmAddress()
	if err != nil {
		t.Fatal(err)
	}
	peerA = swarm.MustParseHexAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	peerB, peerBprivkey, err := generateSwarmAddress()
	if err != nil {
		t.Fatal(err)
	}
	peerB = swarm.MustParseHexAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	tests := []struct {
		numChunksA   int
		numChunksB   int
		similarity   []int // How many chunks are the same between A and B. 0 means all chunks are unique. 100 means all chunks are the same. To have have only 1 peer with unique chunks, put similarity to 100 and give set numChunksA larger than numChunksB.
		wait         int   // The maximum amount of seconds to wait for this test to complete.
		repeat       int   // How many times to repeat the test.
		waitDuration time.Duration
	}{
		{numChunksA: 10, numChunksB: 10, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 1 * time.Second},
		{numChunksA: 100, numChunksB: 100, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 1 * time.Second},
		{numChunksA: 1000, numChunksB: 1000, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 1 * time.Second},
		{numChunksA: 10000, numChunksB: 10000, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 5 * time.Second},
		{numChunksA: 100000, numChunksB: 100000, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 10 * time.Second},
		{numChunksA: 256039, numChunksB: 256039, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 10 * time.Second},
		{numChunksA: 25604, numChunksB: 25604, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 1 * time.Second},
		{numChunksA: 2561, numChunksB: 2561, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 1 * time.Second},
		{numChunksA: 257, numChunksB: 257, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 1 * time.Second},
		{numChunksA: 2560390, numChunksB: 2560390, similarity: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}, wait: 7200, repeat: 1, waitDuration: 60 * time.Second},
	}

	for _, test := range tests {
		// Setup random chunks
		rndChunksA, rndChunksB := make([]swarm.Chunk, test.numChunksA), make([]swarm.Chunk, test.numChunksB)
		for i := 0; i < test.numChunksA; i++ {
			if i%100000 == 0 {
				fmt.Printf("Peer A. Time: %s, Chunk: %d/%d\n", time.Now().Format("2006-01-02 15:04:05"), i, test.numChunksA)
			}
			rndChunksA[i] = testingc.GenerateTestRandomChunk()
		}
		for i := 0; i < test.numChunksB; i++ {
			if i%100000 == 0 {
				fmt.Printf("Peer B. Time: %s, Chunk: %d/%d\n", time.Now().Format("2006-01-02 15:04:05"), i, test.numChunksB)
			}
			rndChunksB[i] = testingc.GenerateTestRandomChunk()
		}
		for _, similarity := range test.similarity {
			for z := 0; z < test.repeat; z++ {
				serviceA, dbA, metricsA := createMockService(t, peerA, peerAprivkey, 0, 10, nil, nil, nil, []mockTopology.Option{mockTopology.WithPeers(peerB)})
				pushsyncA := newPushSync(t, peerA, nil, dbA, nil, []mockTopology.Option{mockTopology.WithPeers(peerB)})

				serviceB, dbB, metricsB := createMockService(t, peerB, peerBprivkey, 0, 10, nil, nil, nil, []mockTopology.Option{mockTopology.WithPeers(peerA)})
				pushsyncB := newPushSync(t, peerB, nil, dbB, nil, []mockTopology.Option{mockTopology.WithPeers(peerA)})

				metricsToCollect := []prometheus.Collector{metricsA.TotalSNIPSproofSent, metricsA.SizeOfSNIPSsentMPHF, metricsA.SizeOfSNIPSsentSignature, metricsA.TotalSNIPSmaintainSent, metricsA.SizeOfSNIPSsentMaintainBV, metricsA.TotalSNIPSreqSent, metricsA.TotalSNIPSuploaddoneRecv, metricsA.TotalSNIPSproofRecv}
				prometheus.MustRegister(metricsToCollect...)

				serviceA.AddStreamer(streamtest.New(streamtest.WithBaseAddr(peerA),
					streamtest.WithProtocols(serviceB.Protocol(), pushsyncB.Protocol())))
				serviceB.AddStreamer(streamtest.New(streamtest.WithBaseAddr(peerB),
					streamtest.WithProtocols(serviceA.Protocol(), pushsyncA.Protocol())))

				uniqueChunks := 0
				for i := 0; i < test.numChunksA+test.numChunksB; i++ {
					if i >= test.numChunksA && i >= test.numChunksB {
						break
					}
					uniquecheck := false
					if i < test.numChunksA {
						dbA.Put(context.Background(), storage.ModePutUploadPin, rndChunksA[i])
						uniquecheck = true
					}
					if i < test.numChunksB {
						var chunksB swarm.Chunk
						if rand.Intn(100) < similarity {
							chunksB = rndChunksA[i]
							uniquecheck = true
						} else {
							chunksB = rndChunksB[i]
							uniqueChunks++
						}

						dbB.Put(context.Background(), storage.ModePutUploadPin, chunksB)
					}
					if uniquecheck {
						uniqueChunks++
					}
				}
				// Store chunks for A and B
				// dbA.Put(context.Background(), storage.ModePutUploadPin, chunksA...)
				// dbB.Put(context.Background(), storage.ModePutUploadPin, chunksB...)
				indicesA, _ := dbA.DebugIndices()
				indicesB, _ := dbB.DebugIndices()
				// t.Logf("BEFORE: Peer A now has %d chunks\n", indicesA["retrievalDataIndex"])
				// t.Logf("BEFORE: Peer B now has %d chunks\n", indicesB["retrievalDataIndex"])

				for i := 0; i < 2*test.wait; i++ {
					if similarity == 100 {
						wantUplDoneB := testutil.ToFloat64(metricsB.TotalSNIPSproofRecv) + 1
						serviceA.CreateAndSendProofForMyChunks(context.Background(), 1)
						waitUntilProofSent(metricsA, metricsB, testutil.ToFloat64(metricsA.TotalSNIPSproofRecv), wantUplDoneB, test.waitDuration, dbA, dbB)
						wantUplDoneA := testutil.ToFloat64(metricsA.TotalSNIPSproofRecv) + 1
						serviceB.CreateAndSendProofForMyChunks(context.Background(), 1)
						waitUntilProofSent(metricsA, metricsB, wantUplDoneA, testutil.ToFloat64(metricsB.TotalSNIPSproofRecv), test.waitDuration, dbA, dbB)
						break
					} else {
						wantUplDoneB := testutil.ToFloat64(metricsB.TotalSNIPSuploaddoneRecv) + 1
						serviceA.CreateAndSendProofForMyChunks(context.Background(), 1)
						waitUntilSyncFinish(metricsA, metricsB, testutil.ToFloat64(metricsA.TotalSNIPSuploaddoneRecv), wantUplDoneB, test.waitDuration, dbA, dbB)
						indicesA, _ := dbA.DebugIndices()
						indicesB, _ := dbB.DebugIndices()
						if indicesA["retrievalDataIndex"] == indicesB["retrievalDataIndex"] && indicesA["retrievalDataIndex"] == uniqueChunks {
							break
						}

						wantUplDoneA := testutil.ToFloat64(metricsA.TotalSNIPSuploaddoneRecv) + 1
						serviceB.CreateAndSendProofForMyChunks(context.Background(), 1)
						waitUntilSyncFinish(metricsA, metricsB, wantUplDoneA, testutil.ToFloat64(metricsB.TotalSNIPSuploaddoneRecv), test.waitDuration, dbA, dbB)
						indicesA, _ = dbA.DebugIndices()
						indicesB, _ = dbB.DebugIndices()
						if indicesA["retrievalDataIndex"] == indicesB["retrievalDataIndex"] && indicesA["retrievalDataIndex"] == uniqueChunks {
							break
						}
					}
				}

				bw, proofCounter, maintainCounter := calculateSNIPSBandwidth(metricsA, metricsB)

				testSimilOut := &testSimilPerf{
					SimilarityRate:  similarity,
					UniqueChunks:    uniqueChunks,
					TotalChunks:     test.numChunksA + test.numChunksB,
					PeerChunks:      []int{test.numChunksA, test.numChunksB},
					Bandwidth:       bw,
					ProofCounter:    proofCounter,
					MaintainCounter: maintainCounter,
				}
				outStr, _ := json.Marshal(testSimilOut)
				fmt.Printf("%v\n", string(outStr))

				for _, metricsToCol := range metricsToCollect {
					prometheus.Unregister(metricsToCol)
				}
				indicesA, _ = dbA.DebugIndices()
				indicesB, _ = dbB.DebugIndices()
				if indicesA["retrievalDataIndex"] != indicesB["retrievalDataIndex"] || indicesA["retrievalDataIndex"] != uniqueChunks {
					t.Fatalf("Peer A and B should have the same number of chunks. A has %d, B has %d. Unique: %d", indicesA["retrievalDataIndex"], indicesB["retrievalDataIndex"], uniqueChunks)
				} else {
					// t.Logf("AFTER: Peer A now has %d chunks\n", indicesA["retrievalDataIndex"])
					// t.Logf("AFTER: Peer B now has %d chunks\n", indicesB["retrievalDataIndex"])
				}

				// dbA.Close()
				// dbB.Close()
			}
		}
	}
}

// SNIPS sizes
const (
	snipsSizeProof      = 76
	snipsSizeMPHF       = 1
	snipsSizeBlockhash  = 1
	snipsSizeSignature  = 0
	snipsSizeMaintain   = 8
	snipsSizeMaintainBV = 1
	snipsSizeSNIPSreq   = 8
)

func calculateSNIPSBandwidth(peerMetrics ...*snips.Metrics) ([]int64, []int, []int) {
	bwArr, proofCounter, maintainCounter := make([]int64, len(peerMetrics)), make([]int, len(peerMetrics)), make([]int, len(peerMetrics))
	for i, metric := range peerMetrics {
		bw := .0
		// TotalSNIPSproofSent (8 + 4 + 32 + 32) byte
		bw += testutil.ToFloat64(metric.TotalSNIPSproofSent) * snipsSizeProof

		// SizeOfSNIPSsentMPHF 1 byte
		bw += testutil.ToFloat64(metric.SizeOfSNIPSsentMPHF) * snipsSizeMPHF

		// SizeOfSNIPSsentBlockhash 1 byte
		bw += testutil.ToFloat64(metric.SizeOfSNIPSsentBlockhash) * snipsSizeBlockhash

		// SizeOfSNIPSsentSignature 0 byte
		bw += testutil.ToFloat64(metric.SizeOfSNIPSsentSignature) * snipsSizeSignature

		// TotalSNIPSmaintainSent 8 byte
		bw += testutil.ToFloat64(metric.TotalSNIPSmaintainSent) * snipsSizeMaintain

		// SizeOfSNIPSsentMaintainBV 1 byte
		bw += testutil.ToFloat64(metric.SizeOfSNIPSsentMaintainBV) * snipsSizeMaintainBV

		// TotalSNIPSreqSent 8 byte
		bw += testutil.ToFloat64(metric.TotalSNIPSreqSent) * snipsSizeSNIPSreq

		proofCounter[i] = int(testutil.ToFloat64(metric.TotalSNIPSproofSent))
		maintainCounter[i] = int(testutil.ToFloat64(metric.TotalSNIPSmaintainSent))
		bwArr[i] = int64(bw)
	}
	return bwArr, proofCounter, maintainCounter
}
