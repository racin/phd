package main

import (
	"container/heap"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ethersphere/swarm/shed"
	"github.com/nolash/go-ethereum/common/hexutil"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

var input_podRemoval, input_chunkRemoval, input_serialize, input_deletionmap, input_serializeDeletion, input_globalserialize string

var global_toBeDeleted []string
var replicationFactor, input_proportionalRep int
var do_bakeDeletion, fullPath, force_bakeDeletion, input_uniformrep bool

// chunkInPods: Key: PodId, Value: Array with ChunkHash
var global_chunksInPod map[string][]string = make(map[string][]string)

var deletechunksCmd = &cobra.Command{
	Use:   "deletechunks [numpeers]",
	Short: "Delete chunks from LevelDB.",
	Long:  "Delete chunks from LevelDB.",
	Run: func(cmd *cobra.Command, args []string) {
		if input_deletionmap != "" {
			deleteChunks_pods(input_deletionmap)
			return
		}
		var numpeers int
		var err error
		if only_Localhost {
			numpeers = 0
		} else {
			if len(args) == 0 {
				log.Fatalf("Must specify number of peers.")
			}
			numpeers, err = strconv.Atoi(args[0])
			if err != nil {
				log.Fatalf("Could not convert [numpeers] to integer")
			}
		}

		if !do_bakeDeletion && input_podRemoval == "" && input_chunkRemoval == "" {
			log.Fatalf("Must specify podRemoval or chunkRemoval.")
		}

		global_searchChunks = strings.Split(input_searchForChunks, ",")

		if input_chunkFilter != "" {
			initChunkFilter(strings.Split(input_chunkFilter, ","))
			// Debug
			// buildChunkRepList(global_chunkFilters, replicationFactor, input_proportionalRep, input_uniformrep)
			// return
		}
		if input_verifyPersistence != "" {
			initVerifyPersist(strings.Split(input_verifyPersistence, ","))
		}

		if only_Localhost {
			if do_Copychunks {
				if err = copyChunkDBFromLocalhost(); err != nil {
					log.Error(err)
				}
			}
			deleteChunks_localhost()
		} else if replicationFactor <= 0 {
			log.Fatal("Replication must be over 0.")
		} else {
			if input_podFilter != "" {
				initPodFilter(strings.Split(input_podFilter, ","), numpeers)
			}
			if do_Copychunks {
				if err = copyChunkDBFromPods(numpeers); err != nil {
					log.Error(err)
				}
			}
			populateGlobalMaps_pods(numpeers)

			if input_globalserialize != "" {
				chunksByte, err := serialize(global_chunksInPod)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("Global serialze is %v bytes.\n", len(chunksByte))
				if err := ioutil.WriteFile(input_globalserialize, chunksByte, os.ModePerm); err != nil {
					log.Fatal(err)
				}
				log.Info("global_chunksInPod serialized.\n")
				return
			}

			// global_chunksInPod and newChunksInPods and chunkDeletionMap has Key: PodId, Value: Chunks
			// global_podsWithChunks and newPodsWithChunks has Key: Chunk, Value: PodIds
			repList := buildChunkRepList(global_chunkFilters, replicationFactor, input_proportionalRep, input_uniformrep)
			newChunksInPods, newPodsWithChunks, chunkDeletionMap := bakeDeletion(repList)
			verifyPersistenceOldList()
			verifyPersistenceNewList(newChunksInPods, newPodsWithChunks)

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
		}

	},
}

func init() {
	deletechunksCmd.Flags().StringVarP(&input_localChunkDir, "localchunkdir", "", "podChunks/", "Local chunk directory")
	deletechunksCmd.Flags().StringVarP(&input_podRemoval, "podRemoval", "", "", "Comma-seperated list of pod IDs. All chunks stored by these pods will be deleted from the network.")
	deletechunksCmd.Flags().StringVarP(&input_chunkRemoval, "chunkRemoval", "", "", "JSON-encoded file(s) to remove the chunks listed [comma separated]")
	deletechunksCmd.Flags().StringVarP(&input_serialize, "serialize", "", "", "Serialize newChunksInPods with Key: PodId, Value: Chunks")
	deletechunksCmd.Flags().StringVarP(&input_globalserialize, "globalserialize", "", "", "Serialize global_chunksInPod with Key: PodId, Value: Chunks")
	deletechunksCmd.Flags().StringVarP(&input_serializeDeletion, "serializeDeletion", "", "", "Serialize chunkDeletionMap with Key: PodId, Value: Chunks")
	deletechunksCmd.Flags().StringVarP(&input_deletionmap, "deletemap", "", "", "Deletion map with chunkDeletionMap with Key: PodId, Value: Chunks")
	deletechunksCmd.Flags().IntVarP(&listChunkFreq, "frequency", "f", 1000, "Frequency of listing a pods")
	deletechunksCmd.Flags().BoolVarP(&verbose_ListChunks, "verboselistchunks", "", true, "Print the replication factor for every chunk.")
	deletechunksCmd.Flags().StringVarP(&input_chunkFilter, "filters", "", "", "JSON-encoded file(s) to filter the chunks listed [comma separated]")
	deletechunksCmd.Flags().BoolVarP(&verbose_ListPods, "verboselistpods", "", false, "Print chunks per pod.")
	deletechunksCmd.Flags().BoolVarP(&do_bakeDeletion, "bakedeletion", "", true, "Do bake deletion.")
	deletechunksCmd.Flags().BoolVarP(&force_bakeDeletion, "forcebake", "", false, "Force bake deletion.")
	deletechunksCmd.Flags().BoolVarP(&only_Localhost, "localhost", "", false, "Check chunks stored on localhost (On this very machine!)")
	deletechunksCmd.Flags().BoolVarP(&fullPath, "fullPath", "", true, "True: Chunks stored in podChunks/pv-X-backup/swarm/bzz-*/chunks/, False: Chunks stored in podChunks/swarm-private-X/")
	deletechunksCmd.Flags().BoolVarP(&do_Copychunks, "copychunks", "", true, "Copy the chunks from the pods (or localhost)")
	deletechunksCmd.Flags().IntVarP(&replicationFactor, "replication", "r", 0, "Desired maximum replication of each chunk")
	deletechunksCmd.Flags().StringVarP(&input_verifyPersistence, "verifypersistence", "", "", "JSON-encoded file(s) to verify the persistence in the network [comma separated]")
	deletechunksCmd.Flags().IntVarP(&input_proportionalRep, "proprep", "", 5, "If replication is non-uniform, this parameter will add additional replicas of tree nodes until the proportional replication of vanilla Swarm is reached.")
	deletechunksCmd.Flags().BoolVarP(&input_uniformrep, "uniformrep", "", true, "Generates replication list uniformerly, equal to the -replication flag.")
	rootCmd.AddCommand(deletechunksCmd)
}

var global_deleteList [][]byte = make([][]byte, 0)

func deleteItemsFunc(item shed.Item) (stop bool, err error) {
	str := fmt.Sprintf("%064x", item.Address)

	if shouldFilterChunk(str) {
		return false, nil
	}

	if str == "fffff7a144713b39e969fab084e4dbef8b9ce296e2f94220344e28f16f0d4273" {
		fmt.Printf("!!! %v Addr: %v  , Bytecast: %v --\n", str, item.Address, []byte(str))
		global_deleteList = append(global_deleteList, item.Address)
	}

	// if containsStr(global_toBeDeleted, str) {
	// 	global_deleteList[str] = struct{}{}
	// }

	return false, nil
}

func getTotalChunks(repList map[string]int) int {
	total := 0
	for _, val := range repList {
		total += val
	}
	return total
}

// buildChunkRepList increases the replication of the Parity tree nodes, as they are not entangled.
func buildChunkRepList(chunkfilters map[string]map[string]*chunkInfo, leafRep, propRep int, uniform bool) map[string]int {
	repList, treeNodes := make(map[string]int), make(map[string]int)
	dataNodes, totalNodes, maxDepth := math.MaxInt64, 0, 0

	// Find the minimal filter, and then skip it.
	// The minimal filter will be the data tree and this is entangled, and thus doenst need additional replication.
	for _, filter := range global_chunkFilters {
		lenfilter := len(filter)
		totalNodes += lenfilter
		if lenfilter < dataNodes {
			dataNodes = lenfilter
		}
	}
	for _, filter := range global_chunkFilters {
		lenfilter := len(filter)

		// Give all nodes in the data tree the leaf replication.
		if lenfilter == dataNodes {
			for key := range filter {
				repList[key] = leafRep
			}
			continue
		}

		for key, val := range filter {
			if val.Depth > maxDepth {
				maxDepth = val.Depth
			}
			if val.Depth == 1 || uniform {
				repList[key] = leafRep
			} else {
				treeNodes[key] = val.Depth
			}
		}
	}

	if uniform {
		fmt.Printf("repList: %v\n", repList)
		fmt.Printf("TOTAL: %v\n", getTotalChunks(repList))
		return repList
	}
	numChunksToAssign := (propRep * dataNodes) - ((totalNodes - len(treeNodes)) * leafRep)
	treeRep := int(float64(numChunksToAssign) / float64(len(treeNodes)))
	fmt.Printf("Tree rep: %v, math.Ceil: %f, LenTreeNodes %f\n", treeRep, float64(propRep*dataNodes-totalNodes*leafRep)/float64(len(treeNodes)),
		float64(len(treeNodes)))
	numChunksToAssign = numChunksToAssign - (treeRep * len(treeNodes))

	// Set replication factor for all tree nodes
	for key := range treeNodes {
		repList[key] = treeRep
	}

	// If we have some to spare, add them to the roots
	fmt.Printf("To add: %v\n", numChunksToAssign)
	for numChunksToAssign > 0 {
		for key, val := range treeNodes {
			if numChunksToAssign == 0 {
				break
			}
			if val == maxDepth {
				repList[key]++
				numChunksToAssign--
			}
		}
	}
	for key, val := range repList {
		if val != leafRep {
			fmt.Printf("Key: %v, Rep: %v\n", key, val)
		}
	}
	fmt.Printf("Calculated TreeRep: %v, NumTreeNodes: %v, Total Nodes: %v, Data Nodes: %v, Replicated Total Nodes: %v, Total Entangled nodes: %v\n",
		treeRep, len(treeNodes), totalNodes, dataNodes, propRep*dataNodes, getTotalChunks(repList))

	return repList
}

// global_chunksInPod and deleteChunksInPods has Key: PodId, Value: Chunks
func deleteChunks_pods(deletionmap string) {
	delmapBytes, err := ioutil.ReadFile(deletionmap)
	if err != nil {
		log.Error(err)
	}
	deleteChunksInPods, err := deserialize(delmapBytes)
	if verbose_ListPods {
		for key, val := range deleteChunksInPods {
			fmt.Printf("** Pod %v contains %v chunks. Listed: \n", key, len(val))
			sort.Strings(val)
			for i := 0; i < len(val); i = i + listChunkFreq {
				fmt.Printf("***** Chunk: %v\n", val[i][:8])
			}
		}
	}
	counter := 0
	for pod, chunks := range deleteChunksInPods {
		global_currPod = pod

		podPath := input_localChunkDir + pod

		// Try to find matching directory with wildcard asterisk
		if matches, err := filepath.Glob(podPath); len(matches) == 0 || err != nil {
			fmt.Printf("Did not find chunks. " + err.Error())
			continue
		} else {
			if matches[0] == podPath {
				podPath = matches[1]
			} else {
				podPath = matches[0]
			}

		}
		counter++
		fmt.Printf("Working in %v. Number: %v\n", podPath, counter)
		db, err := shed.NewDB(podPath, "racin")
		// for j := 0; j < 5; j++ {
		// 	if err != nil {
		// 		fmt.Printf("Retrying copying for pod: %v\n", pod)
		// 		copyChunkDBFromSinglePod(pod)
		// 		db, err = shed.NewDB(podPath, "racin")
		// 	}
		// }
		if err != nil {
			log.Errorf("Could not copy chunks for pod: %v. Error: %v. Giving up.\n", pod, err.Error())
			break
		}

		global_currChunkCnt = 0

		index, err := db.NewIndex("Address->StoreTimestamp|BinID|Data", shed.IndexFuncs{
			EncodeKey: func(fields shed.Item) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (g shed.Item, err error) {
				g.Address = key
				return g, nil
			},
			EncodeValue: encodeValueFunc,
			DecodeValue: decodeValueFunc,
		})

		icount1, _ := index.Count()
		fmt.Printf("Count index: %v\n", icount1)
		for i := 0; i < len(chunks); i++ {
			byteAddr, err := hexutil.Decode("0x" + chunks[i])
			key := shed.Item{
				Address: byteAddr,
			}

			if yes, _ := index.Has(key); yes {
				fmt.Printf("Deleting %v, from pod: %v\n", chunks[i][:8], pod)
				err = index.Delete(key)
				if err != nil {
					fmt.Printf("Error deleting: %064x\n", err)
				}
			} else {
				fmt.Printf("Pod %v did not have chunk %v.\n", pod, chunks[i][:8])
			}
		}

		// Close the DB!
		if err := db.Close(); err != nil {
			log.Error(err)
		}
	}
}

func verifyPersistenceOldList() {
	fmt.Printf("----------------------------------------------------------\n " +
		"VERIFYING OLD GLOBAL LIST\n" +
		"----------------------------------------------------------\n")
	var totalChunks int
	for _, v := range sortedKeys(global_ChunkCount) {
		if verbose_ListPods {
			fmt.Printf("Key: %v, Val: %v\n", v, global_ChunkCount[v])
		}
		totalChunks += global_ChunkCount[v]
	}

	var missingCnt int
	var outputStr string
	for filename, chunkList := range global_verifyPersist {
		missingCnt = 0
		outputStr = ""
		for key, val := range chunkList {
			if _, ok := global_ChunkCount[key]; !ok {
				outputStr += fmt.Sprintf("*** MISSING %v. Index: %v, Depth: %v\n", key, val.Index, val.Depth)
				missingCnt++
			}
		}
		fmt.Printf(" ----- [%v] Verifying persistence with list supplied in %v ----- \n%v", missingCnt, filename, outputStr)
		if missingCnt == 0 {
			fmt.Print("### Persistence verified ###\n")
		}
	}

	// for key, val := range global_podsWithChunks {
	// 	fmt.Printf("Pods with chunk: %v - [[ %v ]]\n", key, val)
	// }

	fmt.Printf("Total chunks found: %v\nUnique chunks found: %v\n", totalChunks, len(global_ChunkCount))
	fmt.Printf("Average replication factor: %v\n", float64(totalChunks)/float64(len(global_ChunkCount)))
}

// global_chunksInPod and newChunksInPods has Key: PodId, Value: Chunks
// global_podsWithChunks and newPodsWithChunks has Key: Chunk, Value: PodIds
func verifyPersistenceNewList(newChunksInPods, newPodsWithChunks map[string][]string) {
	fmt.Printf("----------------------------------------------------------\n " +
		"VERIFYING NEW BAKED LISTS AFTER DELETION\n" +
		"----------------------------------------------------------\n")
	var newChunkCount map[string]int = make(map[string]int)
	for key, val := range newPodsWithChunks {
		newChunkCount[key] = len(val)
	}
	var totalChunks int
	for _, v := range sortedKeys(newChunkCount) {
		if verbose_ListPods {
			fmt.Printf("Key: %v, Val: %v\n", v, newChunkCount[v])
		}
		totalChunks += newChunkCount[v]
	}

	if verbose_ListChunks {
		for key, val := range newChunksInPods {
			freqCnt := 0
			fmt.Printf("LISTING every %vth chunk addr FOR NODE %v. Count: %v\n", listChunkFreq, key, len(val))
			for i := 0; i < len(val); i++ {
				if freqCnt%listChunkFreq == 0 {
					fmt.Printf("Item addr: %v\n", val[i])
				}
				freqCnt++
			}
			fmt.Printf("---------------------------------\n")
		}
	}
	var missingCnt int
	var outputStr string
	for filename, chunkList := range global_verifyPersist {
		missingCnt = 0
		outputStr = ""
		for key, val := range chunkList {
			if _, ok := newChunkCount[key]; !ok {
				outputStr += fmt.Sprintf("*** MISSING %v. Index: %v, Depth: %v\n", key, val.Index, val.Depth)
				missingCnt++
			}
		}
		fmt.Printf(" ----- [%v] Verifying persistence with list supplied in %v ----- \n%v", missingCnt, filename, outputStr)
		if missingCnt == 0 {
			fmt.Print("### Persistence verified ###\n")
		}
	}

	// for key, val := range newPodsWithChunks {
	// 	fmt.Printf("Pods with chunk: %v - [[ %v ]]\n", key, val)
	// }

	fmt.Printf("Total chunks found: %v\nUnique chunks found: %v\n", totalChunks, len(newChunkCount))
	fmt.Printf("Average replication factor: %v\n", float64(totalChunks)/float64(len(newChunkCount)))
}

func populateGlobalMaps_pods(numPods int) {
	var db *shed.DB
	var err error
	for i := 0; i < numPods; i++ {
		if _, ok := global_podFilter[i]; ok {
			fmt.Printf("Skipping pod: %v\n", i)
			continue
		}
		var pod string
		if fullPath {
			pod = "pv-" + strconv.Itoa(i) + "-backup/swarm/bzz-*/chunks"
		} else {
			pod = "swarm-private-" + strconv.Itoa(i)
		}

		global_currPod = pod

		podPath := input_localChunkDir + pod

		if fullPath {
			// Try to find matching directory with wildcard asterisk
			if matches, err := filepath.Glob(podPath); len(matches) == 0 || err != nil {
				fmt.Printf("Did not find chunks. " + err.Error())
				continue
			} else {
				if matches[0] == podPath {
					podPath = matches[1]
				} else {
					podPath = matches[0]
				}
			}
		}

		db, err = shed.NewDB(podPath, "racin")
		for j := 0; j < 5; j++ {
			if err != nil {
				fmt.Printf("Retrying copying for pod: %v\n", pod)
				copyChunkDBFromSinglePod(pod)
				db, err = shed.NewDB(podPath, "racin")
			}
		}
		if err != nil {
			log.Errorf("Could not copy chunks for pod: %v. Giving up.\n", pod)
			continue
		}
		global_currChunkCnt = 0

		i, err := db.NewIndex("Address->StoreTimestamp|BinID|Data", shed.IndexFuncs{
			EncodeKey: func(fields shed.Item) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (e shed.Item, err error) {
				e.Address = key
				return e, nil
			},
			EncodeValue: encodeValueFunc,
			DecodeValue: decodeValueFunc,
		})

		if err != nil {
			log.Error(err)
		}

		err = i.Iterate(iterChunkCounter, nil)

		db.Close()
		if err != nil {
			log.Error(err)
		}

		if verbose_ListChunks {
			fmt.Printf("Pod: %v. TOTAL CHUNKS: %v\n", pod, global_currChunkCnt)
		}
	}
}

func copyChunkToPod(chunkAddr, fromPod, toPod string) error {
	var err error
	var fromPath, toPath string
	if fullPath {
		fromPath = input_localChunkDir + fromPod
		toPath = input_localChunkDir + toPod

		// Try to find matching directory with wildcard asterisk
		matchesFrom, err := filepath.Glob(fromPath)
		if err != nil {
			fmt.Printf("Did not find chunks. " + err.Error())
			return err
		}
		matchesTo, err := filepath.Glob(toPath)
		if len(matchesFrom) == 0 || len(matchesTo) == 0 || err != nil {
			fmt.Printf("Did not find chunks. " + err.Error())
			return err
		} else {
			if matchesFrom[0] == fromPath {
				fromPath = matchesFrom[1]
				toPath = matchesTo[1]
			} else {
				fromPath = matchesFrom[0]
				toPath = matchesTo[0]
			}
		}
	} else {
		fromPath = input_localChunkDir + fromPod
		toPath = input_localChunkDir + toPod
	}

	fromDB, err := shed.NewDB(fromPath, "racin")
	if err != nil {
		log.Errorf("Error open DB for %v. Err: %v\n", fromPod, err.Error())
		return err
	}
	defer fromDB.Close()
	toDB, err := shed.NewDB(toPath, "racin")
	if err != nil {
		log.Errorf("Error open DB for %v. Err: %v\n", toDB, err.Error())
		return err
	}
	defer toDB.Close()
	fromIndex, err := fromDB.NewIndex("Address->StoreTimestamp|BinID|Data", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			return fields.Address, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.Address = key
			return e, nil
		},
		EncodeValue: encodeValueFunc,
		DecodeValue: decodeValueFunc,
	})

	if err != nil {
		return err
	}

	toIndex, err := toDB.NewIndex("Address->StoreTimestamp|BinID|Data", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			return fields.Address, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.Address = key
			return e, nil
		},
		EncodeValue: encodeValueFunc,
		DecodeValue: decodeValueFunc,
	})

	if err != nil {
		return err
	}

	byteAddr, err := hexutil.Decode("0x" + chunkAddr)
	key := shed.Item{
		Address: byteAddr,
	}

	if yes, _ := fromIndex.Has(key); yes {
		fmt.Printf("Copying chunk %v from %v to %v\n", chunkAddr[:8], fromPod, toPod)
		toPut, err := fromIndex.Get(key)
		if err == nil {
			err = toIndex.Put(toPut)
			//db.Put(toPut.Address, toPut.Data)
		}
		if err != nil {
			fmt.Printf("Error putting chunk to %v\n", toPod)
			return err
		}
	} else {
		fmt.Printf("Pod %v did not have chunk %v.\n", fromPod, chunkAddr[:8])
	}

	return nil
}

