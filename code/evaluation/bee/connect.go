package bee

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type Connector struct {
	kubeclient *kubernetes.Clientset
	kubeconfig *rest.Config
	namespace  string
	nameFormat string
	apiPort    int
	debugPort  int
}

func NewConnector(kubeclient *kubernetes.Clientset, kubeconfig *rest.Config, namespace string, nameformat string, apiPort, debugPort int) Connector {
	return Connector{
		kubeclient: kubeclient,
		kubeconfig: kubeconfig,
		namespace:  namespace,
		nameFormat: nameformat,
		apiPort:    apiPort,
		debugPort:  debugPort,
	}
}

func (b *Connector) ConnectAPI(ctx context.Context, bee int) (host string, disconnect func() error, err error) {
	return b.connectPort(ctx, bee, b.apiPort)
}

func (b *Connector) ConnectDebug(ctx context.Context, bee int) (host string, disconnect func() error, err error) {
	return b.connectPort(ctx, bee, b.debugPort)
}

// ConnectDebug returns a http client that can be used to debug the bee.
func (b *Connector) connectPort(ctx context.Context, beeID, svcPort int) (host string, disconnect func() error, err error) {
	port, err := getFreePort(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get a free port: %w", err)
	}

	req := b.kubeclient.RESTClient().
		Post().
		Prefix("api", "v1").
		Resource("pods").
		Namespace(b.namespace).
		Name(fmt.Sprintf(b.nameFormat, beeID)).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(b.kubeconfig)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create spdy roundtripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	errChan := make(chan error, 1)

	pf, err := portforward.New(
		dialer,
		[]string{fmt.Sprintf("%d:%d", port, svcPort)},
		stopChan,
		readyChan,
		io.Discard,
		io.Discard,
	)

	if err != nil {
		return "", nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	go func() { errChan <- pf.ForwardPorts() }()

	select {
	case <-ctx.Done():
		close(stopChan)
		return "", nil, ctx.Err()
	case <-readyChan:
	case err = <-errChan:
		return "", nil, fmt.Errorf("failed to start port forwarder: %w", err)
	}

	disconnect = func() error {
		select {
		// check that channel is not already closed
		case <-stopChan:
			return nil
		default:
			close(stopChan)
			return <-errChan
		}
	}

	return fmt.Sprintf("localhost:%d", port), disconnect, nil
}

func getFreePort(ctx context.Context) (port int, err error) {
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() {
		if cerr := lis.Close(); cerr != nil {
			port, err = 0, cerr
		}
	}()
	return lis.Addr().(*net.TCPAddr).Port, nil
}
