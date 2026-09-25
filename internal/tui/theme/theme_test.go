package theme

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/syntax"
	"github.com/ori-team/oride/internal/tui/layout"
)

func sampleConfig() (config.UIConfig, config.SyntaxColors) {
	return config.UIConfig{
			Background: "#282a36", CursorBG: "#44475a", CursorFG: "#f8f8f2",
			LineNumber: "#6272a4", StatusBG: "#282a36", StatusFG: "#f8f8f2",
			StatusDirty: "#ff5555",
		}, config.SyntaxColors{
			Comment: "#6272a4",
		}
}

// TestNoColorAppliesNothing is the whole no-colour guarantee, and it is asserted
// over every role rather than one: a theme that suppresses colour in one place and
// paints in another is the error the rule names.
func TestNoColorAppliesNothing(t *testing.T) {
	ui, colors := sampleConfig()
	plain := New(ui, colors, NoColor)

	roles := map[string]string{
		"selection": plain.Selection().Render("texto"),
		"gutter":    plain.Gutter().Render("12"),
		"status":    plain.Status().Render("status"),
		"dirty":     plain.StatusDirty().Render("sujo"),
		"comment":   plain.Syntax(syntax.Comment).Render("// nota"),
	}
	for role, rendered := range roles {
		if strings.ContainsRune(rendered, 0x1b) {
			t.Errorf("%s: o perfil sem cor emitiu sequência de escape: %q", role, rendered)
		}
		if rendered != strings.TrimSpace(rendered) && strings.Contains(rendered, "\x1b") {
			t.Errorf("%s trouxe escape", role)
		}
	}
}

// TestNoColorRendersTheTextItself: suppressing colour must not suppress content.
func TestNoColorRendersTheTextItself(t *testing.T) {
	ui, colors := sampleConfig()
	plain := New(ui, colors, NoColor)

	if got := plain.Selection().Render("texto"); got != "texto" {
		t.Errorf("renderizou %q, esperado o texto sem alteração", got)
	}
}

// TestColorProfilesEmitSequences proves the other direction: a rule that only ever
// suppressed colour would pass the tests above while showing a monochrome editor.
func TestColorProfilesEmitSequences(t *testing.T) {
	for _, profile := range []Profile{ANSI16, ANSI256, TrueColor} {
		ui, colors := sampleConfig()
		themed := New(ui, colors, profile)

		if got := themed.Selection().Render("texto"); !strings.ContainsRune(got, 0x1b) {
			t.Errorf("%v: nenhuma sequência emitida: %q", profile, got)
		}
	}
}

// TestStyledTextMeasuresTheSameWidth is the invariant that keeps a styled frame
// aligned: escapes occupy no cells.
func TestStyledTextMeasuresTheSameWidth(t *testing.T) {
	ui, colors := sampleConfig()
	themed := New(ui, colors, TrueColor)

	for _, text := range []string{"texto", "日本語", "com ç e ã", ""} {
		plain := layout.Width(text)
		styled := layout.Width(themed.Selection().Render(text))

		if styled != plain {
			t.Errorf("%q: sem estilo %d células, com estilo %d", text, plain, styled)
		}
	}
}

// TestAStyledRowPadsToTheExactWidth is what a frame actually does, and where a
// width bug would show up as a diagonal layout.
func TestAStyledRowPadsToTheExactWidth(t *testing.T) {
	ui, colors := sampleConfig()
	themed := New(ui, colors, TrueColor)

	for _, width := range []int{10, 40, 79, 80} {
		row := layout.Pad(themed.Selection().Render("um rótulo"), width)
		if got := layout.Width(row); got != width {
			t.Errorf("largura %d: linha com %d células", width, got)
		}
	}
}

// TestEmptyAndResetMeanNoColour: both are normal in the shipped themes, and both
// are values the style library cannot resolve.
func TestEmptyAndResetMeanNoColour(t *testing.T) {
	for _, value := range []string{"", "reset", "none"} {
		if got := parse(value); got != nil {
			t.Errorf("parse(%q) = %v, esperado nada", value, got)
		}
	}
}

