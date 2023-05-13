package snapshot

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/cli"
	"github.com/racin/phd/code/evaluation/util"
)

const (
	addressLen         = 64 + 1 // 32 byte addresses hex encoded -> 64 bytes + newline
	addressesPerUpload = 1000
	uploadSize         = addressLen * addressesPerUpload
)

var buffers sync.Pool

func Create(ctx context.Context, ctl bee.Controller, snapshotsPath string) error {
	err := os.MkdirAll(snapshotsPath, 0755)
	if err != nil {
		return err
	}

	pw, _ := cli.NewProgress(os.Stdout, "Creating snapshots ", ctl.Len())
	defer pw.Done()

	return ctl.Parallel(ctx, func(id int, host string) (err error) {
		f, err := os.OpenFile(filepath.Join(snapshotsPath, fmt.Sprintf("%d", id)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.FilePerm)
		if err != nil {
			return fmt.Errorf("failed to open snapshots file for bee %d: %w", id, err)
		}

		defer func() {
			util.SafeClose(&err, f)
			_ = pw.Increment()
		}()

		req := &http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/dbcontents"},
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error creating snapshot for bee %d: %w", id, err)
		}
		defer util.SafeClose(&err, res.Body)
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected response from server: %s", res.Status)
		}

		n, err := io.Copy(f, res.Body)
		if err != nil {
			return fmt.Errorf("failed to write snapshot for bee %d to file: %w", id, err)
		}

		if n%addressLen != 0 {
			return fmt.Errorf("error reading snapshot for bee %d: unexpected response from server: %s", id, res.Status)
		}

		return nil
	})
}

// Upload uploads snapshots to bees.
// snapshotsPath should be the folder where snapshots are located.
// Snapshots are a list of chunk addresses that the bee should store.
// Snapshots should be named according to their IDs.
// For bee '0', the snapshot file should be named '0'.
func Upload(ctx context.Context, ctl bee.Controller, snapshotsPath string) error {
	pw, _ := cli.NewProgress(os.Stdout, "Uploading snapshots ", ctl.Len())
	defer pw.Done()
	return ctl.Parallel(ctx, func(id int, host string) (err error) {
		f, err := os.Open(filepath.Join(snapshotsPath, fmt.Sprintf("%d", id)))
		if err != nil {
			return fmt.Errorf("failed to open snapshots file for bee %d: %w", id, err)
		}

		defer func() {
			util.SafeClose(&err, f)
			_ = pw.Increment()
		}()

		var buffer *bytes.Buffer
		done := false

		p := buffers.Get()
		if p == nil {
			buffer = new(bytes.Buffer)
		} else {
			buffer = p.(*bytes.Buffer)
		}
		defer func() {
			buffer.Reset()
			buffers.Put(buffer)
		}()

		for !done {
			n, err := io.CopyN(buffer, f, uploadSize)
			if errors.Is(err, io.EOF) {
				done = true
			} else if err != nil {
				return fmt.Errorf("unexpected error reading snapshot for bee %d: %w", id, err)
			}

			if n%addressLen != 0 {
				return fmt.Errorf("error reading snapshot for bee %d: expected to read a multiple of %d bytes, but read %d", id, addressLen, n)
			}

			req := &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/snapshot/add"},
				Body:   io.NopCloser(buffer),
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("error sending snapshot for bee %d: %w", id, err)
			}
			defer util.SafeClose(&err, res.Body)
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("error sending snapshot for bee %d: unexpected response from server: %s", id, res.Status)
			}

			buffer.Reset()
		}

		return nil
	})
}

// UploadAndApplyWithChunkLoss and applies snapshots to bees with random chunk loss.
// snapshotsPath should be the folder where snapshots are located.
// chunksFile should be a file containing the chunk addresses that could be lost.
// lossPercent determines the percentage of chunks in the chunksFile that should be lost.
// Snapshots are a list of chunk addresses that the bee should store.
// Snapshots should be named according to their IDs.
// For bee '0', the snapshot file should be named '0'.
func UploadAndApplyWithChunkLoss(ctx context.Context, ctl bee.Controller, snapshotsPath string, lostChunks map[string]struct{}, whitelist ...int) (err error) {
	err = Clear(ctx, ctl)
	if err != nil {
		return err
	}
	err = UploadWithChunkLoss(ctx, ctl, snapshotsPath, lostChunks, whitelist...)
	if err != nil {
		return err
	}
	err = Apply(ctx, ctl)
	if err != nil {
		return err
	}
	return nil
}

