package app

import (
	"strings"

	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"

	"bjoernblessin.de/screenshare/internal/text"
)

// exitFailure is what a state carries for a run that ended on err:
// the statement naming why, and the words beside it with the child's last lines after them.
// A membership refusal outranks the run's own statement,
// being why the relay closed the connection that the child reports losing.
func exitFailure(err error, tail string, membership *screensharev1.Text) (*screensharev1.Text, string) {
	assert.IsNotNil(err, "an exit failure is read off an error")

	statement, words := text.Statement(err)
	words = strings.Trim(words+"\n"+tail, "\n")
	if membership != nil {
		return membership, words
	}
	return statement, words
}
