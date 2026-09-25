package syntax

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/ori-team/oride/internal/config"
)

// markdownFamily is the lexer name whose tokens mean something other than what
// they mean in a programming language.
//
// Markdown has no keywords, but the lexer emits `Keyword` for a list marker and
// for a block quote, and emits `NameTag` for the text of a link — where in HTML
// those same token types are a tag and an attribute. One flat table would colour
// every Go `func` as a list marker, so the mapping is per family and the reason
// is written here rather than left as a surprise.
const markdownFamily = "markdown"

// Highlighter assigns roles to the lines of a document.
//
// The zero value is a highlighter with no engine, which paints nothing: a file
// whose language nobody can name is plain text, and that is a legitimate state
// rather than an error.
type Highlighter struct {
	lexer  chroma.Lexer
	family string
}

// Disabled is the highlighter of a document no lexer claims.
var Disabled = Highlighter{}

// Enabled reports whether there is an engine to run.
func (h Highlighter) Enabled() bool { return h.lexer != nil }

// For resolves the engine for a path.
//
// The configured `[[languages]]` entries are consulted first, by exact filename
// and then by extension, so a user who names a language the registry does not
// know still gets highlighting; the registry answers for everything else. Before
// this existed those entries were parsed by nobody at all.
func For(path string, languages []config.LanguageConfig) Highlighter {
	if name := configuredFor(path, languages); name != "" {
		if lexer := lexers.Get(name); lexer != nil {
			return newHighlighter(lexer)
		}
	}
	if lexer := lexers.Match(path); lexer != nil {
		return newHighlighter(lexer)
	}
	return Disabled
}

// newHighlighter reads the family off the lexer rather than being told it.
//
// The family decides the mapping, and a mapping chosen by the caller is a
// mapping two places can disagree about — so it is derived from the one thing
// that knows what language it is.
func newHighlighter(lexer chroma.Lexer) Highlighter {
	return Highlighter{lexer: lexer, family: strings.ToLower(lexer.Config().Name)}
}

// configuredFor is the language a `[[languages]]` entry claims for a path.
func configuredFor(path string, languages []config.LanguageConfig) string {
	if path == "" {
		return ""
	}
	name := filepath.Base(path)
	for _, language := range languages {
		for _, filename := range language.Filenames {
			if filename == name {
				return language.ID
			}
		}
	}

	extension := strings.TrimPrefix(filepath.Ext(path), ".")
	if extension == "" {
		return ""
	}
	for _, language := range languages {
		for _, candidate := range language.Extensions {
			if strings.EqualFold(strings.TrimPrefix(candidate, "."), extension) {
				return language.ID
			}
		}
	}
	return ""
}

// Line carries the roles of one line's bytes.
type Line struct {
	Tokens []Token
}

// Token is a run of a line with a role, as byte offsets into that line.
//
// Bytes, not cells and not columns: the buffer's offsets are the canonical
// index, and converting at the edge is the same rule the selection and the
// search follow.
type Token struct {
	Role  Role
	Start int
	End   int
}

// Highlight assigns roles to the lines of a document.
//
// The caller supplies the lines rather than the text because the buffer owns the
// definition of a line break — "\n", "\r\n", "\r" and three Unicode separators —
// and a second definition here would put the tokens of a CRLF file on the wrong
// line without failing anything. The lines are rejoined with "\n" to be lexed and
// split back at the offsets that joining produced, so the mapping is exact
// whatever the file used.
//
// A lexer that cannot read the source yields no tokens rather than an error: a
// malformed file is still a file the user is editing.
func (h Highlighter) Highlight(lines []string) []Line {
	if !h.Enabled() || len(lines) == 0 {
		return nil
	}

	iterator, err := h.lexer.Tokenise(nil, strings.Join(lines, "\n"))
	if err != nil {
		return nil
	}

	out := make([]Line, len(lines))
	splitter := newSplitter(lines)
	offset := 0
	for _, token := range iterator.Tokens() {
		start, end := offset, offset+len(token.Value)
		offset = end
		if role := h.roleOf(token); role != None {
			splitter.add(out, start, end, role)
		}
	}
	return out
}

// roleOf is what one lexer token means in this family.
func (h Highlighter) roleOf(token chroma.Token) Role {
	if h.family == markdownFamily {
		return markdownRole(token)
	}
	return languageRole(token.Type)
}