// UploadWithChunkLoss uploads snapshots to bees with random chunk loss.
// snapshotsPath should be the folder where snapshots are located.
// chunksFile should be a file containing the chunk addresses that could be lost.
// lossPercent determines the percentage of chunks in the chunksFile that should be lost.
// Snapshots are a list of chunk addresses that the bee should store.
// Snapshots should be named according to their IDs.
// For bee '0', the snapshot file should be named '0'.
func UploadWithChunkLoss(ctx context.Context, ctl bee.Controller, snapshotsPath string, lostChunks map[string]struct{}, whitelist ...int) error {
	whitelistMap := whitelistToMap(whitelist)

	log.Printf("%d lost chunks", len(lostChunks))

	pw, _ := cli.NewProgress(os.Stdout, "Uploading snapshots ", ctl.Len())
	defer pw.Done()

	return ctl.Parallel(ctx, func(id int, host string) (err error) {
		_, whitelisted := whitelistMap[id]
		r, err := readBuffered(filepath.Join(snapshotsPath, fmt.Sprintf("%d", id)))
		if err != nil {
			return fmt.Errorf("failed to open snapshots file for bee %d: %w", id, err)
		}

		defer func() {
			util.SafeClose(&err, r)
			_ = pw.Increment()
		}()

		var (
			buffer *bytes.Buffer
			count  int
		)

		p := buffers.Get()
		if p == nil {
			buffer = new(bytes.Buffer)
		} else {
			buffer = p.(*bytes.Buffer)
		}
		defer func() {
			buffer.Reset()
			buffers.Put(buffer)
		}()

		upload := func() (err error) {
			req := &http.Request{
				Method: http.MethodPost,
				URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/snapshot/add"},
				Body:   io.NopCloser(buffer),
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("error sending snapshot for bee %d: %w", id, err)
			}
			defer util.SafeClose(&err, res.Body)
			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("error sending snapshot for bee %d: unexpected response from server: %s", id, res.Status)
			}

			buffer.Reset()

			return nil
		}

		for {
			line, err := r.ReadString('\n')
			if errors.Is(err, io.EOF) {
				if len(line) != 0 {
					return fmt.Errorf("snapshot does not end with newline")
				}
				break
			} else if err != nil {
				return fmt.Errorf("failed to read snapshot for bee %d: %w", id, err)
			}

			if _, ok := lostChunks[strings.TrimSpace(line)]; ok && !whitelisted {
				continue
			}

			_, err = buffer.WriteString(line)
			if err != nil {
				return fmt.Errorf("failed to copy address to upload buffer: %w", err)
			}
			count++

			if count == addressesPerUpload {
				err = upload()
				if err != nil {
					return err
				}
			}
		}

		if buffer.Len() > 0 {
			return upload()
		}

		return nil
	})
}

func Clear(ctx context.Context, ctl bee.Controller) error {
	pw, _ := cli.NewProgress(os.Stdout, "Clearing snapshots ", ctl.Len())
	defer pw.Done()

	return ctl.Parallel(ctx, func(id int, host string) (err error) {
		res, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodDelete,
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/snapshot"},
		})
		if err != nil {
			return fmt.Errorf("error clearing snapshot for bee %d: %w", id, err)
		}
		defer util.SafeClose(&err, res.Body)
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("error clearing snapshot for bee %d: unexpected response from server: %s", id, res.Status)
		}
		_ = pw.Increment()
		return nil
	})
}

func Apply(ctx context.Context, ctl bee.Controller) error {
	pw, _ := cli.NewProgress(os.Stdout, "Applying snapshots ", ctl.Len())
	defer pw.Done()

	return ctl.Parallel(ctx, func(id int, host string) (err error) {
		res, err := http.Post(fmt.Sprintf("http://%s/debug/snapshot/apply", host), "", nil)
		if err != nil {
			return fmt.Errorf("error applying snapshot for bee %d: %w", id, err)
		}
		defer util.SafeClose(&err, res.Body)
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("error applying snapshot for bee %d: unexpected response from server: %s", id, res.Status)
		}
		_ = pw.Increment()
		return nil
	})
}

type snapshotChecksum struct {
	Localstorage   string `json:"Localstorage"`
	Snapshot       string `json:"Snapshot"`
	SnapshotLength int    `json:"SnapshotLength"`
}

type SnapshotStatus uint8

const (
	SnapshotUnknown SnapshotStatus = iota
	SnapshotInvalid
	SnapshotValid
	SnapshotApplied
)

