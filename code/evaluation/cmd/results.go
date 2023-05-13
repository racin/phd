package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/racin/phd/code/evaluation/metrics"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

var (
	resultsOld bool
)

var bandwidthCommand = &cobra.Command{
	Use:   "bandwidth pos|steward resultsPath [outputPath]",
	Short: "calculate bandwidth from results",
	Long: `The bandwidth command calculates reupload bandwidth use
based on metrics collected from experiments.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		topLevelDir, err := os.ReadDir(args[1])
		if err != nil {
			return err
		}

		var rd resultsReader

		if resultsOld {
			var oldResults oldResults
			err = util.ReadJSONFile(path.Join(args[1], "results.json"), &oldResults)
			if err != nil {
				return err
			}
			rd = oldResults
		} else {
			var results results
			err = util.ReadJSONFile(path.Join(args[1], "results.json"), &results)
			if err != nil {
				return err
			}
			rd = results
		}

		allResults := make(results)

		for _, chunklossDirInfo := range topLevelDir {
			if !chunklossDirInfo.IsDir() {
				continue
			}

			runDirPath := filepath.Join(args[1], chunklossDirInfo.Name())
			runDir, err := os.ReadDir(runDirPath)
			if err != nil {
				return err
			}

			for _, runDirInfo := range runDir {
				if !runDirInfo.IsDir() {
					continue
				}

				run, err := strconv.Atoi(runDirInfo.Name())
				if err != nil {
					return err
				}

				chunklossDirPath := filepath.Join(runDirPath, runDirInfo.Name())
				chunklossDir, err := os.ReadDir(chunklossDirPath)
				if err != nil {
					return err
				}

				for _, fileDirInfo := range chunklossDir {
					if !fileDirInfo.IsDir() {
						continue
					}

					or, ok := rd.measurement(fileDirInfo.Name(), chunklossDirInfo.Name(), run)
					if !ok {
						log.Printf("missing results entry for file %s chunkloss %s run %s", fileDirInfo.Name(), chunklossDirInfo.Name(), runDirInfo.Name())
						continue
					}

					metricsFile := filepath.Join(chunklossDirPath, fileDirInfo.Name(), "metrics.json")
					var ms metrics.MetricsSlice
					err = util.ReadJSONFile(metricsFile, &ms)
					if err != nil {
						log.Printf("failed to read metrics file: %v", err)
						continue
					}

					fr, ok := allResults[fileDirInfo.Name()]
					if !ok {
						fr = fileResults{Runs: make(map[string]fileResult)}
						fr.FileSizeKB = or.FileSizeKB
					}

					rr := fr.Runs[chunklossDirInfo.Name()]
					rr.Duration = append(rr.Duration, or.Duration)

					var bw int64
					switch args[0] {
					case "pos":
						bw = calculatePOSBandwidth(metrics.Sum(ms))
					case "steward":
						fallthrough
					case "stewardship":
						bw = calculatePushsyncBandwidth(metrics.Sum(ms))
					default:
						return fmt.Errorf("unknown option '%s'", args[0])
					}

					rr.Bandwidth = append(rr.Bandwidth, bw)

					fr.Runs[chunklossDirInfo.Name()] = rr

					allResults[fileDirInfo.Name()] = fr
				}
			}
		}

		if len(args) == 3 {
			err = util.WriteJSONFile(args[2], util.FilePerm, allResults)
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "\t")
			err = enc.Encode(allResults)
		}

		return err
	},
}

var mergeCommand = &cobra.Command{
	Use:   "merge resultsPath1 resultsPath2 [outputPath]",
	Short: "merge two results files",
	Long:  `The merge command merges two results files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			results1, results2 results
		)

		err := util.ReadJSONFile(args[0], &results1)
		if err != nil {
			return err
		}

		err = util.ReadJSONFile(args[1], &results2)
		if err != nil {
			return err
		}

		// merge results2 files into results1
		for file, fr1 := range results1 {
			n1 := numCompletedRuns(fr1)
			if fr2, ok := results2[file]; ok {
				n2 := numCompletedRuns(fr2)
				// merge chunkloss results into fr1
				for chunkloss, rr1 := range fr1.Runs {
					rr2, ok := fr2.Runs[chunkloss]
					if !ok {
						continue
					}
					rr1.Duration = append(rr1.Duration[:n1], rr2.Duration[:n2]...)
					rr1.Bandwidth = append(rr1.Bandwidth[:n1], rr2.Bandwidth[:n2]...)

					fr1.Runs[chunkloss] = rr1
				}

				// add any chunklosses that are in fr2 but not in fr1
				for chunkloss, rr2 := range fr2.Runs {
					if _, ok := fr1.Runs[chunkloss]; !ok {
						fr1.Runs[chunkloss] = fileResult{
							Duration:  rr2.Duration[:n2],
							Bandwidth: rr2.Bandwidth[:n2],
						}
					}
				}

				results1[file] = fr1
			}
		}

		// add any files that are in results2 but not in results1
		for file, fr2 := range results2 {
			if _, ok := results1[file]; !ok {
				results1[file] = fr2
			}
		}

		if len(args) == 3 {
			err = util.WriteJSONFile(args[2], util.FilePerm, results1)
			if err != nil {
				return err
			}
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "\t")
			err = enc.Encode(results1)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func numCompletedRuns(fr fileResults) int {
	num := math.MaxInt
	for _, rr := range fr.Runs {
		if len(rr.Duration) < num {
			num = len(rr.Duration)
		}
	}
	return num
}

