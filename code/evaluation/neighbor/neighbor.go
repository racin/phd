package neighbor

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/util"
)

type ReverseNeighborMap map[string][]string
type NeighborMap map[string][]addrResp

type addrResp struct {
	Overlay      string   `json:"overlay"`
	Underlay     []string `json:"underlay"`
	Ethereum     string   `json:"ethereum"`
	PublicKey    string   `json:"publicKey"`
	PSSPublicKey string   `json:"pssPublicKey"`
	HostName     string   `json:"hostName"`
}

func GetNeighborMap(ctx context.Context, ctl bee.Controller, beeID int) (NM NeighborMap, reverseNM ReverseNeighborMap, err error) {
	var mut sync.Mutex
	NM, reverseNM = make(NeighborMap), make(ReverseNeighborMap)
	err = ctl.Parallel(ctx, func(id int, host string) (err error) {
		res, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodGet,
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/neighbors"},
		})
		if err != nil {
			return fmt.Errorf("failed to request database contents from %d: %w", id, err)
		}
		defer util.SafeClose(&err, res.Body)
		err = util.CheckResponse(res)
		if err != nil {
			return err
		}

		mut.Lock()
		defer mut.Unlock()
		all, err := ioutil.ReadAll(res.Body)

		var nbors []addrResp
		err = json.Unmarshal(all, &nbors)
		if err != nil {
			return err
		}
		NM[fmt.Sprintf("pod-snarlbee-%d", id)] = nbors

		for neighbor := range nbors {
			reverseNM[nbors[neighbor].HostName] = append(reverseNM[nbors[neighbor].HostName], fmt.Sprintf("pod-snarlbee-%d", id))
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return NM, reverseNM, nil
}

func GetMyAddresses(ctx context.Context, ctl bee.Controller, beeID int) (addrResp, error) {
	host, disconnect, err := ctl.Connector().ConnectDebug(ctx, beeID)
	var myaddr addrResp
	if err != nil {
		return myaddr, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)
	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: host, Path: "/addresses"},
	})
	if err != nil {
		return myaddr, fmt.Errorf("failed to get neighbors from %d: %w", beeID, err)
	}
	defer util.SafeClose(&err, res.Body)
	all, err := ioutil.ReadAll(res.Body)

	err = json.Unmarshal(all, &myaddr)

	return myaddr, err
}

func GetMyNeighbors(ctx context.Context, ctl bee.Controller, beeID int) ([]addrResp, error) {
	host, disconnect, err := ctl.Connector().ConnectDebug(ctx, beeID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bee %d: %w", beeID, err)
	}
	defer util.CheckErr(&err, disconnect)
	res, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: host, Path: "/neighbors"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get neighbors from %d: %w", beeID, err)
	}
	defer util.SafeClose(&err, res.Body)
	all, err := ioutil.ReadAll(res.Body)

	var nbors []addrResp
	err = json.Unmarshal(all, &nbors)
	if err != nil {
		return nil, fmt.Errorf("Got response: %s. Error: %w", string(all), err)
	}

	return nbors, nil
}

func VerifyNeighborMaps(forwardMap NeighborMap, reverseMap ReverseNeighborMap) {
	for pod, neighbors := range forwardMap {
		for _, neighbor := range neighbors {
			if !util.Contains(reverseMap[neighbor.HostName], pod) {
				fmt.Printf("Pod %s has neighbor %s but %s does not have %s as a neighbor\n", pod, neighbor.HostName, neighbor.HostName, pod)
			}
		}
	}
}
