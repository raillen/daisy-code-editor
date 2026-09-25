package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ThemeDefinition is a named set of UI and syntax colours.
type ThemeDefinition struct {
	Name   string
	IsDark bool
	// UI and Syntax carry only the colours the theme overrides. A theme that
	// ships no colours inherits the product defaults, so a change to the
	// defaults reaches every theme instead of drifting from a copy.
	UI     UIConfig
	Syntax SyntaxColors
	// Tokens is the semantic layer: what each **intent** resolves to. It is
	// optional — a theme that declares none inherits `defaultTokens`, which is
	// the direct colours above. É essa opcionalidade que mantém os dez temas
	// gerados do oráculo byte a byte iguais enquanto a camada existe.
	Tokens *Tokens
}

// BuiltInThemes returns the shipped themes, with inherited colours filled in.
func BuiltInThemes() []ThemeDefinition {
	out := make([]ThemeDefinition, 0, len(builtInThemes))
	for _, theme := range builtInThemes {
		out = append(out, completeTheme(theme))
	}
	return out
}

// completeTheme fills the sections a theme left to the defaults.
func completeTheme(theme ThemeDefinition) ThemeDefinition {
	defaults := Default()
	if theme.UI == (UIConfig{}) {
		theme.UI = defaults.UI
	}
	if theme.Syntax == (SyntaxColors{}) {
		theme.Syntax = defaults.Syntax
	}
	// A camada de tokens nasce **depois** das cores diretas, porque é delas que
	// ela herda: um token que caísse no zero seria indistinguível de um tema
	// `no-color`, e a diferença entre "não defini" e "não tenho cor" é exatamente
	// o que a camada precisa preservar.
	if theme.Tokens == nil {
		inherited := defaultTokens(theme.UI)
		theme.Tokens = &inherited
	}
	return theme
}

// NormalizeThemeName produces the lookup key for a theme name.
//
// Case, spaces and underscores all fold to a hyphen, so `One Dark`,
// `one_dark` and `one-dark` name the same theme — which is what a user typing a
// name into a config file or a command line will assume.
func NormalizeThemeName(name string) string {
	normalized := strings.TrimSpace(strings.ToLower(name))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return strings.ReplaceAll(normalized, "_", "-")
}

// ThemeRegistry holds the themes a session can switch between.
type ThemeRegistry struct {
	themes map[string]ThemeDefinition
	order  []string
}

// NewThemeRegistry returns a registry with the built-in themes.
func NewThemeRegistry() *ThemeRegistry {
	registry := &ThemeRegistry{themes: map[string]ThemeDefinition{}}
	for _, theme := range BuiltInThemes() {
		registry.Register(theme)
	}
	// `no-color` por último, e fora do arquivo gerado: é um tema que o oráculo
	// não tem, porque lá a mesma coisa é um perfil do terminal.
	registry.Register(NoColourTheme())
	return registry
}

// Register adds or replaces a theme.
func (r *ThemeRegistry) Register(theme ThemeDefinition) {
	key := NormalizeThemeName(theme.Name)
	if _, exists := r.themes[key]; !exists {
		r.order = append(r.order, key)
	}
	r.themes[key] = completeTheme(theme)
}

// Get returns a theme by name.
func (r *ThemeRegistry) Get(name string) (ThemeDefinition, bool) {
	theme, ok := r.themes[NormalizeThemeName(name)]
	return theme, ok
}

// ListNames returns the registered names, in registration order.
//
// Registration order, not sorted: the built-ins are curated in a deliberate
// order and the picker shows them that way.
func (r *ThemeRegistry) ListNames() []string {
	out := make([]string, 0, len(r.order))
	for _, key := range r.order {
		out = append(out, r.themes[key].Name)
	}
	return out
}

// Len returns how many themes are registered.
func (r *ThemeRegistry) Len() int { return len(r.themes) }

// rawTheme is the on-disk shape of a theme file.
type rawTheme struct {
	Name   string     `toml:"name"`
	IsDark *bool      `toml:"is_dark"`
	UI     *rawUI     `toml:"ui"`
	Syntax *rawSyntax `toml:"syntax"`
	// Tokens is the semantic layer, and it merges **field by field**: um tema
	// que declara `text_muted` e deixa o resto de fora herda o resto. Um struct
	// de strings normais trataria a ausência como "cor vazia", que é o mesmo
	// valor de `no-color` — e a diferença entre "não defini" e "não tenho cor" é
	// exatamente o que esta camada existe para não perder.
	Tokens *rawTokens `toml:"tokens"`
}

