package syntax

import (
	"reflect"
	"testing"

	"github.com/ori-team/oride/internal/config"
)

// TestEveryRoleHasAColourAndAName walks the table in both directions.
//
// A role declared without a colour is a role the theme cannot paint, and a
// configured field no role reads is the defect this package exists to end:
// nineteen of the twenty were parsed and never consulted for the whole life of
// the Go port. Both directions are checked because each alone passes while the
// other is broken.
func TestEveryRoleHasAColourAndAName(t *testing.T) {
	roles := All()
	if len(roles) != 20 {
		t.Fatalf("a tabela tem %d papéis; a configuração declara 20: %v", len(roles), roles)
	}

	names := map[string]bool{}
	for _, role := range roles {
		if name := role.String(); name == "" || name == "None" {
			t.Errorf("papel %d sem nome utilizável: %q", role, name)
		} else if names[name] {
			t.Errorf("papel repetido: %q", name)
		} else {
			names[name] = true
		}
		if Colour(config.Default().Syntax, role) == "" {
			t.Errorf("papel %s não lê cor nenhuma", role.String())
		}
	}

	// A field of the configuration that no role reads is a field nobody acts on.
	fields := reflect.TypeOf(config.SyntaxColors{})
	for index := range fields.NumField() {
		if name := fields.Field(index).Name; !names[name] {
			t.Errorf("config.SyntaxColors.%s não tem papel que a leia", name)
		}
	}
}

// TestNoRoleIsNotPainted keeps the absence of a role out of the palette.
//
// A cell with no role takes the surface's own foreground; inventing a colour for
// it would make plain text look like a decision the theme took.
func TestNoRoleIsNotPainted(t *testing.T) {
	if got := Colour(config.Default().Syntax, None); got != "" {
		t.Errorf("papel None devolveu %q", got)
	}
	if got := Colour(config.Default().Syntax, Role(len(roleNames))); got != "" {
		t.Errorf("papel fora da tabela devolveu %q", got)
	}
}

// TestColourFollowsTheField is the contraprova of the accessors.
//
// The test reads the value out of the struct rather than comparing two functions,
// so it fails if a role stops following its own field — which is the difference
// between a renamed field and a colour that silently became empty.
func TestColourFollowsTheField(t *testing.T) {
	colors := config.Default().Syntax
	colors.Comment = "#010203"

	if got := Colour(colors, Comment); got != "#010203" {
		t.Errorf("Comment devolveu %q, quer #010203", got)
	}
	if got := Colour(colors, Keyword); got != colors.Keyword {
		t.Errorf("Keyword devolveu %q, quer %q", got, colors.Keyword)
	}
}

// TestGoSourceGetsEveryProgrammingRole is what a reader sees when a .go file
// opens: a keyword, a comment, a function, a variable, a number, an operator and
// punctuation, each with its own role.
func TestGoSourceGetsEveryProgrammingRole(t *testing.T) {
	source := []string{"package main", "", "// doc", "func main() { x := 1 }"}
	roles := rolesIn(source, For("main.go", nil).Highlight(source))

	for _, expected := range []struct {
		text string
		role Role
	}{
		{"package", Keyword},
		{"// doc", Comment},
		{"func", Keyword},
		{"main", Function},
		{"x", Variable},
		{"1", Number},
		{":=", Operator},
		{"{", Punctuation},
	} {
		if got := roles[expected.text]; got != expected.role {
			t.Errorf("%q recebeu papel %s, esperava %s", expected.text, got, expected.role)
		}
	}
}

// TestMarkdownRolesUseTheMarkdownTable pins the three collisions and the two
// markers, which is the whole reason the mapping is per family.
func TestMarkdownRolesUseTheMarkdownTable(t *testing.T) {
	source := []string{
		"# Título",
		"- item",
		"> citação",
		"`código` **forte** *itálico*",
		"[texto](http://exemplo)",
	}
	roles := rolesIn(source, For("notes.md", nil).Highlight(source))

	for _, expected := range []struct {
		text string
		role Role
	}{
		{"# Título", Heading},
		{"-", ListMarker},
		{"> ", Quote},
		{"`código`", Code},
		{"**forte", Strong},
		{"itálico", Emphasis},
		{"texto", Link},
		{"http://exemplo", Link},
		// The prose after the marker carries no role: a colour that claims a
		// word is punctuation is a lie the reader sees.
		{"item", None},
	} {
		if got := roles[expected.text]; got != expected.role {
			t.Errorf("%q recebeu papel %s, esperava %s", expected.text, got, expected.role)
		}
	}
}

// TestHTMLKeepsTagAndAttribute is the other side of the markdown override: the
// same two token types mean what a programming language says they mean.
func TestHTMLKeepsTagAndAttribute(t *testing.T) {
	source := []string{`<a href="x">y</a>`}
	roles := rolesIn(source, For("page.html", nil).Highlight(source))

	if got := roles["a"]; got != Tag {
		t.Errorf("o nome da tag recebeu %s, esperava Tag", got)
	}
	if got := roles["href"]; got != Attribute {
		t.Errorf("o nome do atributo recebeu %s, esperava Attribute", got)
	}
}