func init() {
	rootCmd.AddCommand(bandwidthCommand)
	rootCmd.AddCommand(mergeCommand)

	bandwidthCommand.Flags().BoolVar(&resultsOld, "old", false, "read existing results in the old format")
}

type measurement struct {
	Duration   int64
	Bandwidth  int64
	FileSizeKB int
}

type oldResults map[string]map[string]map[string]measurement

func (or oldResults) measurement(file, chunkloss string, run int) (measurement, bool) {
	m, ok := or[strconv.Itoa(run)][chunkloss][file]
	return m, ok
}

type fileResult struct {
	Nonce             uint64  `json:"nonce"`
	Duration          []int64 `json:"duration"`
	Bandwidth         []int64 `json:"bandwidth"`
	AdditionBandwidth []int64 `json:"addition_bandwidth"`
}

type fileResults struct {
	Runs       map[string]fileResult `json:"runs"`
	FileSizeKB int                   `json:"file_size_kb"`
}

type results map[string]fileResults

func (r results) measurement(file, chunkloss string, run int) (measurement, bool) {
	fr, ok := r[file]
	if !ok {
		return measurement{}, false
	}
	rr, ok := fr.Runs[chunkloss]
	if !ok {
		return measurement{}, false
	}
	if len(rr.Bandwidth) <= run || len(rr.Duration) <= run {
		return measurement{}, false
	}
	return measurement{
		Duration:   rr.Duration[run],
		Bandwidth:  rr.Bandwidth[run],
		FileSizeKB: fr.FileSizeKB,
	}, true
}

func calculatePOSBandwidth(metrics metrics.Metrics) int64 {
	bw := .0

	forwarded := metrics["bee_pos_total_challenges_forwarded"]

	// challenges sent by challenger + forwarders
	bw += (metrics["bee_pos_total_challenges"] + forwarded) * posChallengeSize

	// responses sent back + forwarders
	bw += (metrics["bee_pos_total_proofs_generated"] + forwarded) * posResponseSize

	return int64(bw) + calculatePushsyncBandwidth(metrics)
}

func calculatePushsyncBandwidth(metrics metrics.Metrics) int64 {
	bw := .0
	// total sent includes pusher and forwarders
	bw += metrics["bee_pushsync_total_sent"] * pushsyncDeliverySize

	bw += (metrics["bee_pusher_total_synced"] + metrics["bee_pushsync_forwarder"]) * pushsyncReceiptSize

	// include neighborhood replication
	bw += metrics["bee_handler_replication"] * (pushsyncDeliverySize + pushsyncReceiptSize)

	return int64(bw)
}

func calculateSNIPSBandwidth(metrics metrics.Metrics) int64 {
	bw := .0

	// TotalSNIPSproofSent (8 + 4 + 32 + 32) byte
	bw += metrics["bee_snips_total_snips_proof_sent"] * snipsSizeProof

	// SizeOfSNIPSsentmphf 1 byte
	bw += metrics["bee_snips_size_of_snips_sent_mphf"] * snipsSizeMPHF

	// SizeOfSNIPSsentBlockhash 1 byte
	bw += metrics["bee_snips_size_of_snips_sent_blockhash"] * snipsSizeBlockhash

	// SizeOfSNIPSsentSignature 0 byte
	bw += metrics["bee_snips_size_of_snips_sent_signature"] * snipsSizeSignature

	// TotalSNIPSmaintainSent 8 byte
	bw += metrics["bee_snips_total_snips_maintain_sent"] * snipsSizeMaintain

	// SizeOfSNIPSsentMaintainBV 1 byte
	bw += metrics["bee_snips_size_of_snips_sent_maintain_bv"] * snipsSizeMaintainBV

	// TotalSNIPSreqSent 8 byte
	bw += metrics["bee_snips_total_snips_req_sent"] * snipsSizeSNIPSreq

	return int64(bw)
}

func calculatePullsyncBandwidth(metrics metrics.Metrics) (int64, int64) {
	bw := .0

	// SizeOfCursorSent 8 byte
	bw += metrics["bee_pullsync_size_of_cursor_sent"] * pullsyncSizeCursor

	// TotalRuidSent 4 byte
	bw += metrics["bee_pullsync_total_ruid_sent"] * pullsyncSizeRuid

	// TotalRangeSent 20 byte
	bw += metrics["bee_pullsync_total_range_sent"] * pullsyncSizeRange

	// TotalOfferSent 8 byte
	bw += metrics["bee_pullsync_total_offer_sent"] * pullsyncSizeSent

	// SizeOfWantSent 1
	bw += metrics["bee_pullsync_size_of_want_sent"] * pullsyncSizeWant

	additionbw := bw

	// TotalChunksInOffer 32 byte
	bw += metrics["bee_pullsync_total_chunks_in_offer"] * pullsyncChunksInOffer

	// TotalChunksInWant 32 byte. This captures how many chunks would be offered for new chunks
	// if cursors were not cleared
	additionbw += metrics["bee_pullsync_total_chunks_in_want"] * pullsyncChunksInOffer

	return int64(bw), int64(additionbw)
}

type resultsReader interface {
	measurement(file, chunkloss string, run int) (m measurement, ok bool)
}