// Rule 1: All pods must have some chunks
// Rule 2: Cardinality of unique chunks must be equal after deletion
// Rule 3: Pods can not get new chunks
// Rule 4: Replication factor for each chunk must be as specified.
// global_chunksInPod and newChunksInPods and chunkDeletionMap has Key: PodId, Value: Chunks
// global_podsWithChunks and newPodsWithChunks has Key: Chunk, Value: PodIds
func bakeDeletion(repList map[string]int) (newChunksInPods, newPodsWithChunks, chunkDeletionMap map[string][]string) {
	numUniquePods := len(global_chunksInPod)
	newPodsWithChunks = make(map[string][]string)
	pq := make(PriorityQ, numUniquePods)
	// Sort chunks by replication factor. Highest replication is element 0. Lowest replication is the last element.
	sorted_Chunks_Replication := sortedKeys(global_ChunkCount)
	fmt.Printf("First element: %v, Replication: %v \n", sorted_Chunks_Replication[0], len(global_podsWithChunks[sorted_Chunks_Replication[0]]))
	fmt.Printf("Second element: %v, Replication: %v \n", sorted_Chunks_Replication[1], len(global_podsWithChunks[sorted_Chunks_Replication[1]]))
	fmt.Printf("Last element: %v, Replication: %v \n", sorted_Chunks_Replication[len(sorted_Chunks_Replication)-1], len(global_podsWithChunks[sorted_Chunks_Replication[len(sorted_Chunks_Replication)-1]]))

	// Find the chunkcount of the largest pod.
	maxChunkCount := 0
	for _, chunks := range global_chunksInPod {
		if len(chunks) > maxChunkCount {
			maxChunkCount = len(chunks)
		}
	}

	index := 0
	for key, chunks := range global_chunksInPod {
		// Those with the lowest number of chunks must be prioritied first. We set priority to be inverse proportional with the chunkcount of the largest pod.
		pq[index] = &PodPriority{
			index:    index,
			id:       key,
			priority: len(chunks) - maxChunkCount,
		}
		fmt.Printf("Added %+v\n", pq[index])
		index = index + 1

	}
	heap.Init(&pq)

	// Create chunk map
	chunkmap := make(map[string]map[string]struct{})
	for key, val := range global_chunksInPod {
		chunkmap[key] = make(map[string]struct{})
		for i := 0; i < len(val); i++ {
			chunkmap[key][val[i]] = struct{}{}
		}
	}

	assigned := 0
	chunksToAssign := getTotalChunks(repList) //numUniqueChunks * replication // Get Total chunks
	for i := 0; i < chunksToAssign; i++ {
		if len(pq) == 0 {
			fmt.Printf("Exhausted all pods. Quitting.\n")
			break
		}
		pod := heap.Pop(&pq).(*PodPriority)
		if i%1000 == 0 {
			fmt.Printf("Popped: %+v\n", pod)
		}

		// Available chunks for a pod
		//chunklist := global_chunksInPod[pod.id]

		oldAssigned := assigned
		// Iterate chunks in order. Starting from highest replication and assigning it first.
		// for j := len(sorted_Chunks_Replication) - 1; j >= 0; j-- {
		for j := 0; j < len(sorted_Chunks_Replication); j++ {
			// Check if this chunk belong to this pod
			// if j%10000 == 0 {
			// 	//fmt.Printf("Trying chunk %v. Rep: %v. J: %v\n", sorted_Chunks_Replication[j], len(global_podsWithChunks[sorted_Chunks_Replication[j]]), j)
			// }

			// Try next chunk if the pod did not originally store it.
			if _, ok := chunkmap[pod.id][sorted_Chunks_Replication[j]]; !ok {
				continue
			}

			// Check if we already added this chunk to the pod.
			if containsStr(newPodsWithChunks[sorted_Chunks_Replication[j]], pod.id) {
				continue
			}

			podlist, ok := newPodsWithChunks[sorted_Chunks_Replication[j]]
			if !ok {
				newPodsWithChunks[sorted_Chunks_Replication[j]] = make([]string, 0)
			}

			// Pick one that is replicated less time than requested
			if len(podlist) < repList[sorted_Chunks_Replication[j]] {
				assigned++
				if assigned%1 == 0 {
					fmt.Printf("Assigned chunk num %v of %v\n", assigned, chunksToAssign)
				}

				// Add this chunk to the pod
				newPodsWithChunks[sorted_Chunks_Replication[j]] = append(podlist, pod.id)

				// Update priority of pod
				pod.priority += maxChunkCount

				heap.Push(&pq, pod)
				//pq.update(*pod, pod.id, pod.priority-1)

				// Continue to the next pod.
				break
			}
		}
		if oldAssigned == assigned {
			if force_bakeDeletion {
				// Find most (originally) replicated unassigned chunk and assign this to the pod
				chunk, err := getMostReplicatedUnassChunk(sorted_Chunks_Replication, "", repList, newPodsWithChunks)

				if err == nil {
					// Add this chunk to the pod
					fmt.Printf("Forcebly added chunk %v to pod %v \n", chunk, pod)
					assigned++

					newPodsWithChunks[chunk] = append(newPodsWithChunks[chunk], pod.id)

					// Open DB and add this chunk.
					if err := copyChunkToPod(chunk, global_podsWithChunks[chunk][0], pod.id); err != nil {
						log.Error(err)
					}

					// Update priority of pod
					pod.priority += maxChunkCount

					heap.Push(&pq, pod)

					continue
				}
			}
			fmt.Printf("***** Was not able to assign anything to Pod %+v. Force: %v\n", pod, force_bakeDeletion)
			i--
		}
	}
	fmt.Printf("Reached the end of reduction. Average replication: %f\n", float64(getTotalChunks(repList))/float64(len(repList)))

	newChunksInPods, chunkDeletionMap = generateDeletionMap(newPodsWithChunks, repList)
	return
}

