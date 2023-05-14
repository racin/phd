package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/racin/phd/code/evaluation/bee"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	_ "net/http/pprof"
)

var (
	errWantInteractive = errors.New("want interactive mode")
)

type errBatch struct {
	batch string
}

func (errBatch) Error() string {
	return "want batch mode"
}

var (
	kubeconfigPath string
	namespace      string
	nameFormat     string
	prometheusAddr string
	uploaderID     int
	apiPort        int
	debugPort      int
	numBees        int
	connRetries    int
	parallel       int
	retryInterval  time.Duration
	timeout        time.Duration

	exitCode int

	ctl       bee.Controller
	conn      *bee.Connector
	promConn  *bee.Connector
	prom      v1.API
	cancelCtx context.CancelFunc
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "evaluation",
	Short: "Runs tests and benchmarks on a cluster of Swarm Bees",
	Long: `This program executes tests and benchmarks
on a cluster of Swarm bees deployed with Kubernetes.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if timeout != 0 {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			cmd.SetContext(ctx)
			cancelCtx = cancel
		}
		profile, err := cmd.Flags().GetString("profile")
		if err != nil {
			return err
		}
		if profile != "" {
			setUpProfiling(profile)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		batch, err := cmd.Flags().GetString("batch")
		if err != nil {
			return err
		}
		if batch != "" {
			return errBatch{batch}
		}
		interactive, err := cmd.Flags().GetBool("interactive")
		if err != nil {
			return err
		}
		if interactive {
			return errWantInteractive
		}
		return cmd.Usage()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if cancelCtx != nil {
			cancelCtx()
		}
	},
}

func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error)

	go func() {
		var errBatch errBatch
		err := rootCmd.ExecuteContext(ctx)
		if errors.Is(err, errWantInteractive) {
			Repl(ctx)
			close(errCh)
		} else if errors.As(err, &errBatch) {
			Batch(ctx, errBatch.batch, experimentRetries)
			close(errCh)
		} else {
			errCh <- err
		}
	}()

handle:
	select {
	case signal := <-signalCh:
		if ctx.Err() == nil {
			log.Printf("signal: %s", signal)
			os.Stdin.Close() // unblock repl
			cancel()
			goto handle // to wait for error on errCh
		} else {
			// already canceled, this is the second signal
			log.Fatalln("forcefully terminating...")
			exitCode = 1
		}
	case err := <-errCh:
		if ctl != nil {
			util.SafeClose(&err, ctl)
		}

		if err != nil {
			printError(err)
			os.Exit(1)
		}
	}

	os.Exit(exitCode)
}

func init() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().BoolP("interactive", "i", false, "enable interactive mode")
	rootCmd.PersistentFlags().String("profile", "", "enable pprof on the given address")
	rootCmd.PersistentFlags().String("batch", "", "run a batch of commands from a file")

	rootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", filepath.Join(home, ".kube", "config"), "path to kubeconfig")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "snarlbee", "kubernetes namespace")
	rootCmd.PersistentFlags().StringVar(&nameFormat, "name-format", "pod-snarlbee-%d", "format string that gives a pod name from an id")
	rootCmd.PersistentFlags().StringVar(&prometheusAddr, "prometheus-address", "http://localhost:9090", "address of prometheus server")
	rootCmd.PersistentFlags().IntVar(&uploaderID, "uploader-id", 1, "The ID of the bee to use for uploads.")
	rootCmd.PersistentFlags().IntVar(&apiPort, "api-port", 1633, "api port number")
	rootCmd.PersistentFlags().IntVar(&debugPort, "debug-port", 1635, "debug api port number")
	rootCmd.PersistentFlags().IntVar(&numBees, "bees", 10, "number of bees to connect to")
	rootCmd.PersistentFlags().IntVar(&parallel, "parallel", runtime.NumCPU(), "number of parallel operations permitted")
	rootCmd.PersistentFlags().IntVar(&connRetries, "conn-retries", 10, "number of times to retry connecitons")
	rootCmd.PersistentFlags().DurationVar(&retryInterval, "conn-retry-interval", 100*time.Millisecond, "time interval between connection attempts")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 0, "timeout before aborting")
}

func getConfig() (*kubernetes.Clientset, *rest.Config, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build kubernetes configuration: %w", err)
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return kubeclient, kubeconfig, err
}
func getConnector() (*bee.Connector, error) {
	if conn != nil {
		return conn, nil
	}
	kubeclient, kubeconfig, err := getConfig()
	if err != nil {
		return nil, err
	}
	c := bee.NewConnector(kubeclient, kubeconfig, namespace, "pod-snarlbee-%d", 1633, 1635)
	conn = &c
	return conn, nil
}

func getPromConnector() (*bee.Connector, error) {
	if promConn != nil {
		return promConn, nil
	}
	kubeclient, kubeconfig, err := getConfig()
	if err != nil {
		return nil, err
	}
	c := bee.NewConnector(kubeclient, kubeconfig, "monitoring", "prometheus-k8s-%d", 9090, 9090)
	promConn = &c
	return promConn, nil
}

func getController(ctx context.Context) (bee.Controller, error) {
	if ctl != nil {
		if ctl.BaseLen() >= numBees {
			return ctl.Sub(ctl.IDs()[:numBees]), nil
		}
		err := ctl.Close()
		if err != nil {
			return nil, err
		}
	}
	conn, err := getConnector()
	if err != nil {
		return nil, err
	}
	ctl, err = bee.NewController(ctx, *conn, numBees, parallel, connRetries, retryInterval)
	if err != nil {
		return nil, err
	}
	return ctl.Sub(ctl.IDs()[:numBees]), nil
}

func getPrometheus(address string) (v1.API, error) {
	if prom != nil {
		return prom, nil
	}
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, err
	}
	prom = v1.NewAPI(client)
	return prom, nil
}

func printError(err error) {
	errors := multierr.Errors(err)
	fmt.Print("Error")
	if len(errors) > 1 {
		fmt.Println("s: ")
	} else {
		fmt.Print(": ")
	}
	for _, err := range errors {
		fmt.Println("\t", err)
	}
}

var profileOnce sync.Once

func setUpProfiling(addr string) {
	profileOnce.Do(func() {
		runtime.SetBlockProfileRate(1)
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Println(err)
		}
		log.Printf("profilers listening on http://%s/debug/pprof", lis.Addr().String())
		go func() {
			log.Println(http.Serve(lis, nil))
		}()
	})
}
