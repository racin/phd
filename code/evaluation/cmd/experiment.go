package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/dataset"
	"github.com/racin/phd/code/evaluation/metrics"
	"github.com/racin/phd/code/evaluation/neighbor"
	"github.com/racin/phd/code/evaluation/prometheus"
	"github.com/racin/phd/code/evaluation/protocols"
	"github.com/racin/phd/code/evaluation/pullsync"
	"github.com/racin/phd/code/evaluation/reupload"
	"github.com/racin/phd/code/evaluation/snapshot"
	"github.com/racin/phd/code/evaluation/snips"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

const (
	pushsyncDeliverySize = 4256
	pushsyncReceiptSize  = 135
	posChallengeSize     = 68
	posResponseSize      = 205
)

// Pullsync sizes
const (
	pullsyncSizeCursor    = 8  // captures the number of cursors sent. Each cursor is uint64 (8 byte)
	pullsyncSizeRuid      = 4  // Message contains a single uint32 (4 byte)
	pullsyncSizeRange     = 20 // Message contains, 1x int32 and 2x uint64 (20 byte)
	pullsyncSizeSent      = 8  // Each offer contains 1x uint64 plus and array of chunk identifiers. Count only the 1x uint64 (8 byte),
	pullsyncChunksInOffer = 1  // Captures the number of bytes to cover all chunk identifiers. The metrics is always diviasble by 32. (1 byte)
	pullsyncSizeWant      = 1  // Records the actual size of the bit vector sent. Counted in bytes. (1 byte)
)

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

type Config struct {
	EnableProtocols  []string `json:"enable-protocols"`
	DisableProtocols []string `json:"disable-protocols"`

	FailPercent   float64  `json:"fail-percent"`
	FailWhitelist []int    `json:"fail-whitelist"`
	FailProtocols []string `json:"fail-protocols"`
	ChunkLossFile string   `json:"chunkloss-file"`
	FailType      string   `json:"fail-type"`

	SnapshotsPath string `json:"snapshots-path"`

	Metrics []string `json:"metrics"`
}

var (
	experimentRepetitions int
	experimentRandSeed    int64
	experimentChunkloss   []float64

	experimentWritePOSResult bool = true
	experimentWriteMetrics   bool = true
	experimentRetryInterval  time.Duration
	experimentRetries        int
)