func generateDeletionMap(newPodsWithChunks map[string][]string, repList map[string]int) (newChunksInPods, chunkDeletionMap map[string][]string) {
	numUniqueChunks := len(global_podsWithChunks)
	numUniquePods := len(global_chunksInPod)
	// Populate newChunksInPods list from newPodsWithChunks
	newChunksInPods = make(map[string][]string)
	for key, val := range newPodsWithChunks {
		for i := 0; i < len(val); i++ {
			list, ok := newChunksInPods[val[i]]
			if !ok {
				newChunksInPods[val[i]] = make([]string, 0)
			}
			newChunksInPods[val[i]] = append(list, key)
		}
	}

	// Create new chunk map
	newChunkmap := make(map[string]map[string]struct{})
	for pod, chunks := range newChunksInPods {
		newChunkmap[pod] = make(map[string]struct{})
		for i := 0; i < len(chunks); i++ {
			newChunkmap[pod][chunks[i]] = struct{}{}
		}
	}

	// Create chunk deletion map
	chunkDeletionMap = make(map[string][]string)
	for pod, chunks := range global_chunksInPod {
		chunkDeletionMap[pod] = make([]string, 0)

		for i := 0; i < len(chunks); i++ {
			// Check if this chunk is added to newChunksInPods
			if _, ok := newChunkmap[pod][chunks[i]]; ok {
				continue
			}
			chunkDeletionMap[pod] = append(chunkDeletionMap[pod], chunks[i])
		}
	}

	// Debug print
	// for key, val := range newChunksInPods {
	// 	fmt.Printf("** Pod %v contains %v chunks. Listed: \n", key, len(val))
	// 	for i := 0; i < len(val); i = i + 1000 {
	// 		fmt.Printf("***** Chunk: %v\n", val[i][:8])
	// 	}
	// }

	// Rule 1: All pods must have some chunks
	if numUniquePods != len(newChunksInPods) {
		var msg = fmt.Sprintf("Number of unique Pods do not match. Old: %v, New %v.\n", numUniquePods, len(newChunksInPods))
		if force_bakeDeletion {
			fmt.Print(msg)
		} else {
			log.Fatal(msg)
		}
	}

	// Rule 2: Cardinality of unique chunks must be equal after deletion
	if numUniqueChunks != len(newPodsWithChunks) {
		var msg = fmt.Sprintf("Number of unique Chunks do not match. Old: %v, New %v.\n", numUniqueChunks, len(newPodsWithChunks))
		if force_bakeDeletion {
			fmt.Print(msg)
		} else {
			log.Fatal(msg)
		}
	}

	// Rule 3: Pods can not get new chunks
	for key, val := range newChunksInPods {
		for i := 0; i < len(val); i++ {
			if !containsStr(global_chunksInPod[key], val[i]) {
				var msg = fmt.Sprintf("Pod %v has gotten an unknown chunk: %v \n", key, val[i])
				if force_bakeDeletion {
					fmt.Print(msg)
				} else {
					log.Fatal(msg)
				}
			}
		}
	}

	// Rule 4: Replication factor for each chunk must be as specified.
	for key := range newPodsWithChunks {
		if len(newPodsWithChunks[key]) != repList[key] {
			var msg = fmt.Sprintf("Replication for chunk: %v, was not %v.\n", key, repList[key])
			if force_bakeDeletion {
				fmt.Print(msg)
			} else {
				log.Fatal(msg)
			}
		}
	}
	return
}

