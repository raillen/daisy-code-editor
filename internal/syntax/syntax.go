// Package syntax gives a document's bytes the roles the theme colours.
//
// The twenty roles are the `[syntax]` table of the configuration, and until this
// package existed nineteen of them had no reader at all: they were parsed,
// tested, and never consulted — a configuration that describes a product which
// does not exist. A role here is what one of those fields means; turning it into
// a colour is the theme's job and not this one's.
//
// Nothing here knows a terminal exists. The input is a lexer, the output is byte
// ranges, and the surface decides how to paint them — the same shape the search
// and the selection already have, where the model keeps bytes and the viewport
// paints cells.
package syntax

import "github.com/ori-team/oride/internal/config"

// Role is what a run of text means to the highlighter.
//
// The zero value is the absence of a role: text the lexer had nothing to say
// about, which is most of a document and is painted with the surface's own
// foreground.
type Role int

// The roles, in the order of the `[syntax]` table.
//
// The names and the order are the configuration's, deliberately: a reader
// comparing this list with the table in the documentation should find them
// identical, and a role added to one and not the other fails
// TestEveryRoleHasAColour.
const (
	None Role = iota
	Comment
	Keyword
	String
	Number
	TypeName
	Function
	Operator
	Punctuation
	Variable
	Constant
	Property
	Tag
	Attribute
	Heading
	Emphasis
	Strong
	Link
	Code
	ListMarker
	Quote
)

// roleNames is the `[syntax]` table, indexed by role.
//
// Indexed rather than mapped so the declaration above is the single source of
// truth: a role with no name here is a role nothing can name in a failure
// message or a test.
var roleNames = []string{
	Comment:     "Comment",
	Keyword:     "Keyword",
	String:      "String",
	Number:      "Number",
	TypeName:    "TypeName",
	Function:    "Function",
	Operator:    "Operator",
	Punctuation: "Punctuation",
	Variable:    "Variable",
	Constant:    "Constant",
	Property:    "Property",
	Tag:         "Tag",
	Attribute:   "Attribute",
	Heading:     "Heading",
	Emphasis:    "Emphasis",
	Strong:      "Strong",
	Link:        "Link",
	Code:        "Code",
	ListMarker:  "ListMarker",
	Quote:       "Quote",
}

// All lists every role a theme can colour, in the order of the table.
func All() []Role {
	out := make([]Role, 0, len(roleNames))
	for role := Comment; int(role) < len(roleNames); role++ {
		out = append(out, role)
	}
	return out
}

// String is the role as the configuration spells it.
//
// A Stringer rather than a Name because every failure message that names a role
// should be readable, and a role printed as its ordinal tells the reader nothing
// they could look up.
func (r Role) String() string {
	if !r.coloured() {
		return "None"
	}
	return roleNames[r]
}

// coloured reports whether the role is one the table declares.
func (r Role) coloured() bool { return int(r) > int(None) && int(r) < len(roleNames) }

// colours reads the configured colour of each role, indexed by role.
//
// Accessors rather than a map of names: the field is the thing being read, so a
// renamed field is a compile error instead of a colour that silently became
// empty — which is the failure this project already paid for once.
var colours = []func(config.SyntaxColors) string{
	Comment:     func(c config.SyntaxColors) string { return c.Comment },
	Keyword:     func(c config.SyntaxColors) string { return c.Keyword },
	String:      func(c config.SyntaxColors) string { return c.String },
	Number:      func(c config.SyntaxColors) string { return c.Number },
	TypeName:    func(c config.SyntaxColors) string { return c.TypeName },
	Function:    func(c config.SyntaxColors) string { return c.Function },
	Operator:    func(c config.SyntaxColors) string { return c.Operator },
	Punctuation: func(c config.SyntaxColors) string { return c.Punctuation },
	Variable:    func(c config.SyntaxColors) string { return c.Variable },
	Constant:    func(c config.SyntaxColors) string { return c.Constant },
	Property:    func(c config.SyntaxColors) string { return c.Property },
	Tag:         func(c config.SyntaxColors) string { return c.Tag },
	Attribute:   func(c config.SyntaxColors) string { return c.Attribute },
	Heading:     func(c config.SyntaxColors) string { return c.Heading },
	Emphasis:    func(c config.SyntaxColors) string { return c.Emphasis },
	Strong:      func(c config.SyntaxColors) string { return c.Strong },
	Link:        func(c config.SyntaxColors) string { return c.Link },
	Code:        func(c config.SyntaxColors) string { return c.Code },
	ListMarker:  func(c config.SyntaxColors) string { return c.ListMarker },
	Quote:       func(c config.SyntaxColors) string { return c.Quote },
}

// Colour returns the configured colour of a role.
//
// None returns the empty string rather than a colour: a cell with no role is not
// painted, and inventing a colour for it would make plain text look like a
// decision the theme took.
func Colour(colors config.SyntaxColors, role Role) string {
	if !role.coloured() {
		return ""
	}
	return colours[role](colors)
}
