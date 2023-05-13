package snapshot

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/cli"
	"github.com/racin/phd/code/evaluation/util"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

// ChunkLoss simulates loss of chunks by removing a percentage of chunks from read snapshots stored at snapshotsPath.
// The resulting "lossy" snapshots are created in outputPath. The snapshots of bee IDs in whiteList are skipped.
// The list of all chunks to choose from are read from chunkListPath.
func ChunkLoss(ctx context.Context, lossPercent float64, whitelist []int, chunkListPath, snapshotsPath, outputPath string, parallel int) (err error) {
	var removeChunks map[string]struct{}

	whitelistMap := whitelistToMap(whitelist)

	f, err := readBuffered(chunkListPath)
	if err != nil {
		return fmt.Errorf("failed to open chunk list file: %w", err)
	}
	defer util.SafeClose(&err, f)

	removeChunks, err = ReadPercent(ctx, f.Reader, lossPercent)
	if err != nil {
		return err
	}
	log.Printf("%d chunks to remove...", len(removeChunks))

	err = os.MkdirAll(outputPath, util.DirPerm)
	if err != nil {
		return fmt.Errorf("failed to make output directory: %w", err)
	}

	files, err := os.ReadDir(snapshotsPath)
	if err != nil {
		return fmt.Errorf("failed to list snapshot files: %w", err)
	}

	pw, _ := cli.NewProgress(os.Stdout, "Copying snapshots ", len(files))
	defer pw.Done()

	sem := make(chan struct{}, parallel)
	eg, ctx := errgroup.WithContext(ctx)

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		if file.IsDir() {
			pw.Increment()
			continue
		}

		readPath := filepath.Join(snapshotsPath, file.Name())
		writePath := filepath.Join(outputPath, file.Name())

		id, err := strconv.Atoi(file.Name())
		if err != nil {
			return fmt.Errorf("name of snapshot file '%s' is not a numerical bee ID: %w", readPath, err)
		}

		_, whitelisted := whitelistMap[id]

		sem <- struct{}{}

		eg.Go(func() error {
			defer func() { <-sem }()
			if whitelisted {
				err = lossyCopy(ctx, nil, readPath, writePath)
			} else {
				err = lossyCopy(ctx, removeChunks, readPath, writePath)
			}
			return err
		})

		pw.Increment()
	}

	return eg.Wait()
}

func lossyCopy(ctx context.Context, removeChunks map[string]struct{}, readPath, writePath string) (err error) {
	r, err := readBuffered(readPath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s' for reading: %w", readPath, err)
	}
	defer util.SafeClose(&err, r)

	w, err := writeBuffered(writePath)
	if err != nil {
		return fmt.Errorf("failed to open file '%s' for writing: %w", writePath, err)
	}
	defer util.SafeClose(&err, w)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		line, err := r.ReadString('\n')
		if errors.Is(err, io.EOF) {
			if len(line) != 0 {
				return fmt.Errorf("snapshot file does not end with newline")
			}
			break
		} else if err != nil {
			return fmt.Errorf("failed to read snapshot file: %w", err)
		}

		if _, ok := removeChunks[strings.TrimSpace(line)]; !ok {
			_, err = w.WriteString(line)
			if err != nil {
				return fmt.Errorf("failed to write to snapshot file: %w", err)
			}
		}
	}

	return nil
}

func GetChunkLossFromFile(ctx context.Context, chunkIDsPath string, chunkloss float64) (chunks map[string]struct{}, err error) {
	f, err := readBuffered(chunkIDsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open chunk list file: %w", err)
	}
	defer util.SafeClose(&err, f)

	chunks, err = ReadPercent(ctx, f.Reader, chunkloss)
	if err != nil {
		return nil, err
	}

	return chunks, nil
}

func GetChunkLossTraverse(ctx context.Context, conn *bee.Connector, beeID int, fileAddr string, chunkloss float64) (chunks map[string]struct{}, err error) {
	host, disconnect, err := conn.ConnectDebug(ctx, beeID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to uploader: %w", err)
	}
	defer util.CheckErr(&err, disconnect)

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/traverse/" + fileAddr},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to traverse addr %v: %w", fileAddr, err)
	}
	defer util.SafeClose(&err, res.Body)
	err = util.CheckResponse(res)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse addr %v: %w", fileAddr, err)
	}

	rd := bufio.NewReader(res.Body)

	chunks, err = ReadPercent(ctx, rd, chunkloss)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk IDs: %w", err)
	}

	return chunks, nil
}