// global_chunksInPod and newChunksInPods and chunkDeletionMap has Key: PodId, Value: Chunks
// global_podsWithChunks and newPodsWithChunks has Key: Chunk, Value: PodIds
func getMostReplicatedUnassChunk(sortedChunkList []string, prefix string, repList map[string]int, newPodsWithChunks map[string][]string) (chunk string, err error) {
	fmt.Printf("Got prefix %v \n", prefix)
	for i := 0; i < len(sortedChunkList); i++ {
		podlist, ok := newPodsWithChunks[sortedChunkList[i]]
		if (!ok || len(podlist) < repList[sortedChunkList[i]]) && strings.HasPrefix(sortedChunkList[i], prefix) {
			return sortedChunkList[i], nil
		}
	}

	return "", errors.New("No chunks can be forced.")
}

func iterChunkCounter(item shed.Item) (stop bool, err error) {
	str := fmt.Sprintf("%064x", item.Address)

	if shouldFilterChunk(str) {
		return false, nil
	}

	chunklist, ok := global_chunksInPod[global_currPod]
	if !ok {
		global_chunksInPod[global_currPod] = make([]string, 0)
	}
	global_chunksInPod[global_currPod] = append(chunklist, str)

	podlist, ok := global_podsWithChunks[str]
	if !ok {
		global_podsWithChunks[str] = make([]string, 0)
	}
	global_podsWithChunks[str] = append(podlist, global_currPod)

	if v, ok := global_ChunkCount[str]; ok {
		global_ChunkCount[str] = v + 1
	} else {
		global_ChunkCount[str] = 1
	}

	global_currChunkCnt++
	return false, nil
}