var experimentCmd = &cobra.Command{
	Use:   "experiment name outputPath snapshotsPath dataset.json",
	Short: "run experiments",
	Long:  `The experiment command runs experiments.`,
	Args:  cobra.RangeArgs(4, 5),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() { experimentChunkloss = nil }() // reset chunkloss flag

		if experimentChunkloss == nil {
			return fmt.Errorf("--chunkloss flag required")
		}

		var dataSet dataset.Dataset
		err = util.ReadJSONFile(args[3], &dataSet)
		if err != nil {
			return fmt.Errorf("failed to read dataset file: %w", err)
		}

		// Setup or get prometheus connection.
		promconn, err := getPromConnector()
		if err != nil {
			return err
		}
		host, disconnect, err := promconn.ConnectAPI(context.Background(), 0)
		defer util.CheckErr(&err, disconnect)
		promapi, err := getPrometheus("http://" + host)
		if err != nil {
			return err
		}

		var allChunksPath string
		outputPath := args[1]
		snapshotsPath := args[2]
		runMultiExperies := false
		if len(args) == 5 {
			allChunksPath = args[4]
			if strings.Contains(allChunksPath, ",") && strings.Contains(outputPath, ",") && strings.Contains(snapshotsPath, ",") {
				allChunksPathArr := strings.Split(allChunksPath, ",")
				outputPathArr := strings.Split(outputPath, ",")
				snapshotsPathArr := strings.Split(snapshotsPath, ",")
				if len(allChunksPathArr) == len(outputPathArr) && len(allChunksPathArr) == len(snapshotsPathArr) {
					runMultiExperies = true
					for i := 0; i < len(allChunksPathArr); i++ {
						fmt.Printf("Running experiment %d/%d. outputpath: %v, snapshotpath: %v, allchunkspath: %v\n", i+1, len(allChunksPathArr), outputPathArr[i], snapshotsPathArr[i], allChunksPathArr[i])
						err = runExperiment(cmd.Context(), args[0], outputPathArr[i], snapshotsPathArr[i], dataSet, experimentChunkloss, experimentRepetitions, allChunksPathArr[i], promapi)
						if err != nil {
							return err
						}
					}
				}
			}
		}
		if !runMultiExperies {
			err = runExperiment(cmd.Context(), args[0], outputPath, snapshotsPath, dataSet, experimentChunkloss, experimentRepetitions, allChunksPath, promapi)

			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(experimentCmd)

	experimentCmd.Flags().Int64Var(&experimentRandSeed, "seed", 0, "random seed")
	experimentCmd.Flags().IntVar(&experimentRepetitions, "repeat", 1, "how many times to run the experiment")
	experimentCmd.Flags().Float64SliceVar(&experimentChunkloss, "chunkloss", nil, "list of chunk loss percentages to test with")
	rootCmd.PersistentFlags().IntVar(&experimentRetries, "retries", 10, "number of times to retry experiment")
	rootCmd.PersistentFlags().DurationVar(&experimentRetryInterval, "retry-interval", 10*time.Second, "time interval between experiment retries")
}

func isDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

func runExperiment(ctx context.Context, name, outputPath, snapshotsPath string, dataset dataset.Dataset, chunkloss []float64, repeat int, allChunksPath string, promapi v1.API) (err error) {
	ctl, err := getController(ctx)
	if err != nil {
		return err
	}

	allResults := make(results)
	// write the results even if the experiment errors
	defer util.CheckErr(&err, func() error {
		return util.WriteJSONFile(filepath.Join(outputPath, "results.json"), util.FilePerm, allResults)
	})

	_, revNeighborMap, err := neighbor.GetNeighborMap(ctx, ctl, uploaderID)
	if err != nil {
		return err
	}

	for r := 0; r < repeat; r++ {
		log.Println("RUN", r)

		for _, percent := range chunkloss {
			percentStr := strconv.FormatFloat(percent, 'f', 1, 64)
			log.Printf("Chunkloss %s%%", percentStr)

			needSnapshotUpload := false
			var allChunks map[string]struct{}
			var lostChunks map[string]struct{}

			err = retryFunc(ctx, ctl, func() error {
				if allChunksPath == "" {
					err = uploadSnapshotsForWholeDataset(ctx, ctl, snapshotsPath, dataset, percent)

					if errors.Is(err, ErrDuplicate) {
						needSnapshotUpload = true
						return nil
					}
				} else {
					allChunks, err = snapshot.GetChunkListFromFile(ctx, allChunksPath)
					if err != nil {
						return err
					}

					lostChunks, err = snapshot.GetChunkLossFromFile(ctx, allChunksPath, percent)
					if err != nil {
						return err
					}
					err = snapshot.UploadAndApplyWithChunkLoss(ctx, ctl, snapshotsPath, lostChunks, uploaderID)
				}
				return err
			})
			if err != nil {
				return err
			}

			err = dataset.Iterate(func(addr string, sizeKB int) error {
				log.Printf("reupload %s file %s", humanizeBytes(int64(sizeKB)*1024), addr)

				dir := filepath.Join(outputPath, percentStr, strconv.Itoa(r), addr)
				err = os.MkdirAll(dir, util.DirPerm)
				if err != nil {
					return fmt.Errorf("failed to create output dir: %w", err)
				}

				var result reuploadResult
				nonce := rand.Uint64()
				err = retryFunc(ctx, ctl, func() error {
					result, err = runSingle(ctx, ctl, name, dir, snapshotsPath, needSnapshotUpload, addr, percent, nonce, allChunks, lostChunks, promapi, revNeighborMap)
					if err != nil {
						needSnapshotUpload = true
					}
					return err
				})
				result.FileSizeKB = sizeKB
				if err != nil {
					return err
				}

				fr, ok := allResults[addr]
				if !ok {
					fr = fileResults{Runs: make(map[string]fileResult)}
					fr.FileSizeKB = sizeKB
				}

				rr := fr.Runs[percentStr]
				rr.Nonce = nonce
				rr.Duration = append(rr.Duration, int64(result.Duration))
				rr.Bandwidth = append(rr.Bandwidth, result.Bandwidth)
				rr.AdditionBandwidth = append(rr.AdditionBandwidth, result.AdditionBandwidth)

				fr.Runs[percentStr] = rr
				allResults[addr] = fr

				return nil
			})
			if err != nil {
				return err
			}

			err = util.WriteJSONFile(filepath.Join(outputPath, "results.json"), util.FilePerm, allResults)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func retryFunc(ctx context.Context, ctl bee.Controller, f func() error) (err error) {
	retries := 0

retry:
	err = f()
	if err == nil {
		return nil
	}
	log.Println("Error: ", err)

retryHealth:
	if err = ctx.Err(); err != nil {
		return err
	}

	retries++
	if retries >= experimentRetries {
		return errors.New("too many retries")
	}

	log.Printf("Retrying in %s", experimentRetryInterval)
	time.Sleep(experimentRetryInterval)

	err = ctl.CheckHealth(ctx)
	if err != nil {
		log.Println("Health check failed: ", err)
		goto retryHealth
	}
	goto retry
}

var ErrDuplicate = errors.New("duplicated chunk")

func uploadSnapshotsForWholeDataset(ctx context.Context, ctl bee.Controller, snapshotsPath string, dataset dataset.Dataset, chunkloss float64) (err error) {
	allChunks := make(map[string]struct{})
	err = dataset.Iterate(func(addr string, sizeKB int) error {
		chunks, err := snapshot.GetChunkLossTraverse(ctx, ctl.Connector(), uploaderID, addr, chunkloss)
		if err != nil {
			return err
		}
		for chunk := range chunks {
			if _, ok := allChunks[chunk]; ok {
				log.Println("duplicated chunk: ", chunk)
				return ErrDuplicate
			}
			allChunks[chunk] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = snapshot.UploadAndApplyWithChunkLoss(ctx, ctl, snapshotsPath, allChunks, uploaderID)
	if err != nil {
		return err
	}

	return nil
}

type reuploadResult struct {
	Duration          time.Duration
	Bandwidth         int64
	AdditionBandwidth int64
	FileSizeKB        int
}

func runSingle(ctx context.Context, ctl bee.Controller, name, outputPath, snapshotsPath string, uploadSnapshot bool, fileAddr string, chunkloss float64, nonce uint64, allChunks map[string]struct{}, lostChunks map[string]struct{}, promapi v1.API, revNeighborMap neighbor.ReverseNeighborMap) (_ reuploadResult, err error) {
	if uploadSnapshot {
		if name != "pullsync" && name != "snips" {
			chunks, err := snapshot.GetChunkLossTraverse(ctx, ctl.Connector(), uploaderID, fileAddr, chunkloss)
			if err != nil {
				return reuploadResult{}, err
			}
			err = snapshot.UploadAndApplyWithChunkLoss(ctx, ctl, snapshotsPath, chunks, uploaderID)
			if err != nil {
				return reuploadResult{}, err
			}
		} else {
			err = snapshot.UploadAndApplyWithChunkLoss(ctx, ctl, snapshotsPath, lostChunks, uploaderID) // Pick one of the peers in the neighborhood (allChunksPath)
			if err != nil {
				return reuploadResult{}, err
			}
		}
	}

	beforeMetrics, err := metrics.GetMetrics(ctx, ctl)
	if err != nil {
		return reuploadResult{}, err
	}

	var duration time.Duration
	switch name {
	case "pos":
		duration, err = runSinglePOS(ctx, ctl, outputPath, fileAddr)
	case "pullsync":
		var subdur1, subdur2, subdur3 time.Duration
		_ = subdur1
		myAddr, err := neighbor.GetMyAddresses(ctx, ctl, uploaderID)
		if err != nil {
			return reuploadResult{}, err
		}

		subdur1, err = pullsync.ClearPullSync(ctx, ctl, uploaderID, "all")
		if err != nil {
			return reuploadResult{}, fmt.Errorf("failed to run Clear Cursors: %w", err)
		}

		revUploaderMap := revNeighborMap[fmt.Sprintf(nameFormat, uploaderID)]
		fmt.Printf("Got revUploaderMap map: %v. My key: %v\n", revUploaderMap, fmt.Sprintf(nameFormat, uploaderID))
		allRevNeighborIDs := make([]int, len(revUploaderMap))
		for i, neighborIDFullstr := range revUploaderMap {
			neighborIDstr := strings.Split(neighborIDFullstr, "-")
			neighborID, err := strconv.Atoi(neighborIDstr[len(neighborIDstr)-1])
			if err != nil {
				return reuploadResult{}, err
			}
			allRevNeighborIDs[i] = neighborID
			subsubdur2, err := pullsync.ClearPullSync(ctx, ctl, neighborID, myAddr.Overlay)
			if err != nil {
				return reuploadResult{}, fmt.Errorf("failed to run Clear Cursors: %w", err)
			}
			subdur2 += subsubdur2
		}

		for _, neighborID := range allRevNeighborIDs {
			subsubdur3, err := pullsync.StartPullSync(ctx, ctl, neighborID)
			if err != nil {
				return reuploadResult{}, err
			}
			subdur3 += subsubdur3
		}
		dur, err := prometheus.MeasureRateFallTime(ctx, promapi, 600*time.Second, 12*time.Hour, "sum(bee_pullsync_db_ops) + sum(bee_pullsync_total_cursor_req_sent)+ sum(bee_pullsync_total_cursor_req_recv) + sum(bee_pullsync_total_ruid_sent) + sum(bee_pullsync_total_offer_sent) + sum(bee_pullsync_total_want_sent)")

		if err != nil {
			return reuploadResult{}, err
		}
		duration = subdur3 + dur // Don't count time to clear cursors

		for _, neighborID := range allRevNeighborIDs {
			hasAllChunk, err := snapshot.HasAllChunks(ctx, ctl, neighborID, allChunks)
			if err != nil {
				return reuploadResult{}, err
			}
			if !hasAllChunk {
				return reuploadResult{}, fmt.Errorf("neighbor %d does not have all chunks", neighborID)
			}
		}
	case "snips":
		var subdur time.Duration
		protocols.SetProtocol(ctx, ctl, "pullsync", false)
		subdur, err = snips.StartSNIPS(ctx, ctl, uploaderID, nonce)

		dur, err := prometheus.MeasureRateFallTime(ctx, promapi, 600*time.Second, 12*time.Hour, "sum(bee_snips_total_snips_proof_sent) + sum(bee_snips_total_snips_maintain_sent) + sum(bee_snips_total_snips_upload_done_sent)  + sum(bee_snips_total_snips_req_sent) + sum(bee_snips_total_snips_proof_recv) + sum(bee_snips_total_snips_maintain_recv) + sum(bee_snips_total_chunks_uploaded)")
		if err != nil {
			return reuploadResult{}, err
		}
		duration = subdur + dur

		neighbors, err := neighbor.GetMyNeighbors(ctx, ctl, uploaderID)
		if err != nil {
			return reuploadResult{}, err
		}
		for _, neighbor := range neighbors {
			neighborIDstr := strings.Split(neighbor.HostName, "-")
			neighborID, err := strconv.Atoi(neighborIDstr[len(neighborIDstr)-1])
			if err != nil {
				return reuploadResult{}, err
			}
			hasAllChunk, err := snapshot.HasAllChunks(ctx, ctl, neighborID, allChunks)
			if err != nil {
				return reuploadResult{}, err
			}
			if !hasAllChunk {
				return reuploadResult{}, fmt.Errorf("neighbor %d does not have all chunks", neighborID)
			}
		}
	case "steward":
		duration, err = reupload.Steward(ctx, ctl, uploaderID, fileAddr)
	default:
		return reuploadResult{}, fmt.Errorf("unknown experiment '%s'", name)
	}
	if err != nil {
		return reuploadResult{}, fmt.Errorf("failed to run POS: %w", err)
	}

	afterMetrics, err := metrics.GetMetrics(ctx, ctl)
	if err != nil {
		return reuploadResult{}, err
	}
	afterMetrics.Sub(beforeMetrics)

	var bw int64
	var additionbw int64

	switch name {
	case "pos":
		bw = calculatePOSBandwidth(metrics.Sum(afterMetrics))
	case "pullsync":
		bw, additionbw = calculatePullsyncBandwidth(metrics.Sum(afterMetrics))
	case "snips":
		bw = calculateSNIPSBandwidth(metrics.Sum(afterMetrics))
		additionbw = bw
	case "steward":
		bw = calculatePushsyncBandwidth(metrics.Sum(afterMetrics))
	default:
		panic("unknown experiment: " + name)
	}

	log.Printf("Reupload duration: %s, bandwidth: %s", duration, humanizeBytes(bw))

	if experimentWriteMetrics {
		f, err := os.OpenFile(filepath.Join(outputPath, "metrics.json"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.FilePerm)
		if err != nil {
			return reuploadResult{}, fmt.Errorf("failed to create metrics file: %w", err)
		}
		defer util.SafeClose(&err, f)

		enc := json.NewEncoder(f)
		err = enc.Encode(afterMetrics)
		if err != nil {
			return reuploadResult{}, fmt.Errorf("failed to write metrics: %w", err)
		}
	}

	return reuploadResult{
		Duration:          duration,
		Bandwidth:         bw,
		AdditionBandwidth: additionbw,
	}, nil
}

func runSinglePOS(ctx context.Context, ctl bee.Controller, outputPath, fileAddr string) (duration time.Duration, err error) {
	result, duration, err := reupload.POS(ctx, ctl, uploaderID, fileAddr)
	if err != nil {
		return 0, err
	}

	// write files
	if experimentWritePOSResult {
		f, err := os.OpenFile(filepath.Join(outputPath, "pos.json"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.FilePerm)
		if err != nil {
			return 0, fmt.Errorf("failed to create POS result file: %w", err)
		}
		defer util.SafeClose(&err, f)

		enc := json.NewEncoder(f)
		enc.SetIndent("", "\t")
		err = enc.Encode(result)
		if err != nil {
			return 0, fmt.Errorf("failed to write POS result: %w", err)
		}
	}
	return duration, err
}

func humanizeBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
