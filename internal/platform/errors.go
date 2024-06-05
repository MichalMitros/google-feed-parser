package platform

import (
	"errors"
)

// ErrAlreadyRunning is an error returned when run can't be started because previous run is not finished yet.
var ErrAlreadyRunning = errors.New("parsing already running for this shop")