func initChunkRemoval(files []string) {
	for i := 0; i < len(files); i++ {
		f, err := os.Open(files[i])
		if err != nil {
			continue
		}
		defer f.Close()

		byteArr, err := ioutil.ReadAll(f)
		if err != nil {
			continue
		}
		output := make(map[string]*chunkInfo)
		err = json.Unmarshal(byteArr, &output)
		if err != nil {
			continue
		}

		global_chunkFilters[files[i]] = output
	}
}

func deleteChunks_localhost() {
	pod := "localhost"
	global_currPod = pod

	podPath := input_localChunkDir + "/localhost"

	db, err := shed.NewDB(podPath, "racin")

	if err != nil {
		log.Error(err)
	}
	defer db.Close()

	global_currChunkCnt = 0

	i, err := db.NewIndex("Address->StoreTimestamp|BinID|Data", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			return fields.Address, nil
		},
		DecodeKey: func(key []byte) (g shed.Item, err error) {
			str := fmt.Sprintf("%064x", key)
			if str == "fffff7a144713b39e969fab084e4dbef8b9ce296e2f94220344e28f16f0d4273" {
				fmt.Printf("FOUND IT DURING DECODING\n")
				if yes, err := db.Has(key); yes {
					fmt.Printf("Yes, it has the chunks.\n")
				} else if err != nil {
					fmt.Printf("Error doing has: %v\n", err)
				}
			}
			g.Address = key
			return g, nil
		},
		EncodeValue: encodeValueFunc,
		DecodeValue: decodeValueFunc,
	})

	// if err != nil {
	// 	log.Error(err)
	// }

	// if verbose_ListChunks {
	// 	fmt.Printf("LISTING every %vth chunk addr FOR NODE %v\n", listChunkFreq, pod)
	// }
	err = i.Iterate(deleteItemsFunc, nil)

	// Iterate DB
	cnt := 0
	iter := db.NewIterator()
	for iter.Next() {
		// 	key := iter.Key()
		// 	val := iter.Value()

		//fmt.Printf("Key: %v, Value: %v\n", len(key), val[:10])
		cnt++
	}
	fmt.Printf("Total count: %v\n", cnt)
	// if err != nil {
	// 	log.Error(err)
	// }
	icount1, _ := i.Count()
	fmt.Printf("22 Count index: %v\n", icount1)
	for _, val := range global_deleteList {
		str := fmt.Sprintf("%064x", val)

		byteAddr, err := hexutil.Decode("0x" + str)
		fmt.Printf("DelList: %v, byteAddr: %v\n", val, byteAddr)
		if err != nil {
			fmt.Printf("Error during decode: %v\n", err)
		}
		key := shed.Item{
			Address: byteAddr,
		}
		if yes, _ := db.Has(byteAddr); yes {
			fmt.Printf("DB has chunk: %064x\n", val)
		}

		if yes, _ := i.Has(key); yes {
			fmt.Printf("Index has chunk: %064x\n", val)
		}
		fmt.Printf("Deleting... %064x , val: %v\n", val, val)

		err = db.Delete(byteAddr)

		if err != nil {
			fmt.Printf("Error deleteting: %064x\n", err)
		}
		err = i.Delete(key)
		if err != nil {
			fmt.Printf("Error deleteting: %064x\n", err)
		}
	}
	icount2, _ := i.Count()
	fmt.Printf("22 Count index: %v\n", icount2)
	//err = db.Delete([]byte("feee210c901582345ac87b0a2092ff0166e8243c89b45585145a104fc5e55d6c"))

	// for key, _ := range global_deleteList {
	// 	db.Delete([]byte(key))
	// }

	if verbose_ListChunks {
		fmt.Printf("TOTAL CHUNKS: %v\n", global_currChunkCnt)
	}
}

// global_chunksInPod and newChunksInPods has Key: PodId, Value: Chunks
func serialize(newChunksInPods map[string][]string) ([]byte, error) {
	return json.Marshal(newChunksInPods)
}

func deserialize(data []byte) (map[string][]string, error) {
	output := make(map[string][]string)
	err := json.Unmarshal(data, &output)

	return output, err
}
