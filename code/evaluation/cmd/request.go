package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

var (
	requestMethod string
	requestOutput string
	requestID     int
	requestAPI    bool
)

var requestCmd = &cobra.Command{
	Use:   "request path",
	Short: "send requests to a bee",
	Long:  `The request command sends http requests to a bee.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer resetRequestFlags(cmd)

		conn, err := getConnector()
		if err != nil {
			return err
		}

		var (
			host       string
			disconnect func() error
		)

		if requestAPI {
			host, disconnect, err = conn.ConnectAPI(cmd.Context(), requestID)
		} else {
			host, disconnect, err = conn.ConnectDebug(cmd.Context(), requestID)
		}
		if err != nil {
			return fmt.Errorf("failed to connect to bee %d: %w", requestID, err)
		}
		defer util.CheckErr(&err, disconnect)

		res, err := http.DefaultClient.Do(&http.Request{
			Method: requestMethod,
			URL:    &url.URL{Scheme: "http", Host: host, Path: args[0]},
		})
		if err != nil {
			return fmt.Errorf("error sending request to bee %d: %w", requestID, err)
		}
		defer util.SafeClose(&err, res.Body)
		err = util.CheckResponse(res)
		if err != nil {
			return fmt.Errorf("error sending request to bee %d: %w", requestID, err)
		}

		fmt.Printf("got response from bee %d: %s\n\n", requestID, res.Status)

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

		if strings.Contains(res.Header.Get("Content-Type"), "application/json") {
			err = formatJSON(output, res.Body)
		} else {
			_, err = io.Copy(output, res.Body)
		}
		if err != nil {
			return fmt.Errorf("failed to write response body to stdout: %w", err)
		}

		return nil
	},
}

func resetRequestFlags(cmd *cobra.Command) {
	cmd.ResetFlags()
	cmd.Flags().StringVarP(&requestMethod, "method", "m", "GET", "the http request method")
	cmd.Flags().StringVarP(&requestOutput, "output", "o", "", "File to write response body to (defaults to stdout)")
	cmd.Flags().IntVar(&requestID, "bee-id", 1, "the ID of the bee to send requests to")
	cmd.Flags().BoolVar(&requestAPI, "api", false, "use the normal API port instead of the debug API port")
}

func init() {
	rootCmd.AddCommand(requestCmd)
	resetRequestFlags(requestCmd)
}

// formatJSON tries to write the content of src to dst as formatted JSON.
func formatJSON(dst io.Writer, src io.Reader) (err error) {
	var val json.RawMessage

	dec := json.NewDecoder(src)
	err = dec.Decode(&val)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(dst)
	enc.SetIndent("", "\t")
	err = enc.Encode(val)
	if err != nil {
		return err
	}

	return nil
}
