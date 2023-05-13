package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/ethersphere/swarm/chunk"
	"github.com/racin/snarl/swarmconnector"
	"github.com/spf13/cobra"
)

var listchunksSigCmd = &cobra.Command{
	Use:   "listchunks_sig [checklog] [chunklist]",
	Short: "List all chunks stored in the peers",
	Long:  "List all chunks stored in the peers. Accepts multiple JSON-encoded files to filter the listing.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			log.Fatal("Need to pass 2 arguments.")
		}
		folder := "/Users/racin/Documents/Skole/PhD/SnarlExperiments061120/checklogs/"
		// Parse chunk list
		nodeToChunk, chunkToNode, _ := getChunkDist(folder + args[0])
		//fmt.Printf("ChunkToNode: %v\n", chunkToNode)

		if len(args) == 3 && args[2] == "nodeToChunk" {
			result := marshalNodeToChunk(nodeToChunk)

			fmt.Printf("%v\n", string(result))
		} else {
			// Parse replication file
			result := parseRepfile(folder+args[1], chunkToNode)

			fmt.Printf("%v\n", string(result))
		}

	},
}

func marshalNodeToChunk(nodeToChunk map[int][]string) []byte {
	output := make(map[int]int)

	minVal, maxVal, totalVal := math.MaxInt64, 0, 0
	for key, val := range nodeToChunk {
		lVal := len(val)
		if lVal < minVal {
			minVal = lVal
		}
		if lVal > maxVal {
			maxVal = lVal
		}
		totalVal += lVal
		output[key] = lVal
	}
	fmt.Printf("Min: %v, Max: %v, Total: %v\n", minVal, maxVal, totalVal)
	bytes, _ := json.Marshal(sortMapSpecial(output))
	return bytes
}

func sortMapSpecial(themap map[int]int) map[int]int {
	type keyvalue struct {
		Key   int
		Value int
	}
	mapKeyVal := make([]keyvalue, len(themap))

	i := 0
	for key, val := range themap {
		mapKeyVal[i] = keyvalue{key, val}
		i++
	}

	sort.Slice(mapKeyVal, func(a, b int) bool {
		return mapKeyVal[a].Value < mapKeyVal[b].Value
	})

	sorted := make(map[int]int)
	for i := 0; i < len(mapKeyVal); i++ {
		fmt.Printf("MapKeyVal %v\n", mapKeyVal[i])
		sorted[i] = mapKeyVal[i].Value
	}

	return sorted
}

func parseRepfile(file string, chunkToNode map[string][]int) []byte {
	f, _ := os.Open(file)
	defer f.Close()

	byteArr, _ := ioutil.ReadAll(f)
	output := make(map[string]*chunkInfo_new)
	_ = json.Unmarshal(byteArr, &output)

	var leaves uint64 = 0
	for key, val := range output {
		if nodes, ok := chunkToNode[key]; ok {
			val.Replication = len(nodes)
		}
		if val.Depth == 1 {
			leaves++
		}
	}
	sizeList, _ := swarmconnector.GenerateChunkMetadata(leaves * chunk.DefaultSize)
	// fmt.Printf("SL: %+v\n", sizeList)
	for _, val := range output {
		val.Parent = sizeList[val.Index-1].Parent
	}
	outBytes, _ := json.Marshal(output)
	return outBytes
}

type chunkInfo_old struct {
	Index int `json:"index"`
	Depth int `json:"depth"`
}

type chunkInfo_new struct {
	Index       int `json:"index"`
	Depth       int `json:"depth"`
	Replication int `json:"replication"`
	Parent      int `json:"parent"`
}

func init() {
	rootCmd.AddCommand(listchunksSigCmd)
}

func getChunkDist(checklog string) (nodeToChunk map[int][]string, chunkToNode map[string][]int, err error) {
	f, err := os.Open(checklog)
	if err != nil {
		return
	}
	defer f.Close()

	nodeToChunk, chunkToNode = make(map[int][]string), make(map[string][]int)
	// var lastPod string
	//chunkToID := make(map[string]int)
	chunksFound, nodesFound, totalChunks := 0, 0, 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		str := scanner.Text()
		// fmt.Printf("Str: %v", str)
		// break

		if strings.Contains(str, "TOTAL CHUNKS:") {
			// lastPod = "x"
			continue
		}

		split := strings.Split(str, "chunk addr FOR NODE swarm-private-")

		if len(split) == 2 {
			// lastPod = split[1]
			nodesFound++
			nodeToChunk[nodesFound] = make([]string, 0)
			continue
		}

		if strings.HasPrefix(str, "Item addr: ") {
			keyval := strings.Split(str, ",")
			key := keyval[0][11:]
			chunksFound++
			/*if _, ok := chunkToID[key]; !ok {
				chunksFound++
				chunkToID[key] = chunksFound
			}
			chunkID := chunkToID[key]*/
			nodeToChunk[nodesFound] = append(nodeToChunk[nodesFound], key)
			chunkToNode[key] = append(chunkToNode[key], nodesFound)
			totalChunks++
		}
	}
	// fmt.Printf("NodeToChunk: %v, ChunkToNode: %v\n", nodeToChunk, chunkToNode)
	// fmt.Printf("Number of nodes: %v, Unique chunks: %v, Total chunks: %v\n",
	// 	nodesFound, chunksFound, totalChunks)

	return
}
