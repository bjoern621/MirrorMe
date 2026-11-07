// Package text builds the statements the backend makes about a configuration.
//
// A statement is a code and the identifiers it is about, never a sentence
// (api/proto/mirrorme/v1/text.proto).
// Wording, clause order and the short form against the long one are the shell's,
// so a rule that greys a control states what is true and nothing about how it reads.
//
// A failure crosses as a statement with a third party's words beside it, raw:
// Refusal.message, ExitInfo.message, RelayStatus.error.
// Those words and Summary.command are shown verbatim and never matched against.
//
// Every constructor asserts what the value it carries has to be.
// An empty identifier or a placeholder figure renders as a sentence with a hole in it,
// on screen indistinguishable from a control greyed for no reason at all.
// Both are Entwicklungsfehler, so both fail here,
// rather than on the surface that cannot tell which happened.
package text

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
)

// Of is one statement: which one, and what it is about.
// The code is asserted rather than defaulted, the zero value meaning "not set" on every enum here.
func Of(code mirrormev1.TextCode, args ...*mirrormev1.TextArg) *mirrormev1.Text {
	assert.Assert(code != mirrormev1.TextCode_TEXT_CODE_UNSPECIFIED,
		"a statement names which statement it is")

	out := &mirrormev1.Text{Code: code}
	for _, a := range args {
		// nil argument: an optional clause the caller had nothing for.
		// Dropped here, so a builder passes the same argument list either way.
		if a == nil {
			continue
		}
		out.Args = append(out.Args, a)
	}

	assert.Assert(out.Code != mirrormev1.TextCode_TEXT_CODE_UNSPECIFIED,
		"an assembled statement still names which statement it is", len(out.Args))
	return out
}

// ID carries one identifier of the axis its name declares.
// An empty identifier is dropped, so a caller with none states the shorter form of the same
// statement.
func ID(name mirrormev1.TextArgName, id string) *mirrormev1.TextArg {
	assert.Assert(name != mirrormev1.TextArgName_TEXT_ARG_NAME_UNSPECIFIED,
		"an argument names which substitution it fills")

	if id == "" {
		return nil
	}
	return &mirrormev1.TextArg{Name: name, Value: &mirrormev1.TextArg_Id{Id: id}}
}

// IDs carries several identifiers of one axis, in the order the domain table declares them.
// How they read together, the comma and the "or", is the shell's.
// An empty list is dropped, the same shorter form an empty identifier takes in ID.
func IDs(name mirrormev1.TextArgName, ids []string) *mirrormev1.TextArg {
	assert.Assert(name != mirrormev1.TextArgName_TEXT_ARG_NAME_UNSPECIFIED,
		"an argument names which substitution it fills")

	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		assert.Assert(id != "", "a listed identifier is one the domain declares", name.String())
	}
	return &mirrormev1.TextArg{
		Name:  name,
		Value: &mirrormev1.TextArg_Ids{Ids: &mirrormev1.IdList{Ids: ids}},
	}
}

// Num carries a whole figure in the unit its name declares.
// Zero is carried rather than dropped, being a measurement:
// a statement quoting no uplink reads as a line with no capacity, not as one nobody measured.
func Num(name mirrormev1.TextArgName, n int64) *mirrormev1.TextArg {
	assert.Assert(name != mirrormev1.TextArgName_TEXT_ARG_NAME_UNSPECIFIED,
		"an argument names which substitution it fills")
	return &mirrormev1.TextArg{Name: name, Value: &mirrormev1.TextArg_Number{Number: n}}
}

// Dec carries a fractional figure in the unit its name declares.
// Decimal places are the shell's, one figure serving a rounded chip and a spelled-out tooltip.
func Dec(name mirrormev1.TextArgName, v float64) *mirrormev1.TextArg {
	assert.Assert(name != mirrormev1.TextArgName_TEXT_ARG_NAME_UNSPECIFIED,
		"an argument names which substitution it fills")
	return &mirrormev1.TextArg{Name: name, Value: &mirrormev1.TextArg_Decimal{Decimal: v}}
}

// Nested carries a statement inside a statement,
// so a reason quotes the fact behind it with neither half knowing the other's wording.
// A nil statement is dropped, so a caller with nothing to quote gets the shorter form.
func Nested(name mirrormev1.TextArgName, t *mirrormev1.Text) *mirrormev1.TextArg {
	assert.Assert(name != mirrormev1.TextArgName_TEXT_ARG_NAME_UNSPECIFIED,
		"an argument names which substitution it fills")

	if t == nil {
		return nil
	}
	return &mirrormev1.TextArg{Name: name, Value: &mirrormev1.TextArg_Text{Text: t}}
}