// rawTokens is `[tokens]` as it arrives from a file: every field optional.
type rawTokens struct {
	Surface       *string `toml:"surface"`
	Panel         *string `toml:"panel"`
	PanelText     *string `toml:"panel_text"`
	BorderSubtle  *string `toml:"border_subtle"`
	Text          *string `toml:"text"`
	TextMuted     *string `toml:"text_muted"`
	Accent        *string `toml:"accent"`
	Success       *string `toml:"success"`
	Warning       *string `toml:"warning"`
	Error         *string `toml:"error"`
	Focus         *string `toml:"focus"`
	LineHighlight *string `toml:"line_highlight"`
}

// mergeTokens applies the declared tokens over an inherited layer.
//
// Uma sequência de `if` e não um mapa de ponteiros: um mapa de `**string` é
// escreve o mesmo campo duas vezes, e o teste que pega isso é o compilador — mas
// um compilador que aceita `**string` não está dizendo nada. A sequência
// também nomeia o token na linha, que é o que o teste de merge cita.
func mergeTokens(base Tokens, raw *rawTokens) *Tokens {
	if raw == nil {
		return nil
	}
	merged := base
	if raw.Surface != nil {
		merged.Surface = *raw.Surface
	}
	if raw.Panel != nil {
		merged.Panel = *raw.Panel
	}
	if raw.PanelText != nil {
		merged.PanelText = *raw.PanelText
	}
	if raw.BorderSubtle != nil {
		merged.BorderSubtle = *raw.BorderSubtle
	}
	if raw.Text != nil {
		merged.Text = *raw.Text
	}
	if raw.TextMuted != nil {
		merged.TextMuted = *raw.TextMuted
	}
	if raw.Accent != nil {
		merged.Accent = *raw.Accent
	}
	if raw.Success != nil {
		merged.Success = *raw.Success
	}
	if raw.Warning != nil {
		merged.Warning = *raw.Warning
	}
	if raw.Error != nil {
		merged.Error = *raw.Error
	}
	if raw.Focus != nil {
		merged.Focus = *raw.Focus
	}
	if raw.LineHighlight != nil {
		merged.LineHighlight = *raw.LineHighlight
	}
	return &merged
}

// LoadFromDir registers every `.toml` theme in a directory.
//
// A missing directory is not an error — most users have none — but a malformed
// file is reported, naming the file. A theme that silently failed to load looks
// exactly like a theme that does not exist, and the user would have no way to
// tell which.
func (r *ThemeRegistry) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("lendo temas em %s: %w", dir, err)
	}

	// Sorted so the registration order does not depend on the filesystem.
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		path := filepath.Join(dir, name)
		theme, err := loadThemeFile(path)
		if err != nil {
			return err
		}
		r.Register(theme)
	}
	return nil
}

func loadThemeFile(path string) (ThemeDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ThemeDefinition{}, fmt.Errorf("lendo tema %s: %w", path, err)
	}

	var raw rawTheme
	if err := toml.Unmarshal(data, &raw); err != nil {
		return ThemeDefinition{}, fmt.Errorf("tema %s: %w", path, err)
	}
	if strings.TrimSpace(raw.Name) == "" {
		// Falling back to the file name would produce a theme the user cannot
		// refer to by the name they wrote.
		return ThemeDefinition{}, fmt.Errorf("tema %s: campo `name` ausente", path)
	}

	theme := ThemeDefinition{Name: raw.Name, IsDark: true}
	if raw.IsDark != nil {
		theme.IsDark = *raw.IsDark
	}

	base := Default()
	theme.UI = base.UI
	theme.Syntax = base.Syntax
	if raw.UI != nil {
		mergeUI(&theme.UI, raw.UI)
	}
	if raw.Syntax != nil {
		mergeSyntax(&theme.Syntax, raw.Syntax)
	}
	// Os tokens herdam das cores diretas **já mescladas**, para que um tema de
	// arquivo que declara `[ui].line_number` e nenhum token tenha o token mudo
	// seguindo a cor que ele mesmo acabou de declarar.
	if raw.Tokens != nil {
		inherited := defaultTokens(theme.UI)
		theme.Tokens = mergeTokens(inherited, raw.Tokens)
	}
	return theme, nil
}

// LoadThemeRegistry returns a registry with the built-ins plus any theme files
// the user and the workspace provide.
//
// Workspace themes load after user themes, so a project can pin its own palette
// without editing anything global.
func LoadThemeRegistry(workspace string) *ThemeRegistry {
	registry := NewThemeRegistry()

	if base, err := os.UserConfigDir(); err == nil {
		_ = registry.LoadFromDir(filepath.Join(base, "oride", "themes"))
	}
	if workspace != "" {
		_ = registry.LoadFromDir(filepath.Join(workspace, ".oride", "themes"))
	}
	return registry
}