// TestTokensAreOrderedAndWithinTheLine is the property the painter relies on: it
// walks a line's tokens in order, so an overlap would make the second one win
// silently and a token past the end would slice the padding.
//
// The tokens do not tile the line — whitespace and prose carry no role and are
// painted plain — so what is checked is that they never disagree about the bytes
// between them.
func TestTokensAreOrderedAndWithinTheLine(t *testing.T) {
	source := []string{
		"package main",
		"",
		"// um comentário",
		"func soma(a, b int) int { return a + b }",
	}
	lines := For("main.go", nil).Highlight(source)

	if len(lines) != len(source) {
		t.Fatalf("o realce devolveu %d linhas para %d", len(lines), len(source))
	}
	for index, line := range lines {
		next := 0
		for _, token := range line.Tokens {
			if token.Start < next {
				t.Errorf("linha %d: token começa em %d, sobre o anterior que terminava em %d (%+v)",
					index, token.Start, next, token)
			}
			if token.End <= token.Start {
				t.Errorf("linha %d: token vazio %+v", index, token)
			}
			if token.End > len(source[index]) {
				t.Errorf("linha %d: token passa do fim da linha %+v", index, token)
			}
			next = token.End
		}
	}
}

// TestUnknownLanguageIsPlainText is fail-closed: a file nobody can name lexes to
// nothing rather than to an error, and the document is still editable.
func TestUnknownLanguageIsPlainText(t *testing.T) {
	highlighter := For("notes.qqq", nil)
	if highlighter.Enabled() {
		t.Errorf("a extensão desconhecida resolveu um lexer: %+v", highlighter)
	}
	if lines := highlighter.Highlight([]string{"qualquer coisa"}); lines != nil {
		t.Errorf("realce sem lexer devolveu %+v", lines)
	}
}

// TestEmptyDocumentIsNotAnError keeps a new file from taking a path nobody
// thought about.
func TestEmptyDocumentIsNotAnError(t *testing.T) {
	if lines := For("main.go", nil).Highlight(nil); lines != nil {
		t.Errorf("documento vazio devolveu %+v", lines)
	}
	if lines := For("", nil).Highlight([]string{"texto"}); lines != nil {
		t.Errorf("documento sem caminho devolveu %+v", lines)
	}
}

// TestConfiguredLanguagesWinOverTheRegistry is what gives `[[languages]]` a
// reader: a user entry names the language of a file the registry would guess
// differently, or of one it does not know at all.
func TestConfiguredLanguagesWinOverTheRegistry(t *testing.T) {
	languages := []config.LanguageConfig{
		{ID: "toml", Filenames: []string{"Cargo.qqq"}, Extensions: []string{".qqq"}},
	}

	byFilename := For("Cargo.qqq", languages)
	if byFilename.family != "toml" {
		t.Errorf("o nome exato resolveu %q, esperava toml", byFilename.family)
	}

	byExtension := For("conf.QQQ", languages)
	if byExtension.family != "toml" {
		t.Errorf("a extensão resolveu %q, esperava toml", byExtension.family)
	}
	if lines := byExtension.Highlight([]string{"a = 1"}); len(lines) != 1 || len(lines[0].Tokens) == 0 {
		t.Errorf("a linguagem configurada não produziu tokens: %+v", lines)
	}

	// An entry naming a language the registry does not have must not take the
	// highlighting away: the registry still answers.
	unknown := []config.LanguageConfig{{ID: "nao-existe", Extensions: []string{".go"}}}
	if family := For("main.go", unknown).family; family != "go" {
		t.Errorf("linguagem desconhecida devolveu %q, esperava o registro", family)
	}
}

// TestListMarkerForms covers the markers the lexer emits, and the ones it does
// not, separately.
//
// The lexer only ever produces a keyword token for "-" and for "N."; "+", "*"
// and "N)" are accepted by the mapping because a token that arrived would then be
// read correctly, and the test says which half of that is observable today.
func TestListMarkerForms(t *testing.T) {
	for _, marker := range []string{"-", "1.", "2.", "10."} {
		// Two lines, because the lexer only recognises a list when something
		// follows it: a one-line document is read as prose.
		source := []string{marker + " item", "texto"}
		lines := For("notes.md", nil).Highlight(source)
		if got := firstRole(lines[0]); got != ListMarker {
			t.Errorf("o marcador %q recebeu %s, esperava ListMarker", marker, got)
		}
	}

	// The mapping reads a token, not a document, so every form the function knows
	// is checked here even when the lexer never produces it.
	for _, text := range []string{"-", "*", "+", "1.", "2)", "10."} {
		if !isListMarker(text) {
			t.Errorf("%q não foi lido como marcador de lista", text)
		}
	}
	for _, text := range []string{"", "package", "1.5", "2024-01-01", "1234567890.", "-x"} {
		if isListMarker(text) {
			t.Errorf("%q foi lido como marcador de lista", text)
		}
	}
}

// rolesIn maps a token's text to its role, walking the source beside the
// result.
//
// The source has to come along because the highlighter keeps offsets and not
// text: an assertion on the offsets alone would pass with every role shifted one
// byte to the left, which is the bug a test like this exists to catch.
func rolesIn(source []string, lines []Line) map[string]Role {
	roles := map[string]Role{}
	for index, line := range lines {
		if index >= len(source) {
			break
		}
		for _, token := range line.Tokens {
			roles[source[index][token.Start:token.End]] = token.Role
		}
	}
	return roles
}

// firstRole is the role of a line's first token.
func firstRole(line Line) Role {
	if len(line.Tokens) == 0 {
		return None
	}
	return line.Tokens[0].Role
}
