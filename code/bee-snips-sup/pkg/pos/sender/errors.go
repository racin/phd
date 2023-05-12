package sender

import (
	"errors"
	"fmt"

	"github.com/ethersphere/bee/pkg/swarm"
)

var (
	ErrNotSent = errors.New("request not sent")
)

// Error describes an error that happened while communicating with a peer.
type Error struct {
	peer swarm.Address
	err  error
}

func (err Error) Peer() swarm.Address {
	return err.peer
}

func (err Error) Unwrap() error {
	return err.err
}

func (err Error) Error() string {
	return fmt.Sprintf("error communicating with %v: %v", err.peer, err.err)
}
