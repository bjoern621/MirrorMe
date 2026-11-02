package control

import (
	"bjoernblessin.de/go-utils/util/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/text"
)

// The two error kinds cross this boundary differently,
// and the difference is the whole error model (docs/ipc-api.md, "Errors").
//
// An Umgebungsfehler, a condition the app must survive, is a gRPC status:
// expected, and the user's to see.
// Its one detail is a Refusal, a coded statement and a third party's words beside it,
// and its message spells the code for a reader of the raw status.
// These helpers are the only way one is produced here,
// so a code is chosen against the contract's table rather than per call site.
//
// An Entwicklungsfehler never crosses.
// assert panics in the backend, as everywhere else in this repository,
// and a shell that could receive a bug as a status would start handling bugs.
//
// An unreachable relay is no call failure:
// GetRelayStatus succeeds with a snapshot whose reachable is false,
// "the relay is down" being a thing the screen says rather than a thing the call failed at.

// refuse serves a stated failure (text.Fail, text.Because) under the status code given.
func refuse(c codes.Code, failure error) error {
	statement, words := text.Statement(failure)
	assert.IsNotNil(statement, "a served refusal states what it refused", failure)

	st, err := status.New(c, statement.GetCode().String()).
		WithDetails(&screensharev1.Refusal{Text: statement, Message: words})
	assert.Assert(err == nil, "a Refusal packs into a status detail", err)
	return st.Err()
}

// invalidArgument is a request naming something that cannot exist: an unknown transport,
// an empty stream name.
func invalidArgument(failure error) error { return refuse(codes.InvalidArgument, failure) }

// failedPrecondition is a valid request the world is not ready for:
// a different pipeline already publishing, nothing to apply settings to,
// a measurement while a stream is live.
func failedPrecondition(failure error) error { return refuse(codes.FailedPrecondition, failure) }

// unavailable is the world failing at something legal to ask for: a relay out of reach,
// a child process that would not start.
func unavailable(failure error) error { return refuse(codes.Unavailable, failure) }

// notFound is a preset, log or decode that the name does not reach.
func notFound(failure error) error { return refuse(codes.NotFound, failure) }

// resourceExhausted is a bounded resource over-asked: more test streams than this machine runs.
func resourceExhausted(failure error) error { return refuse(codes.ResourceExhausted, failure) }
