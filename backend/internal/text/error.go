package text

import (
	"errors"

	"bjoernblessin.de/go-utils/util/assert"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
)

// Error carries a statement up the stack as a Go error,
// so a failure reaches the contract as a code and the words of a third party behind it.
type Error struct {
	Text *screensharev1.Text
	// Words of a third party, verbatim: an OS error, a child's stderr line. Empty where Text says it all.
	Words string
	cause error
}

// Error spells the code and the words, for a log line.
func (e *Error) Error() string {
	if e.Words == "" {
		return e.Text.GetCode().String()
	}
	return e.Text.GetCode().String() + ": " + e.Words
}

// Unwrap keeps errors.Is reaching the third party's error, fs.ErrNotExist under an unreadable store.
func (e *Error) Unwrap() error { return e.cause }

// Fail states a failure this app names in full.
func Fail(code screensharev1.TextCode, args ...*screensharev1.TextArg) error {
	return &Error{Text: Of(code, args...)}
}

// Because states a failure with an error behind it.
// A statement in the cause's chain nests under TEXT_ARG_NAME_CAUSE,
// and the words beside it travel up unchanged.
func Because(cause error, code screensharev1.TextCode, args ...*screensharev1.TextArg) error {
	assert.IsNotNil(cause, "a failure stated because of an error has one", code.String())

	reason, words := Statement(cause)
	args = append(args, Nested(screensharev1.TextArgName_TEXT_ARG_NAME_CAUSE, reason))
	return &Error{Text: Of(code, args...), Words: words, cause: cause}
}

// Statement reads the outermost statement off an error chain, and the words beside it.
// An error carrying none is all words.
func Statement(err error) (*screensharev1.Text, string) {
	assert.IsNotNil(err, "a statement is read off a failure")

	var e *Error
	if errors.As(err, &e) {
		return e.Text, e.Words
	}
	return nil, err.Error()
}
