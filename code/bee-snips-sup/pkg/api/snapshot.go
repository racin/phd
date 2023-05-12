package api

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/ethersphere/bee/pkg/jsonhttp"
	"github.com/ethersphere/bee/pkg/shed"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/swarm"
)

type snapshotMap map[string]struct{}

type snapshotChecksum struct {
	Localstorage       string `json:"Localstorage"`
	LocalstorageLength int    `json:"LocalstorageLength"`
	Snapshot           string `json:"Snapshot"`
	SnapshotLength     int    `json:"SnapshotLength"`
}

var snapshot snapshotMap

// Creates the checksum of both the snapshot and the localstorage.
// All chunks identifiers are sorted before hashing.
func (s *Service) getSnapshotChecksum(ctx context.Context, snapshot snapshotMap) (*snapshotChecksum, error) {
	keys := make([]string, 0)
	length := 0
	err := s.localStore.IterStoredChunks(ctx, false, func(item shed.Item) error {
		a := swarm.NewAddress(item.Address)
		keys = append(keys, a.String())
		length++
		return nil
	})
	if err != nil {
		s.logger.Error(err, "snapshot: checksum: failed to get localstore %v")
		return nil, err
	}

	hasher := sha256.New()
	// Hash the localstorage first
	sort.Strings(keys)
	for _, key := range keys {
		hasher.Write([]byte(key))
	}

	checksum := &snapshotChecksum{
		SnapshotLength:     len(snapshot),
		Localstorage:       fmt.Sprintf("%x", hasher.Sum(nil)),
		LocalstorageLength: length,
	}

	hasher.Reset()
	keys = make([]string, 0, len(snapshot))

	// Hash the snapshot
	for key := range snapshot {
		keys = append(keys, swarm.NewAddress([]byte(key)).String())
	}

	sort.Strings(keys)
	for _, key := range keys {
		hasher.Write([]byte(key))
	}

	checksum.Snapshot = fmt.Sprintf("%x", hasher.Sum(nil))

	return checksum, nil
}

func (s *Service) snapshotGetHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	for key := range snapshot {
		tmp := swarm.NewAddress([]byte(key))
		_, err := io.WriteString(w, tmp.String()+"\n")
		if err != nil {
			s.logger.Error(err, "snapshot: failed to get address %v")
		}
	}
}

func (s *Service) snapshotGetChecksumHandler(w http.ResponseWriter, r *http.Request) {
	checksum, err := s.getSnapshotChecksum(r.Context(), snapshot)
	if err != nil {
		jsonhttp.InternalServerError(w, err)
		return
	}
	jsonhttp.OK(w, *checksum)
}

func (s *Service) snapshotAddHandler(w http.ResponseWriter, r *http.Request) {
	if snapshot == nil {
		snapshot = make(map[string]struct{})
	}

	rd := bufio.NewReader(r.Body)

	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			s.logger.Error(err, "snapshot add: could not read stream")
			jsonhttp.InternalServerError(w, err)
			return
		}
		stringAddr := strings.TrimSpace(line)
		hexAddr, err := swarm.ParseHexAddress(stringAddr)

		if err != nil {
			s.logger.Error(err, "snapshot add: invalid chunk addr %v", stringAddr)
			jsonhttp.InternalServerError(w, err)
			return
		}

		snapshot[hexAddr.ByteString()] = struct{}{}
	}
	jsonhttp.OK(w, len(snapshot))
}

func (s *Service) snapshotClearHandler(w http.ResponseWriter, r *http.Request) {
	snapshot = make(map[string]struct{})
	w.WriteHeader(http.StatusOK)
}

func (s *Service) snapshotApplyHandler(w http.ResponseWriter, r *http.Request) {
	// Racin: Remove this guard to simplify experiments
	// if len(snapshot) == 0 {
	// 	jsonhttp.BadRequest(w, "Snapshot is empty. Cannot apply it.")
	// 	return
	// }

	err := s.localStore.IterStoredChunks(r.Context(), false, func(item shed.Item) error {
		a := swarm.NewAddress(item.Address)
		if _, ok := snapshot[a.ByteString()]; !ok {
			// This chunk is not in the snapshot. Delete it
			err := s.storer.Set(r.Context(), storage.ModeSetRemove, a)
			if err != nil {
				w.Write([]byte("Error on chunk: " + a.String() + "\n"))
				s.logger.Debug("snapshot apply: chunk: %v, set: %v", a, err)
				jsonhttp.InternalServerError(w, err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		s.logger.Error(err, "snapshot apply: getstoredchunks %v")
		jsonhttp.InternalServerError(w, nil)
	}

	w.WriteHeader(http.StatusOK)
}