// ReadPercent reads a percentage of the chunks from the chunkList reader.
func ReadPercent(ctx context.Context, chunkList *bufio.Reader, percent float64) (chunkIDs map[string]struct{}, err error) {
	var allChunkIDs []string

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		line, err := chunkList.ReadString('\n')
		if errors.Is(err, io.EOF) {
			if len(line) != 0 {
				return nil, fmt.Errorf("chunk list does not end with newline")
			}
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read chunk list: %w", err)
		}

		allChunkIDs = append(allChunkIDs, strings.TrimSpace(line))
	}

	util.Rand.Shuffle(len(allChunkIDs), reflect.Swapper(allChunkIDs))

	chunkIDs = make(map[string]struct{})

	toChoose := int(float64(len(allChunkIDs)) * percent / 100)

	for i := 0; i < toChoose; i++ {
		chunkIDs[allChunkIDs[i]] = struct{}{}
	}

	return chunkIDs, nil
}

// ReadApproxPercent reads a percentage (probabalistic approximate) of the chunks from the chunkList reader.
func ReadApproxPercent(ctx context.Context, chunkList *bufio.Reader, percent float64) (chunkIDs map[string]struct{}, err error) {
	chunkIDs = make(map[string]struct{})

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		line, err := chunkList.ReadString('\n')
		if errors.Is(err, io.EOF) {
			if len(line) != 0 {
				return nil, fmt.Errorf("chunk list does not end with newline")
			}
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read chunk list: %w", err)
		}

		if util.Rand.Float64()*100 < percent {
			chunkIDs[strings.TrimSpace(line)] = struct{}{}
		}
	}

	return chunkIDs, nil
}

func GetChunkListFromFile(ctx context.Context, chunkIDsPath string) (map[string]struct{}, error) {
	chunkMap := make(map[string]struct{})
	chunkList, err := readBuffered(chunkIDsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open chunk list file: %w", err)
	}
	defer util.SafeClose(&err, chunkList)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		line, err := chunkList.ReadString('\n')
		if errors.Is(err, io.EOF) {
			if len(line) != 0 {
				return nil, fmt.Errorf("chunk list does not end with newline")
			}
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to read chunk list: %w", err)
		}

		chunkid := strings.TrimSpace(line)
		chunkMap[chunkid] = struct{}{}
	}
	return chunkMap, err

}
func HasAllChunks(ctx context.Context, ctl bee.Controller, beeID int, chunkMap map[string]struct{}) (bool, error) {
	host, disconnect, err := ctl.Connector().ConnectDebug(ctx, beeID)
	if err != nil {
		return false, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)
	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/dbcontents"},
	})
	if err != nil {
		return false, fmt.Errorf("failed to request database contents from %d: %w", beeID, err)
	}
	defer util.SafeClose(&err, res.Body)
	err = util.CheckResponse(res)
	if err != nil {
		return false, err
	}

	rd := bufio.NewReader(res.Body)
	foundChunks := make(map[string]struct{})
	for {
		line, rerr := rd.ReadString('\n')
		if errors.Is(rerr, io.EOF) {
			break
		} else if rerr != nil {
			err = multierr.Append(rerr, fmt.Errorf("failed to read from %d's response body: %w", beeID, err))
			return false, err
		}

		chunkid := strings.TrimSpace(line)
		foundChunks[chunkid] = struct{}{}
	}

	for chunkid := range chunkMap {
		if _, ok := foundChunks[chunkid]; !ok {
			fmt.Printf("Found %v chunk. Expected %v chunks.\n", len(foundChunks), len(chunkMap))
			return false, nil
		}
	}

	return true, nil
}

func readBuffered(path string) (bufReaderCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return bufReaderCloser{}, err
	}
	return bufReaderCloser{
		Reader: bufio.NewReader(f),
		Closer: f,
	}, nil
}

func writeBuffered(path string) (bufWriterCloser, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.FilePerm)
	if err != nil {
		return bufWriterCloser{}, err
	}
	return bufWriterCloser{
		Writer: bufio.NewWriter(f),
		closer: f,
	}, nil
}

type bufWriterCloser struct {
	*bufio.Writer
	closer io.Closer
}

func (bw bufWriterCloser) Close() error {
	return multierr.Combine(
		bw.Flush(),
		bw.closer.Close(),
	)
}

type bufReaderCloser struct {
	*bufio.Reader
	io.Closer
}

func whitelistToMap(whitelist []int) map[int]struct{} {
	whitelistMap := make(map[int]struct{})
	for _, id := range whitelist {
		whitelistMap[id] = struct{}{}
	}
	return whitelistMap
}
