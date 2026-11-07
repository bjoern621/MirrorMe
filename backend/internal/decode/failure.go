package decode

import (
	"google.golang.org/protobuf/proto"

	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/text"
)

// failure is an error as it crosses from the host to the backend: its statement, marshalled,
// and its words, so the code survives the process boundary.
type failure struct {
	Statement []byte
	Words     string
}

func failureOf(err error) *failure {
	assert.IsNotNil(err, "a failure crosses for an error")

	statement, words := text.Statement(err)
	if statement == nil {
		return &failure{Words: words}
	}
	encoded, marshalErr := proto.Marshal(statement)
	assert.IsNil(marshalErr, "a statement built in this process marshals")
	return &failure{Statement: encoded, Words: words}
}

func (f *failure) err() error {
	if len(f.Statement) == 0 {
		return text.Restated(nil, f.Words)
	}
	statement := &mirrormev1.Text{}
	// Both ends are one build, so a statement that does not unmarshal is a broken host.
	assert.IsNil(proto.Unmarshal(f.Statement, statement), "a statement from the host unmarshals")
	return text.Restated(statement, f.Words)
}

// nothingDecoding is the refusal for a call naming a decode the host does not hold.
func nothingDecoding(id ID) error {
	switch id.Kind {
	case KindStream:
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_NOTHING_DECODING, text.Stream(id.Name), text.Transport(id.Transport))
	case KindMonitor:
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_MONITOR_NOT_PREVIEWING, text.Monitor(id.Monitor))
	case KindPreview:
		return text.Fail(mirrormev1.TextCode_TEXT_CODE_PREVIEW_NOT_RUNNING)
	default:
		assert.Never("a decode is one of the three kinds", id.Kind)
		return nil
	}
}
