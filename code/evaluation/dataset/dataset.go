package dataset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethersphere/bee/pkg/cac"
	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/cli"
	"github.com/racin/phd/code/evaluation/util"
)

type DataSizeConfig struct {
	SizeKB int `json:"size_kb"`
	Count  int `json:"count"`
}

type DatasetConfig struct {
	Sizes      []DataSizeConfig `json:"sizes"`
	UploaderID int              `json:"uploader_id"`
	Prefix     string           `json:"prefix"`
	StampID    string           `json:"stamp_id"`
	OutputPath string           `json:"output_path"`
}

type Dataset struct {
	RootAddressesPerSizeKB map[int][]string `json:"root_addresses_per_size_kb"`
}

func (ds Dataset) Iterate(iterFn func(addr string, sizeKB int) error) error {
	for sizeKB, addrs := range ds.RootAddressesPerSizeKB {
		for _, addr := range addrs {
			err := iterFn(addr, sizeKB)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CreateDataset(config DatasetConfig, outputPath string) (files map[int][]string, err error) {
	err = os.MkdirAll(outputPath, util.DirPerm)
	if err != nil {
		return nil, err
	}

	files = make(map[int][]string)

	for _, size := range config.Sizes {
		files[size.SizeKB] = make([]string, size.Count)
		for i := 0; i < size.Count; i++ {
			filePath := filepath.Join(outputPath, fmt.Sprintf("%d-kb-%s", size.SizeKB, util.RandomString(8)))
			f, err := os.OpenFile(
				filepath.Join(filePath),
				os.O_WRONLY|os.O_CREATE|os.O_EXCL,
				util.FilePerm,
			)
			if err != nil {
				return nil, err
			}

			defer util.SafeClose(&err, f)

			_, err = io.CopyN(f, util.Rand, int64(size.SizeKB*1024))
			if err != nil {
				return nil, fmt.Errorf("failed to write random data to %s: %w", f.Name(), err)
			}

			files[size.SizeKB][i] = filePath
		}
	}

	return files, nil
}

func generateChunksWithPrefix(numChunks int, prefix string) []swarm.Chunk {
	chunks := make([]swarm.Chunk, numChunks)
	for i := 0; i < numChunks; i++ {
		data := make([]byte, swarm.ChunkSize)
		_, _ = rand.Read(data)
		ch, _ := cac.New(data)
		addr := ch.Address().String()
		if !strings.HasPrefix(addr, prefix) {
			i--
			continue
		}
		chunks[i] = ch
	}
	return chunks
}

func CreateDatasetWithPrefix(config DatasetConfig, outputPath string) (files map[int][]string, err error) {
	err = os.MkdirAll(outputPath, util.DirPerm)
	if err != nil {
		return nil, err
	}

	files = make(map[int][]string)

	for _, size := range config.Sizes {
		files[size.SizeKB] = make([]string, size.Count)
		for i := 0; i < size.Count; i++ {
			chunks := generateChunksWithPrefix(size.SizeKB/4, config.Prefix)
			filePath := filepath.Join(outputPath, fmt.Sprintf("%d-kb-%s-%s", size.SizeKB, config.Prefix, util.RandomString(8)))
			f, err := os.OpenFile(
				filepath.Join(filePath),
				os.O_WRONLY|os.O_CREATE|os.O_EXCL,
				util.FilePerm,
			)
			if err != nil {
				return nil, err
			}

			defer util.SafeClose(&err, f)
			for _, chunk := range chunks {
				_, err = f.Write(chunk.Data()[swarm.SpanSize:])
				if err != nil {
					return nil, fmt.Errorf("failed to write random data to %s: %w", f.Name(), err)
				}
			}

			files[size.SizeKB][i] = filePath
		}
	}

	return files, nil
}

func UploadDataset(ctx context.Context, conn *bee.Connector, config DatasetConfig) (dataSet Dataset, err error) {
	dataSet = Dataset{
		RootAddressesPerSizeKB: make(map[int][]string),
	}

	uploader, disconnect, err := conn.ConnectAPI(ctx, config.UploaderID)
	if err != nil {
		return Dataset{}, fmt.Errorf("failed to connect to bee %d: %w", config.UploaderID, err)
	}
	defer util.CheckErr(&err, disconnect)
	var outputpath string
	if config.OutputPath == "" {
		outputpath, err := os.MkdirTemp(os.TempDir(), "snarl-eval-*")
		if err != nil {
			return Dataset{}, fmt.Errorf("error creating temporary directory: %w", err)
		}
		defer util.CheckErr(&err, func() error { return os.RemoveAll(outputpath) })
	} else {
		outputpath = config.OutputPath
	}
	var files map[int][]string
	if config.Prefix != "" {
		files, err = CreateDatasetWithPrefix(config, outputpath)
	} else {
		files, err = CreateDataset(config, outputpath)
	}

	if err != nil {
		return Dataset{}, fmt.Errorf("failed to create dataset: %w", err)
	}

	for size, filesOfSize := range files {
		pw, _ := cli.NewProgress(os.Stdout, fmt.Sprintf("Uploading %dkb files ", size), len(filesOfSize))

		dataSet.RootAddressesPerSizeKB[size] = make([]string, len(filesOfSize))
		for i, file := range filesOfSize {
			rootAddr, err := UploadFile(uploader, file, config.StampID)
			if err != nil {
				return Dataset{}, fmt.Errorf("failed to upload file of size %dKB: %w", size, err)
			}
			dataSet.RootAddressesPerSizeKB[size][i] = rootAddr
			_ = pw.Increment()
		}

		pw.Done()

		log.Printf("Root addresses: ")
		for _, addr := range dataSet.RootAddressesPerSizeKB[size] {
			log.Printf("\t%s", addr)
		}
	}

	return dataSet, nil
}

type uploadResponse struct {
	Reference string `json:"reference"`
}

func UploadFile(host string, path string, stampID string) (rootAddr string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	fi, err := f.Stat()
	if err != nil {
		util.SafeClose(&err, f)
		return "", err
	}

	header := make(http.Header)
	header.Set("Content-Type", "application/octet-stream")
	header.Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	header.Set("Swarm-Postage-Batch-Id", stampID)

	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: host, Path: "/bzz" /* RawQuery: fmt.Sprintf("name=%s", url.QueryEscape(filepath.Base(path))) */},
		Header: header,
		Body:   f,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer util.SafeClose(&err, res.Body)
	err = util.CheckResponse(res)
	if err != nil {
		return "", fmt.Errorf("got unexpected response: %w", err)
	}

	var responseData uploadResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&responseData)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return responseData.Reference, nil
}
