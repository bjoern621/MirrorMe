package control

import (
	"errors"

	"bjoernblessin.de/go-utils/util/assert"
)

// A backend refusal the request earned, as against the machine's state.
//
// The contract (api/proto/screenshare/v1/control.proto) splits them:
// INVALID_ARGUMENT for a request naming something absent from this build or this machine,
// FAILED_PRECONDITION for one whose arguments are all real and whose moment is wrong.
// Both arrive here as an error from one method, so the type carries the difference.

// Refused marks an error as earned by the request:
// a monitor index no output is enumerated under, a transport this build has no viewer for.
// Everything else a backend returns is the world failing at something it was legal to ask for.
type Refused struct {
	cause error
}

// Refuse marks the backend's own failure, statement and all, as earned by the request.
func Refuse(err error) error {
	assert.IsNotNil(err, "a refusal says what it refused")

	return &Refused{cause: err}
}

func (e *Refused) Error() string { return e.cause.Error() }

func (e *Refused) Unwrap() error { return e.cause }

// refused reports whether err carries a Refused anywhere in its chain,
// so a backend wrapping one with context keeps the code it earned.
func refused(err error) bool {
	var r *Refused
	return errors.As(err, &r)
}
