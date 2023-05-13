package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

// global_chunksInPod and newChunksInPods and chunkDeletionMap has Key: PodId, Value: Chunks
// global_podsWithChunks and newPodsWithChunks has Key: Chunk, Value: PodIds
var combinestorageCmd = &cobra.Command{
	Use:   "combinestorage",
	Short: "Combines storage for pods.",
	Long:  "Combines storage for pods according to the supplied list. The lists are newChunksInPods and generated from the [deletechunks] function and the --serialize flag.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatalf("Must pass a comma-separated list of newChunksInPods.")
			return
		}

		newChunksInPods := make(map[string][]string)
		newPodsWithChunks := make(map[string][]string)
		repList := make(map[string]int)
		filters := strings.Split(args[0], ",")

		initNewChunksInPods(filters, newChunksInPods)
		// Populate newPodsWithChunks list from newChunksInPods
		for pod, chunks := range newChunksInPods {
			for i := 0; i < len(chunks); i++ {
				list, ok := newPodsWithChunks[chunks[i]]
				if !ok {
					newPodsWithChunks[chunks[i]] = make([]string, 0)
				}
				newPodsWithChunks[chunks[i]] = append(list, pod)
				repList[chunks[i]]++
			}
		}

		if input_globalserialize != "" {
			initNewChunksInPods([]string{input_globalserialize}, global_chunksInPod)
			// Populate global_podsWithChunks list from global_chunksInPod
			for pod, chunks := range global_chunksInPod {
				for i := 0; i < len(chunks); i++ {
					list, ok := global_podsWithChunks[chunks[i]]
					if !ok {
						global_podsWithChunks[chunks[i]] = make([]string, 0)
					}
					global_podsWithChunks[chunks[i]] = append(list, pod)
				}
			}
		}

		newChunksInPods, chunkDeletionMap := generateDeletionMap(newPodsWithChunks, repList)

		if input_serialize != "" {
			chunksByte, err := serialize(newChunksInPods)
			if err != nil {
				log.Fatal(err)
			}
			if err := ioutil.WriteFile(input_serialize, chunksByte, os.ModePerm); err != nil {
				log.Fatal(err)
			}
		}
		if input_serializeDeletion != "" {
			chunksByte, err := serialize(chunkDeletionMap)
			if err != nil {
				log.Fatal(err)
			}
			if err := ioutil.WriteFile(input_serializeDeletion, chunksByte, os.ModePerm); err != nil {
				log.Fatal(err)
			}
		}

		fmt.Printf("Finished combining %v lists\n", len(filters))
	},
}

func init() {
	combinestorageCmd.Flags().IntVarP(&replicationFactor, "replication", "r", 1, "Desired maximum replication of each chunk")
	combinestorageCmd.Flags().StringVarP(&input_globalserialize, "globalserialize", "", "", "File containing global_chunksInPod with Key: PodId, Value: Chunks")
	combinestorageCmd.Flags().StringVarP(&input_serialize, "serialize", "", "", "Serialize newChunksInPods with Key: PodId, Value: Chunks")
	combinestorageCmd.Flags().StringVarP(&input_serializeDeletion, "serializeDeletion", "", "", "Serialize chunkDeletionMap with Key: PodId, Value: Chunks")
	combinestorageCmd.Flags().BoolVarP(&force_bakeDeletion, "forcebake", "", false, "Force bake deletion.")
	rootCmd.AddCommand(combinestorageCmd)
}

// global_chunksInPod and newChunksInPods and chunkDeletionMap has Key: PodId, Value: Chunks
func initNewChunksInPods(files []string, chunksInPod map[string][]string) {
	for i := 0; i < len(files); i++ {
		f, err := os.Open(files[i])
		if err != nil {
			log.Error(err)
		}
		defer f.Close()

		byteArr, err := ioutil.ReadAll(f)
		if err != nil {
			log.Error(err)
		}
		output := make(map[string][]string)
		err = json.Unmarshal(byteArr, &output)
		if err != nil {
			log.Error(err)
		}

		for pod, chunks := range output {
			knownChunks, ok := chunksInPod[pod]
			if !ok {
				chunksInPod[pod] = chunks
			} else {
				chunksInPod[pod] = append(knownChunks, chunks...)
			}
		}
	}
}
