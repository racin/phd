package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethersphere/swarm/shed"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
	"github.com/zloylos/grsync"
)

// "github.com/ethereum/go-ethereum/common"
// "github.com/ethersphere/swarm/chunk"
// "github.com/ethersphere/swarm/storage/localstore"

// Swarmtool Listchunks lists all the chunks stored in the pods.

// Setup HTTP API to listen for requests
const defaultLDBCapacity = 5000000 // capacity for LevelDB, by default 5*10^6*4096 bytes == 20GB
var mutex = &sync.Mutex{}

var input_searchForChunks, input_localChunkDir, input_chunkFilter, input_podFilter, input_verifyPersistence string
var global_chunkFilters map[string]map[string]*chunkInfo = make(map[string]map[string]*chunkInfo)
var global_verifyPersist map[string]map[string]*chunkInfo = make(map[string]map[string]*chunkInfo)
var global_podFilter map[int]struct{} = make(map[int]struct{})

// podsWithChunks: Key: ChunkHash, Value: Array with PodId
var global_podsWithChunks map[string][]string = make(map[string][]string)

var global_ChunkCount map[string]int = make(map[string]int)
var global_searchChunks []string
var global_currPod string
var global_currChunkCnt int

var listChunkFreq int
var verbose_ListChunks, verbose_ListPods, only_Localhost, do_Copychunks bool

var listchunksCmd = &cobra.Command{
	Use:   "listchunks [numpeers]",
	Short: "List all chunks stored in the peers",
	Long:  "List all chunks stored in the peers. Accepts multiple JSON-encoded files to filter the listing.",
	Run: func(cmd *cobra.Command, args []string) {
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

		global_searchChunks = strings.Split(input_searchForChunks, ",")

		if input_chunkFilter != "" {
			initChunkFilter(strings.Split(input_chunkFilter, ","))
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
			countChunks_localhost()
		} else {
			if input_podFilter != "" {
				initPodFilter(strings.Split(input_podFilter, ","), numpeers)
			}
			if do_Copychunks {
				if err = copyChunkDBFromPods(numpeers); err != nil {
					log.Error(err)
				}
			}
			countChunks_pods(numpeers)
		}
		countChunks()
	},
}

type chunkInfo struct {
	Index int `json:"index"`
	Depth int `json:"depth"`
}

func init() {
	listchunksCmd.Flags().StringVarP(&input_searchForChunks, "podswithchunks", "", "", "List pods that has this particular chunks(s) [comma separated]")
	listchunksCmd.Flags().StringVarP(&input_localChunkDir, "localchunkdir", "", "podChunks/", "Local chunk directory")
	listchunksCmd.Flags().StringVarP(&input_podFilter, "podFilter", "", "", "Accepts a file with space-seperated pod IDs (See ONLINENODES). Only copy chunks from these nodes.")
	listchunksCmd.Flags().StringVarP(&input_chunkFilter, "filters", "", "", "JSON-encoded file(s) to filter the chunks listed [comma separated]")
	listchunksCmd.Flags().StringVarP(&input_verifyPersistence, "verifypersistence", "", "", "JSON-encoded file(s) to verify the persistence in the network [comma separated]")
	listchunksCmd.Flags().IntVarP(&listChunkFreq, "frequency", "f", 1000, "Frequency of listing a pods")
	listchunksCmd.Flags().BoolVarP(&verbose_ListChunks, "verboselistchunks", "", true, "Print the replication factor for every chunk.")
	listchunksCmd.Flags().BoolVarP(&verbose_ListPods, "verboselistpods", "", false, "Print chunk replication.")
	listchunksCmd.Flags().BoolVarP(&only_Localhost, "localhost", "", false, "Check chunks stored on localhost (On this very machine!)")
	listchunksCmd.Flags().BoolVarP(&fullPath, "fullPath", "", false, "True: Chunks stored in podChunks/pv-X-backup/swarm/bzz-*/chunks/, False: Chunks stored in podChunks/swarm-private-X/")
	listchunksCmd.Flags().BoolVarP(&do_Copychunks, "copychunks", "", true, "Copy the chunks from the pods (or localhost)")

	rootCmd.AddCommand(listchunksCmd)
}

func countChunks_localhost() {
	pod := "localhost"
	global_currPod = pod

	podPath := input_localChunkDir + "/localhost"

	db, err := shed.NewDB(podPath, "racin")
	if err != nil {
		log.Error(err)
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

	if verbose_ListChunks {
		fmt.Printf("LISTING every %vth chunk addr FOR NODE %v\n", listChunkFreq, pod)
	}
	err = i.Iterate(iterFunc, nil)

	db.Close()
	if err != nil {
		log.Error(err)
	}

	if verbose_ListChunks {
		fmt.Printf("TOTAL CHUNKS: %v\n", global_currChunkCnt)
	}
}

func countChunks_pods(numPods int) {
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

		if verbose_ListChunks {
			fmt.Printf("LISTING every %vth chunk addr FOR NODE %v\n", listChunkFreq, pod)
		}
		err = i.Iterate(iterFunc, nil)

		db.Close()
		if err != nil {
			log.Error(err)
		}

		if verbose_ListChunks {
			fmt.Printf("TOTAL CHUNKS: %v\n", global_currChunkCnt)
		}
	}
}