// TestParseAcceptsHexAndNames.
func TestParseAcceptsHexAndNames(t *testing.T) {
	for _, value := range []string{"#282a36", "#fff", "red", "brightblack"} {
		if parse(value) == nil {
			t.Errorf("parse(%q) não resolveu", value)
		}
	}
}

// TestNamedColoursActuallyRender is the regression guard for a bug that every
// other test in this file missed.
//
// They used hex, the shipped configuration uses names, and an unresolved name is a
// silent no-op — so the package passed its tests while the product rendered
// monochrome. This asserts the rendered output, not the parsed value.
func TestNamedColoursActuallyRender(t *testing.T) {
	for _, name := range []string{"darkgray", "red", "brightblack"} {
		rendered := lipgloss.NewStyle().Foreground(parse(name)).Render("x")
		if !strings.ContainsRune(rendered, 0x1b) {
			t.Errorf("a cor %q não produziu estilo: %q", name, rendered)
		}
	}
}

// TestShippedDefaultRendersWithColour uses the configuration the product ships,
// which is what the previous test could not cover.
func TestShippedDefaultRendersWithColour(t *testing.T) {
	shipped := config.Default()
	built := New(shipped.UI, shipped.Syntax, TrueColor)

	for role, rendered := range map[string]string{
		"status": built.Status().Render("x"),
		"gutter": built.Gutter().Render("1"),
	} {
		if !strings.ContainsRune(rendered, 0x1b) {
			t.Errorf("%s: a configuração padrão não produz cor: %q", role, rendered)
		}
	}
}

// TestProfileOrdering lets a caller ask "at least this capable" without a switch.
func TestProfileOrdering(t *testing.T) {
	if !(NoColor < ANSI16 && ANSI16 < ANSI256 && ANSI256 < TrueColor) {
		t.Error("os perfis não estão ordenados por capacidade")
	}
}

// TestEverySyntaxRolePaintsItsOwnColour is the contraprova of the single
// accessor.
//
// Each of the twenty roles gets a distinct colour, so a role reading the wrong
// field — or no field, which the shipped palette hides because several roles
// share "reset" — shows up as a repeat or as a missing escape. The lesson of
// TestNamedColoursActuallyRender applies twice: the assertion is on the rendered
// output, because a role that resolves nothing is a silent no-op.
func TestEverySyntaxRolePaintsItsOwnColour(t *testing.T) {
	colors := config.Default().Syntax
	value := reflect.ValueOf(&colors).Elem()

	// The style library renders a true colour as decimal components, so the
	// expectation is built the same way: asserting on the hex would assert on the
	// input rather than on what the terminal receives.
	expected := map[syntax.Role]string{}
	for index, role := range syntax.All() {
		if !value.FieldByName(role.String()).CanSet() {
			t.Fatalf("o papel %s não corresponde a um campo de SyntaxColors", role)
		}
		value.FieldByName(role.String()).SetString(fmt.Sprintf("#%02x4060", 16+index))
		expected[role] = fmt.Sprintf("38;2;%d;64;96", 16+index)
	}

	built := New(config.Default().UI, colors, TrueColor)
	seen := map[string]syntax.Role{}
	for _, role := range syntax.All() {
		rendered := built.Syntax(role).Render("x")
		if !strings.Contains(rendered, expected[role]) {
			t.Errorf("%s: pintou %q, esperava a cor %s", role, rendered, expected[role])
		}
		if other, repeated := seen[rendered]; repeated {
			t.Errorf("%s e %s pintam o mesmo: %q", other, role, rendered)
		}
		seen[rendered] = role
	}

	// A role with no configured colour paints nothing, which is the promise the
	// absence makes.
	if got := built.Syntax(syntax.None).Render("x"); got != "x" {
		t.Errorf("o papel None pintou %q", got)
	}
}
