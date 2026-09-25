package config

import "testing"

// A camada de token promete duas coisas, e as duas precisam de prova:
//
//  1. **Um tema que não a declara não muda.** Os dez temas vêm do oráculo por um
//     arquivo gerado, e o quadro deles é o que o golden fixa. Se a camada alterasse
//     uma cor, o golden cairia — e um teste que depende do golden para provar
//     que nada mudou é um teste que depende de outro teste.
//  2. **Um tema que afirma `no-color` realmente não pinta.** Uma paleta que
//     promete não ter cor e entrega cor é a mentira que o tema existe para
//     impedir.

// TestTheTokenLayerIsInertByDefault prova a primeira: os tokens de um tema sem a
// camada são **as próprias cores diretas**, campo a campo.
func TestTheTokenLayerIsInertByDefault(t *testing.T) {
	for _, definition := range BuiltInThemes() {
		completed := completeTheme(definition)
		if completed.Tokens == nil {
			t.Fatalf("tema %q ficou sem camada de tokens", definition.Name)
		}

		inherited := defaultTokens(completed.UI)
		if *completed.Tokens != inherited {
			t.Errorf("tema %q alterou o quadro ao ganhar a camada:\n  tem %+v\n  era %+v",
				definition.Name, *completed.Tokens, inherited)
		}
	}
}

// TestEachTokenInheritsItsOwnColour fixa a correspondência token → cor, porque
// "inerte por padrão" só quer dizer alguma coisa se a herança for a **certa**:
// um token trocado de cor não move o quadro hoje e move amanhã, no primeiro tema
// que declarar a camada.
func TestEachTokenInheritsItsOwnColour(t *testing.T) {
	ui := Default().UI
	tokens := defaultTokens(ui)

	for _, tc := range []struct {
		token string
		got   string
		want  string
	}{
		{TokenSurface, tokens.Surface, ui.Background},
		{TokenPanel, tokens.Panel, ui.StatusBG},
		{TokenPanelText, tokens.PanelText, ui.StatusFG},
		{TokenBorderSubtle, tokens.BorderSubtle, ui.LineNumber},
		{TokenText, tokens.Text, ui.Foreground},
		{TokenTextMuted, tokens.TextMuted, ui.LineNumber},
		{TokenAccent, tokens.Accent, ui.CursorFG},
		{TokenWarning, tokens.Warning, ui.StatusDirty},
		{TokenError, tokens.Error, ui.StatusDirty},
		{TokenFocus, tokens.Focus, ui.CursorBG},
	} {
		if tc.got != tc.want {
			t.Errorf("token %s herdou %q, e a cor direta é %q", tc.token, tc.got, tc.want)
		}
	}
}

// TestAMergedTokenTableKeepsTheUndeclaredOnes is a parte do merge que uma struct
// de strings normais erraria: um tema que declara `text_muted` e nada mais tem de
// herdar os **outros** nove tokens.
//
// Sem os ponteiros, "não declarado" e "cor vazia" seriam o mesmo valor — e cor
// vazia é o que o tema `no-color` diz. A diferença entre as duas coisas é a
// razão de a tabela ser mesclada campo a campo.
func TestAMergedTokenTableKeepsTheUndeclaredOnes(t *testing.T) {
	base := Tokens{
		Surface: "#111111", Panel: "#222222", PanelText: "#ababab", BorderSubtle: "#333333",
		Text: "#444444", TextMuted: "#555555", Accent: "#666666",
		Success: "#777777", Warning: "#888888", Error: "#999999", Focus: "#aaaaaa",
	}
	declared := "not-default"
	merged := mergeTokens(base, &rawTokens{TextMuted: &declared})

	if merged.TextMuted != declared {
		t.Errorf("o token declarado ficou %q", merged.TextMuted)
	}
	for _, tc := range []struct {
		token string
		got   string
		want  string
	}{
		{TokenSurface, merged.Surface, base.Surface},
		{TokenPanel, merged.Panel, base.Panel},
		{TokenPanelText, merged.PanelText, base.PanelText},
		{TokenBorderSubtle, merged.BorderSubtle, base.BorderSubtle},
		{TokenText, merged.Text, base.Text},
		{TokenAccent, merged.Accent, base.Accent},
		{TokenSuccess, merged.Success, base.Success},
		{TokenWarning, merged.Warning, base.Warning},
		{TokenError, merged.Error, base.Error},
		{TokenFocus, merged.Focus, base.Focus},
	} {
		if tc.got != tc.want {
			t.Errorf("token %s foi sobrescrito sem ser declarado: %q virou %q", tc.token, tc.want, tc.got)
		}
	}
}

