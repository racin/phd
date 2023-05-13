package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/racin/phd/code/evaluation/distribution"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

var rootaddrfilter string

var distributionCmd = &cobra.Command{
	Use:   "distribution chunk|node|all summary|raw|count",
	Short: "calculate chunk distribution",
	Long:  "The distribution command calculates how many times each chunk is replicated. Use the 'raw' argument to output the distribution.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[1] != "summary" && args[1] != "raw" && args[1] != "count" {
			fmt.Println("Invalid second parameter. Valid options: [ summary | raw | count ]")
			return nil
		} else if args[0] != "chunk" && args[0] != "node" && args[0] != "all" {
			fmt.Println("Invalid first parameter. Valid options: [ chunk | node | all ]")
			return nil
		}
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}
		var filterFunc func(string) bool
		if rootaddrfilter != "" {
			filterFunc, err = getRootAddrFilter(cmd.Context(), rootaddrfilter)
			if err != nil {
				return err
			}
		}
		cd, nd, err := distribution.GetChunkDistribution(cmd.Context(), ctl, nameFormat, filterFunc)
		if err != nil {
			return err
		}

		if len(cd) == 0 {
			return fmt.Errorf("no chunks stored")
		}

		var output io.Writer
		if requestOutput != "" {
			f, err := os.OpenFile(requestOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.FilePerm)
			if err != nil {
				return fmt.Errorf("failed to open output file: %w", err)
			}
			defer util.SafeClose(&err, f)
			output = f
		} else {
			output = os.Stdout
		}

		switch args[1] {
		case "raw":
			if args[0] == "chunk" {
				return util.PrintJson(output, cd)
			}
			if args[0] == "node" {
				return util.PrintJson(output, nd)
			}
			if args[0] == "all" {
				return util.PrintJson(output, cd, nd)
			}
		case "summary":
			if args[0] == "chunk" || args[0] == "all" {
				cdStats := util.CalcStats(cd)
				fmt.Fprintf(output, "Chunk stats. total: %d, unique: %d, min: %d, max: %d, median: %d mean: %f\n",
					cdStats.Total, cdStats.Unique, cdStats.Min, cdStats.Max, cdStats.Median, cdStats.Mean)
			}
			if args[0] == "node" || args[0] == "all" {
				ndStats := util.CalcStats(nd)
				fmt.Fprintf(output, "Node stats. total: %d, unique: %d, min: %d, max: %d, median: %d mean: %f\n",
					ndStats.Total, ndStats.Unique, ndStats.Min, ndStats.Max, ndStats.Median, ndStats.Mean)
			}
		case "count":
			if args[0] == "chunk" {
				return util.PrintJson(output, util.CountMap(cd))
			}
			if args[0] == "node" {
				return util.PrintJson(output, util.CountMap(nd))
			}
			if args[0] == "all" {
				return util.PrintJson(output, util.CountMap(cd), util.CountMap(nd))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(distributionCmd)
	distributionCmd.ResetFlags()
	distributionCmd.Flags().StringVarP(&rootaddrfilter, "rootaddrfilter", "r", "", "only consider chunks within the supplied trees. comma separated.")
	distributionCmd.Flags().StringVarP(&requestOutput, "output", "o", "", "File to write response body to (defaults to stdout)")
}

func getRootAddrFilter(ctx context.Context, rootsaddrs string) (_ func(string) bool, err error) {
	roots := strings.Split(rootsaddrs, ",")
	chunksToFilter := make(map[string]struct{})
	host, disconnect, err := conn.ConnectDebug(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to bee %d: %w", requestID, err)
	}
	defer util.CheckErr(&err, disconnect)

	for _, root := range roots {
		res, err := http.DefaultClient.Do(&http.Request{
			Method: "GET",
			URL:    &url.URL{Scheme: "http", Host: host, Path: "/debug/traverse/" + root},
		})
		if err != nil {
			return nil, fmt.Errorf("error sending request to bee %d: %w", requestID, err)
		}
		defer util.SafeClose(&err, res.Body)
		err = util.CheckResponse(res)
		if err != nil {
			return nil, fmt.Errorf("error sending request to bee %d: %w", requestID, err)
		}
		rd := bufio.NewReader(res.Body)
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("error reading traverse response. line: %v. %w", line, err)
			}
			stringAddr := strings.TrimSpace(line)
			chunksToFilter[stringAddr] = struct{}{}
		}
	}
	return func(a string) bool {
		if _, ok := chunksToFilter[a]; ok {
			return true
		}
		return false
	}, nil
}
