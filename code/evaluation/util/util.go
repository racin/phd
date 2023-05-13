package util

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const (
	DirPerm  = 0755 // default directory permissions
	FilePerm = 0644 // default file permissions
)

var (
	Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type Stats struct {
	Median int
	Min    int
	Max    int
	Total  int
	Unique int
	Mean   float64
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[Rand.Intn(len(letters))]
	}
	return string(s)
}

func SafeClose(errPtr *error, closer io.Closer) {
	CheckErr(errPtr, closer.Close)
}

func CheckErr(errPtr *error, f func() error) {
	if errPtr == nil {
		panic("error pointer is nil")
	}
	err := f()
	if *errPtr == nil {
		*errPtr = err
	}
}

func ReadJSONFile[T any](path string, out *T) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(f)
	err = decoder.Decode(out)
	if err != nil {
		return fmt.Errorf("failed to decode JSON file: %w", err)
	}

	return nil
}

func WriteJSONFile(path string, perm os.FileMode, jsonData any) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer SafeClose(&err, f)

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	return enc.Encode(jsonData)
}

type messageResponse struct {
	Message string `json:"message"`
}

func ReadJSONResponse(rd io.Reader) (string, error) {
	var msg messageResponse
	dec := json.NewDecoder(rd)
	err := dec.Decode(&msg)
	if err != nil {
		return "", err
	}
	return msg.Message, nil
}

func CheckResponse(res *http.Response) error {
	if res.StatusCode/100 == 2 {
		return nil
	}

	if strings.Contains(res.Header.Get("Content-Type"), "application/json") {
		msg, err := ReadJSONResponse(res.Body)
		if err == nil {
			return fmt.Errorf("%s: %s", res.Status, msg)
		}
	}

	return fmt.Errorf("%s", res.Status)
}

func PrintJson[K comparable, T any](dst io.Writer, themapz ...map[K]T) error {
	for _, themap := range themapz {
		jayzon, err := json.Marshal(themap)
		if err != nil {
			return err
		}
		fmt.Fprintf(dst, "\n%v", string(jayzon)) // keep newline to allow printing both.
	}
	return nil
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func CalcStats[K comparable, T any](themap map[K][]T) (s Stats) {
	cdCounted := CountMap(themap)
	vals := maps.Values(cdCounted)
	slices.Sort(vals)
	s.Median = vals[len(vals)/2]
	s.Total = 0
	s.Min = math.MaxInt
	s.Max = 0
	for _, n := range vals {
		if n > 0 {
			s.Unique++
		}
		if n > s.Max {
			s.Max = n
		}
		if n < s.Min {
			s.Min = n
		}
		s.Total += n
	}
	s.Mean = float64(s.Total) / float64(len(vals))
	return
}

func CountMap[K comparable, T any](themap map[K][]T) map[K]int {
	r := make(map[K]int)

	for key, val := range themap {
		r[key] = len(val)
	}

	return r
}