func Check(ctx context.Context, ctl bee.Controller, snapshotsPath string) (statuses []SnapshotStatus, err error) {
	pw, _ := cli.NewProgress(os.Stdout, "Checking snapshots ", ctl.Len())
	defer pw.Done()

	var mut sync.Mutex
	statuses = make([]SnapshotStatus, ctl.BaseLen())

	err = ctl.Parallel(ctx, func(id int, host string) (err error) {
		defer func() {
			_ = pw.Increment()
		}()

		hash, err := getSnapshotHash(filepath.Join(snapshotsPath, fmt.Sprintf("%d", id)))
		if err != nil {
			return fmt.Errorf("failed to compute snapshot hash for bee %d: %w", id, err)
		}

		res, err := http.Get(fmt.Sprintf("http://%s/debug/snapshot/checksum", host))
		if err != nil {
			return fmt.Errorf("failed to get snapshot hash from bee %d: %w", id, err)
		}
		defer util.SafeClose(&err, res.Body)
		if err := util.CheckResponse(res); err != nil {
			return err
		}

		var checksums snapshotChecksum
		decoder := json.NewDecoder(res.Body)
		err = decoder.Decode(&checksums)
		if err != nil {
			return fmt.Errorf("failed to decode response from bee %d: %w", id, err)
		}

		status := SnapshotInvalid
		if checksums.Snapshot == hash {
			status = SnapshotValid
			if checksums.Localstorage == checksums.Snapshot {
				status = SnapshotApplied
			}
		}

		mut.Lock()
		statuses[id] = status
		mut.Unlock()

		return nil
	})

	if err != nil {
		return nil, err
	}

	return statuses, nil
}

func getSnapshotHash(snapshotPath string) (hash string, err error) {
	f, err := os.Open(snapshotPath)
	if err != nil {
		return "", err
	}
	defer util.SafeClose(&err, f)

	rd := bufio.NewReader(f)
	var addresses []string

	for {
		line, err := rd.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return "", err
		}

		addresses = append(addresses, strings.TrimSpace(line))
	}

	sort.Strings(addresses)

	hasher := sha256.New()

	for _, address := range addresses {
		_, err := io.WriteString(hasher, address)
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func PrintStatuses(statuses []SnapshotStatus) {
	statusesPerLine := 50

	for i, status := range statuses {
		if i%statusesPerLine == 0 {
			fmt.Println()
		}
		switch status {
		case SnapshotInvalid:
			fmt.Print("💔")
		case SnapshotValid:
			fmt.Print("💙")
		case SnapshotApplied:
			fmt.Print("💚")
		case SnapshotUnknown:
			fmt.Print("❓")
		default:
			fmt.Print("❗")
		}
	}
	fmt.Println()
}

// UploadAndApply uploads snapshots and then applies them to the bees.
func UploadAndApply(ctx context.Context, ctl bee.Controller, snapshotsPath string) error {
	statuses, err := Check(ctx, ctl, snapshotsPath)
	if err != nil {
		return err
	}

	var (
		needUpload []int
		needApply  []int
		applied    int
	)

	for id, status := range statuses {
		switch status {
		case SnapshotInvalid:
			needUpload = append(needUpload, id)
			fallthrough
		case SnapshotValid:
			needApply = append(needApply, id)
		case SnapshotApplied:
			applied++
		case SnapshotUnknown:
			// do nothing, assume the node wasn't queried
		default:
			return fmt.Errorf("unexpected snapshot status %d for bee %d", status, id)
		}
	}

	log.Printf("snapshot status: %d ready, %d to upload, %d to apply", applied, len(needUpload), len(needApply))

	if len(needUpload) > 0 {
		uploadCtl := ctl.Sub(needUpload)

		err = Clear(ctx, uploadCtl)
		if err != nil {
			return fmt.Errorf("failed to clear snapshots: %w", err)
		}

		err = Upload(ctx, uploadCtl, snapshotsPath)
		if err != nil {
			return fmt.Errorf("failed to upload snapshots: %w", err)
		}
	}

	if len(needApply) > 0 {
		applyCtl := ctl.Sub(needApply)

		err = Apply(ctx, applyCtl)
		if err != nil {
			return fmt.Errorf("failed to apply snapshots: %w", err)
		}

		// check that all are applied now
		statuses, err = Check(ctx, ctl, snapshotsPath)
		if err != nil {
			return err
		}

		for id, status := range statuses {
			switch status {
			case SnapshotApplied:
				// do nothing
			case SnapshotUnknown:
				// do nothing, assume the bee wasn't queried
			default:
				PrintStatuses(statuses)
				return fmt.Errorf("unexpected snapshot status %d for bee %d", status, id)
			}
		}
	}

	return nil
}