func countChunks() {
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

	for key, val := range global_podsWithChunks {
		fmt.Printf("Pods with chunk: %v - [[ %v ]]\n", key, val)
	}

	fmt.Printf("Total chunks found: %v\nUnique chunks found: %v\n", totalChunks, len(global_ChunkCount))
	fmt.Printf("Average replication factor: %v\n", float64(totalChunks)/float64(len(global_ChunkCount)))
}

func decodeValueFunc(keyItem shed.Item, value []byte) (e shed.Item, err error) {
	e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[8:16]))
	e.BinID = binary.BigEndian.Uint64(value[:8])
	e.Data = value[16:]
	return e, nil
}

func encodeValueFunc(fields shed.Item) (value []byte, err error) {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[:8], fields.BinID)
	binary.BigEndian.PutUint64(b[8:16], uint64(fields.StoreTimestamp))
	value = append(b, fields.Data...)
	return value, nil
}

func iterFunc(item shed.Item) (stop bool, err error) {
	str := fmt.Sprintf("%064x", item.Address)

	if shouldFilterChunk(str) {
		return false, nil
	}

	if containsStr(global_searchChunks, str) {
		podlist, ok := global_podsWithChunks[str]
		if !ok {
			global_podsWithChunks[str] = make([]string, 0)
		}
		global_podsWithChunks[str] = append(podlist, global_currPod)
	}
	if v, ok := global_ChunkCount[str]; ok {
		global_ChunkCount[str] = v + 1
	} else {
		global_ChunkCount[str] = 1
	}
	if global_currChunkCnt%listChunkFreq == 0 {
		if verbose_ListChunks {
			fmt.Printf("Item addr: %v, Item val: %v\n", str, item.Data[:Min(16, len(item.Data))])
		}
	}
	global_currChunkCnt++
	return false, nil
}