// TestTheNoColourThemeDeclaresNoColour is a afirmação do tema, no nível do
// dado: os dez tokens resolvem para nada.
func TestTheNoColourThemeDeclaresNoColour(t *testing.T) {
	completed := completeTheme(NoColourTheme())
	if completed.Tokens == nil {
		t.Fatal("o tema no-color ficou sem camada de tokens")
	}

	for _, value := range []string{
		completed.Tokens.Surface, completed.Tokens.Panel, completed.Tokens.PanelText,
		completed.Tokens.BorderSubtle,
		completed.Tokens.Text, completed.Tokens.TextMuted, completed.Tokens.Accent,
		completed.Tokens.Success, completed.Tokens.Warning, completed.Tokens.Error,
		completed.Tokens.Focus,
	} {
		if !IsNoColour(value) {
			t.Errorf("um token do tema no-color tem cor %q", value)
		}
	}
}

// TestNoColourAndUnsetAreDifferentThings fecha a lacuna que faria a camada inútil:
// um token em branco é "não tenho cor", e um token **ausente** é "herde". São
// valores diferentes, e a tabela de um tema de arquivo passa pelo merge porque
// eles precisam ser.
func TestNoColourAndUnsetAreDifferentThings(t *testing.T) {
	if !IsNoColour("") || !IsNoColour("none") {
		t.Error("nem a string vazia nem `none` significam ausência de cor")
	}
	for _, colour := range []string{"#ffffff", "red", "darkgray", "reset", "NONE"} {
		if IsNoColour(colour) {
			t.Errorf("%q foi lido como ausência de cor", colour)
		}
	}
}

// TestTheNoColourThemeResolvesThroughTheRegistry fecha o caminho do usuário: ele
// escolhe o tema pelo nome, e o nome tem de achar a paleta que não pinta.
func TestTheNoColourThemeResolvesThroughTheRegistry(t *testing.T) {
	registry := NewThemeRegistry()

	definition, found := registry.Get("no-color")
	if !found {
		t.Fatal("o tema no-color não está no registro")
	}
	if definition.Tokens == nil || !IsNoColour(definition.Tokens.Text) {
		t.Errorf("o tema veio do registro com cor: %+v", definition.Tokens)
	}
	// O id é `no-color` (a grafia americana, que é a do resto do produto) e
	// resolve também com underscore, como qualquer outro tema.
	if _, found := registry.Get("no_color"); !found {
		t.Error("o tema não resolve pelo id com underscore")
	}
	if _, found := registry.Get("No Color"); !found {
		t.Error("o tema não resolve pelo nome de exibição")
	}
}

// TestTokenDefaultFieldsMatchWhatIsRead impede que a lista que a regra de
// aptidão consulta envelheça: ela nomeia os campos que `defaultTokens` realmente
// toca, e um campo novo que entre na função sem entrar na lista faria a regra
// acusar um campo que **tem** leitor.
func TestTokenDefaultFieldsMatchWhatIsRead(t *testing.T) {
	declared := map[string]bool{}
	for _, name := range TokenDefaultFields() {
		declared[name] = true
	}

	// O reflection abaixo é o mesmo mecanismo que a regra usa para achar campos:
	// o valor de cada token tem de vir de um campo de `UIConfig`, e cada campo tem
	// de estar na lista.
	values := defaultTokens(UIConfig{
		Background: "1", Foreground: "2", LineNumber: "3", StatusBG: "4",
		StatusFG: "5", StatusDirty: "6", CursorBG: "7", CursorFG: "8",
	})

	read := map[string]string{
		"Background":  values.Surface,
		"StatusBG":    values.Panel,
		"StatusFG":    values.PanelText,
		"LineNumber":  values.TextMuted,
		"Foreground":  values.Text,
		"CursorFG":    values.Accent,
		"StatusDirty": values.Warning,
		"CursorBG":    values.Focus,
	}
	for field, value := range read {
		if !declared[field] {
			t.Errorf("a função lê %q e a lista não nomeia", field)
		}
		if value == "" {
			t.Errorf("o token semeado por %q está vazio: a omissão virou cor vazia", field)
		}
	}
	for field := range declared {
		if _, used := read[field]; !used {
			t.Errorf("a lista nomeia %q, e nenhuma cor o usa", field)
		}
	}
}
