package verifypersistence

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/cli"
	"github.com/racin/phd/code/evaluation/util"
)

// Verifies the persistence in the Swarm main net.

func VerifyPersistence(ctx context.Context, chunksdir, host string, verifyInterval, requestInterval time.Duration) (err error) {

	files, err := ioutil.ReadDir(chunksdir)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	chunksHash := make(map[string]string)
	for _, file := range files {
		filedata, err := ioutil.ReadFile(filepath.Join(chunksdir, file.Name()))
		if err != nil {
			return err
		}
		hasher.Write(filedata)
		chunksHash[file.Name()] = fmt.Sprintf("%x", hasher.Sum(nil))
		hasher.Reset()
	}

	ticker := time.NewTicker(verifyInterval)
	defer ticker.Stop()
	requestTicker := time.NewTicker(requestInterval) // Wait 5 seconds between gateway requests.
	defer requestTicker.Stop()
	for {
		log.Printf("Verifying chunks every %v, request interval %v\n", verifyInterval, requestInterval)

		pw, _ := cli.NewProgress(os.Stdout, "Verifying chunks ", len(chunksHash))
		correctChunks := 0
		for key, hash := range chunksHash {
			if err := ctx.Err(); err != nil {
				return err
			}

			_ = pw.Increment()
			// Download chunk
			<-requestTicker.C // Wait between requests
			resp, err := http.DefaultClient.Do(&http.Request{
				Method: "GET",
				URL:    &url.URL{Scheme: "http", Host: host, Path: "/bytes/" + key},
			})
			if err != nil {
				log.Printf(" - Request error for chunk %v, Err: %s\n", key, err)
				continue
			}
			err = util.CheckResponse(resp)
			if err != nil {
				log.Printf(" - Response error for chunk %v, Err: %s\n", key, err)
				util.SafeClose(&err, resp.Body)
				continue
			}

			data, err := ioutil.ReadAll(resp.Body)
			util.SafeClose(&err, resp.Body)
			if err != nil {
				log.Printf(" - Cannot read response data\n")
				continue
			}
			hasher.Reset()

			hasher.Write(data)
			retrievedHash := fmt.Sprintf("%x", hasher.Sum(nil))
			if hash != retrievedHash {
				log.Printf(" - Retrieved hash does not match for chunk %v, Want: %v, Got: %v\n", key, hash, retrievedHash)
			}
			//fmt.Printf("%v\n", data)
			correctChunks++
		}
		pw.Done()
		log.Printf("Received correct response for %v chunks. Total: %v", correctChunks, len(chunksHash))

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			log.Println("Verify. Context cancelled")
			return
		}
	}
	return nil
}

func DownloadAllChunksOfFile(ctx context.Context, conn *bee.Connector, rootaddr, outputDir string, requestID int) (count int, err error) {
	// Connect debug
	debughost, disconnect, err := conn.ConnectDebug(ctx, requestID)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to Debug bee %d: %w", requestID, err)
	}
	defer util.CheckErr(&err, disconnect)

	// Traverse the file
	res, err := http.DefaultClient.Do(&http.Request{
		Method: "GET",
		URL:    &url.URL{Scheme: "http", Host: debughost, Path: "/debug/traverse/" + rootaddr},
	})
	if err != nil {
		return 0, fmt.Errorf("traverse, error sending request to %s, addr: %s: %w", debughost, rootaddr, err)
	}
	defer util.SafeClose(&err, res.Body)
	err = util.CheckResponse(res)
	if err != nil {
		return 0, fmt.Errorf("traverse response, error sending request to %s, addr: %s: %w", debughost, rootaddr, err)
	}
	rd := bufio.NewReader(res.Body)
	pw, _ := cli.NewProgress(os.Stdout, "Downloading root chunk ", 1)
	defer pw.Done()

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			return 0, err
		}
	}

	// Connect API
	host, disconnect, err := conn.ConnectAPI(ctx, requestID)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to API bee %d: %w", requestID, err)
	}
	defer util.CheckErr(&err, disconnect)

	var filesize int
	var chunks int = 1

	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("error reading traverse response. line: %v. %w", line, err)
		}
		stringAddr := strings.TrimSpace(line)

		// Download chunk
		resp, err := http.DefaultClient.Do(&http.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/bytes/" + stringAddr},
		})
		if err != nil {
			return 0, fmt.Errorf("bytes, error sending request to %s, addr: %s: %w", host, rootaddr, err)
		}
		defer util.SafeClose(&err, resp.Body)
		err = util.CheckResponse(resp)
		if err != nil {
			return 0, fmt.Errorf("bytes response, error sending request to %s, addr: %s: %w", debughost, rootaddr, err)
		}

		size, err := strconv.Atoi(resp.Header.Get("Decompressed-Content-Length"))
		if stringAddr == rootaddr && size < 4096 {
			pw.SetMessage("Downloading metadata ")
			chunks += 4
			pw.SetTotal(chunks)
		} else if size > 4096 {
			chunks++
			if size > filesize {
				chunks += (size / 4096)
				filesize = size
				pw.SetMessage("Downloading chunks   ")
			}
			pw.SetTotal(chunks)
		}

		_ = pw.Increment()
		count++

		// For each chunk, download and store it to the output directory
		out, err := os.Create(outputDir + "/" + stringAddr)
		if err != nil {
			return 0, err
		}
		defer out.Close()
		io.Copy(out, resp.Body)
		if err != nil {
			return 0, err
		}
	}

	return
}