func copyChunkDBFromLocalhost() error {
	if _, err := os.Stat(input_localChunkDir); err != nil {
		if os.IsNotExist(err) {
			// Create the direcotry
			if err := os.Mkdir(input_localChunkDir, 0775); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	localSwarm := "/Users/racin/Library/Ethereum/swarm/"
	if _, err := os.Stat(localSwarm); err != nil {
		fmt.Printf("ERR: %v\n", err.Error())
		localSwarm = os.Getenv("HOME") + "/.ethereum/swarm/"
		if _, err := os.Stat(localSwarm); err != nil {
			log.Error(err)
		}
	}

	if matches, err := filepath.Glob(localSwarm + "bzz*/chunks"); len(matches) == 0 || err != nil {
		return errors.New("Did not find chunks. " + err.Error())
	} else {
		localSwarm = matches[0]
	}

	fmt.Printf("Found chunks: %v\n", localSwarm)

	podPath := input_localChunkDir + "localhost"

	if err := os.RemoveAll(podPath); err != nil {
		fmt.Printf("ERROR REMOVING DIR: %v\n", err.Error())
	}
	if err := os.Mkdir(podPath, 0775); err != nil {
		fmt.Printf("Cannot create chunk dir %v \n", err)
		return err
	}
	cmd := exec.Command("sh", "-c", "cp -R "+localSwarm+"/* "+podPath)

	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Printf("Copied chunks for pod: localhost.\n")
	cmd = exec.Command("sh", "-c", "chmod -R 755 "+input_localChunkDir)
	cmd.Run()
	fmt.Println("Copied all chunks.")
	return nil
}

func copyChunkDBFromSinglePod(pod string) error {
	podPath := input_localChunkDir + pod
	err := os.RemoveAll(podPath)
	if err != nil {
		return fmt.Errorf("ERROR REMOVING DIR: %v\n", err.Error())
	}
	if err := os.Mkdir(podPath, 0775); err != nil {
		return fmt.Errorf("Cannot create chunk dir %v \n", err)
	}
	cmd := exec.Command("sh", "-c", "kubectl --insecure-skip-tls-verify=true exec -n \"swarm\" \""+pod+"\" -- \"/bin/sh\" "+
		"\"-c\" \"tar cf - root/.ethereum/swarm/bzz-*/chunks\" "+
		"| tar -xf - -C "+podPath+" --strip-components 5")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ERROR RUNNING CMD %v: %v\n", pod, err.Error())
	}

	return nil
}

func copyChunkDBFromPods(numPods int) error {
	if _, err := os.Stat(input_localChunkDir); err != nil {
		if os.IsNotExist(err) {
			// Create the direcotry
			if err := os.Mkdir(input_localChunkDir, 0775); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Rate limied to 10 because we have 28 nodes with storage and 28 seems to be too high(?)
	rl := NewRatelimiter(10)
	wg := &sync.WaitGroup{}
	for i := 0; i < numPods; i++ {
		if _, ok := global_podFilter[i]; ok {
			continue
		}
		wg.Add(1)
		rl.Acquire()
		var pod string
		if fullPath {
			pod = "pv-" + strconv.Itoa(i) + "-backup/swarm/bzz-*/chunks"
		} else {
			pod = "swarm-private-" + strconv.Itoa(i)
		}
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

		err := os.RemoveAll(podPath)
		if err != nil {
			fmt.Printf("ERROR REMOVING DIR: %v\n", err.Error())
		}
		if err := os.Mkdir(podPath, 0775); err != nil {
			fmt.Printf("Cannot create chunk dir %v \n", err)
			rl.Release()
			wg.Done()
			continue
		}
		cmd := exec.Command("sh", "-c", "kubectl --insecure-skip-tls-verify=true exec -n \"swarm\" \""+pod+"\" -- \"/bin/sh\" "+
			"\"-c\" \"tar cf - root/.ethereum/swarm/bzz-*/chunks\" "+
			"| tar -xf - -C "+podPath+" --strip-components 5")
		go func() {
			err := cmd.Run()
			if err != nil {
				fmt.Printf("ERROR RUNNING CMD %v: %v\n", pod, err.Error())
			}

			fmt.Printf("Copied chunks for pod: %v. CMD: %v\n", pod, "kubectl cp swarm/"+pod+":root/.ethereum/swarm/ "+input_localChunkDir+pod)
			wg.Done()
			rl.Release()
		}()
	}

	wg.Wait()
	cmd := exec.Command("sh", "-c", "chmod -R 755 "+input_localChunkDir)
	cmd.Run()
	fmt.Println("Copied all chunks.")
	return nil
}

// Setup rsync to copy chunkDB (Find it too?)
func syncLocalDB(src, dest string) {
	mutex.Lock()
	defer mutex.Unlock()

	// TODO - Gracefully flush leveldb to persistant storage. How ??

	task := grsync.NewTask(src, dest,
		grsync.RsyncOptions{Recursive: true, Update: false,
			Existing: false, IgnoreExisting: false, Delete: true, Verbose: true, Quiet: false},
	)

	err := task.Run()
	var retry int = 10
	for err != nil && retry > 0 {
		time.Sleep(10 * time.Millisecond)
		//fmt.Printf("Error! %+v\n", err)
		err = task.Run()
		retry--
	}

	// if err != nil {
	// 	log.Fatalf("Error rsync. %+v", err)
	// }

	// // Wait until synching finishes.
	// for task.State().Remain != 0 {
	// 	time.Sleep(10 * time.Millisecond)
	// }
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type sortedMap struct {
	m map[string]int
	s []string
}

func (sm *sortedMap) Len() int {
	return len(sm.m)
}

func (sm *sortedMap) Less(i, j int) bool {
	return sm.m[sm.s[i]] > sm.m[sm.s[j]]
}

func (sm *sortedMap) Swap(i, j int) {
	sm.s[i], sm.s[j] = sm.s[j], sm.s[i]
}

func sortedKeys(m map[string]int) []string {
	sm := new(sortedMap)
	sm.m = m
	sm.s = make([]string, len(m))
	i := 0
	for key := range m {
		sm.s[i] = key
		i++
	}
	sort.Sort(sm)
	return sm.s
}

func containsStr(haystack []string, needle string) bool {
	if haystack == nil {
		return false
	}
	for _, val := range haystack {
		if val == needle {
			return true
		}
	}
	return false
}

// initChunkFilter generates global_chunkFilters (map[string]map[string]*chunkInfo)
// Key: File name, Value: Chunks
func initChunkFilter(files []string) {
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

func initPodFilter(files []string, numpeers int) {
	// Reversefilter: All the pods that we WANT
	reverseFilter := make(map[int]struct{})
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

		pods := strings.Split(strings.ReplaceAll(string(byteArr), "\n", ""), " ")
		for j := 0; j < len(pods); j++ {
			if podNum, err := strconv.Atoi(pods[j]); err == nil {
				reverseFilter[podNum] = struct{}{}
			} else {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}

	// Global_podfilter: All the pods that we do NOT want
	if len(reverseFilter) > 0 {
		for i := 0; i < numpeers; i++ {
			if _, ok := reverseFilter[i]; !ok {
				global_podFilter[i] = struct{}{}
			}
		}
	}
}

func initVerifyPersist(files []string) {
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

		global_verifyPersist[files[i]] = output
	}
}

func shouldFilterChunk(chunk string) bool {
	if len(global_chunkFilters) == 0 {
		return false
	}
	for _, filter := range global_chunkFilters {
		if _, ok := filter[chunk]; ok {
			return false
		}
	}
	return true
}
