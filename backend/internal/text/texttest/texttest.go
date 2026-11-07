// Package texttest reads a failure's statement for a test, which asserts on codes and arguments
// and never on the words a failure spells for a log.
package texttest

import (
	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/text"
)

// Code is the code of a failure's outermost statement.
func Code(err error) mirrormev1.TextCode {
	statement, _ := text.Statement(err)
	return statement.GetCode()
}

// IDs is every identifier a failure's statement names, the listed ones included.
func IDs(err error) []string {
	statement, _ := text.Statement(err)
	var out []string
	for _, arg := range statement.GetArgs() {
		if arg.GetId() != "" {
			out = append(out, arg.GetId())
		}
		out = append(out, arg.GetIds().GetIds()...)
	}
	return out
}

// Nums is every number a failure's statement names.
func Nums(err error) []int64 {
	statement, _ := text.Statement(err)
	var out []int64
	for _, arg := range statement.GetArgs() {
		if n, ok := arg.GetValue().(*mirrormev1.TextArg_Number); ok {
			out = append(out, n.Number)
		}
	}
	return out
}