// languageRole is the mapping for a programming language.
func languageRole(tokenType chroma.TokenType) Role {
	switch tokenType {
	case chroma.NameFunction, chroma.NameFunctionMagic:
		return Function
	case chroma.NameClass, chroma.NameException, chroma.KeywordType:
		return TypeName
	case chroma.NameConstant, chroma.KeywordConstant:
		return Constant
	case chroma.NameProperty:
		return Property
	case chroma.NameAttribute:
		return Attribute
	case chroma.NameTag:
		return Tag
	case chroma.LiteralStringBacktick, chroma.LiteralStringHeredoc:
		return Code
	case chroma.GenericHeading, chroma.GenericSubheading:
		return Heading
	case chroma.GenericEmph, chroma.GenericUnderline:
		return Emphasis
	case chroma.GenericStrong:
		return Strong
	}

	// The sub-categories catch what the switch above does not name: a language
	// with a token type added after this table was written still gets a colour,
	// and a name this table forgot to enumerate is not left plain.
	switch {
	case tokenType.InSubCategory(chroma.Comment):
		return Comment
	case tokenType.InSubCategory(chroma.Keyword):
		return Keyword
	case tokenType.InSubCategory(chroma.Name):
		return Variable
	case tokenType.InSubCategory(chroma.Number):
		return Number
	case tokenType.InSubCategory(chroma.String):
		return String
	case tokenType.InSubCategory(chroma.Operator):
		return Operator
	case tokenType.InSubCategory(chroma.Punctuation):
		return Punctuation
	}
	return None
}

// markdownRoles overrides the programming mapping where Markdown means otherwise.
//
// Three collisions, each because the lexer emits the token type a programming
// language would use: `NameTag` is a link's text, `NameAttribute` is its target,
// and `LiteralString` is a code fence or an inline code span.
var markdownRoles = map[chroma.TokenType]Role{
	chroma.NameTag:       Link,
	chroma.NameAttribute: Link,
	chroma.LiteralString: Code,
}

// markdownRole is the mapping for Markdown.
func markdownRole(token chroma.Token) Role {
	if role, overridden := markdownRoles[token.Type]; overridden {
		return role
	}
	if token.Type == chroma.Keyword {
		// The lexer marks a list bullet and a block quote as a keyword, so the
		// text is what tells them apart — and from each other.
		return markdownMarker(token.Value)
	}
	return languageRole(token.Type)
}

// markdownMarker is the role of a Markdown keyword token.
func markdownMarker(value string) Role {
	trimmed := strings.TrimSpace(value)
	switch {
	case trimmed == "":
		return None
	case strings.HasSuffix(trimmed, ">"), trimmed == ">":
		return Quote
	case strings.Contains(value, ">"):
		// The block quote arrives as "<newline>> ", so the marker is inside the
		// token rather than at its edge.
		return Quote
	case isListMarker(trimmed):
		return ListMarker
	}
	return Keyword
}

// isListMarker reports whether the text opens a list item.
//
// The forms Markdown accepts: three bullets, and an ordered marker of digits and
// at most one dot. The dot has to close the marker — "1.5" is a number, not a
// list, and a reader who sees a list where a decimal was is reading a lie.
func isListMarker(text string) bool {
	switch text {
	case "-", "*", "+":
		return true
	}
	if text == "" {
		return false
	}

	separator := text[len(text)-1:]
	if separator != "." && separator != ")" {
		return false
	}
	digits := text[:len(text)-1]
	if digits == "" || len(digits) > 9 {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// splitter turns a token's byte range in the joined text into per-line tokens.
//
// A lexer token may cross a line break — a Markdown heading, a heredoc — so the
// range is clipped per line. The cursor only moves forward because the tokens
// arrive in order, which is what keeps a large file linear rather than quadratic.
type splitter struct {
	starts  []int
	lengths []int
	cursor  int
}

// newSplitter records where each line begins in the joined text.
func newSplitter(lines []string) *splitter {
	starts := make([]int, len(lines))
	lengths := make([]int, len(lines))
	offset := 0
	for index, line := range lines {
		starts[index] = offset
		lengths[index] = len(line)
		// One byte for the "\n" that joined this line to the next.
		offset += len(line) + 1
	}
	return &splitter{starts: starts, lengths: lengths}
}

// add records the role over the part of each line the range touches.
func (s *splitter) add(out []Line, from, to int, role Role) {
	for s.cursor < len(s.starts) && s.starts[s.cursor]+s.lengths[s.cursor] <= from {
		s.cursor++
	}
	for index := s.cursor; index < len(s.starts); index++ {
		lineStart, lineEnd := s.starts[index], s.starts[index]+s.lengths[index]
		if lineStart >= to {
			break
		}
		start, end := max(lineStart, from), min(lineEnd, to)
		if end <= start {
			continue
		}
		out[index].Tokens = append(out[index].Tokens, Token{
			Role:  role,
			Start: start - lineStart,
			End:   end - lineStart,
		})
	}
}
