package architecture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ori-team/oride/internal/config"
)

// configField is one field of a configuration table.
type configField struct {
	Table string
	Name  string
	Path  string
}

// fieldPattern matches a struct field declaration inside a Go source file.
//
// A linha de um campo de struct é `	Nome Tipo` — com indentação de tabulação e o
// tipo depois. O padrão é deliberadamente **estreito**: o que não parece um campo
// de struct não entra, porque um padrão largo transformaria os 200 arquivos do
// repositório em falsos positivos e a regra viraria ruído.
var fieldPattern = regexp.MustCompile(`(?m)^\t([A-Z]\w*)\s+[\[\]\*\w\.]+`)

// selectorPattern matches a read of a field: `.Shell`, `. TreeWidth`.
//
// A leitura é um **seletor**, e não uma declaração: `config.TerminalConfig` tem o
// campo `Shell` em uma linha, e o que interessa é o `cfg.Shell` em outro pacote.
var selectorPattern = regexp.MustCompile(`\.\s*([A-Z]\w*)`)

// configImport is the import path whose presence marks a file as one that can
// possibly read the configuration.
const configImport = "internal/config"

// configTables are the structs whose fields are configuration.
//
// A lista é explícita em vez de "toda struct do pacote": `Config` é o agregado, e
// as tabelas de apoio (`UIConfig`, `SyntaxColors`) **têm** leitores — a regra
// mede honestidade da configuração, não cobertura de tipos.
var configTables = map[string]bool{
	"EditorConfig":   true,
	"TreeConfig":     true,
	"TerminalConfig": true,
	"LSPConfig":      true,
	"MarkdownConfig": true,
	"LanguageConfig": true,
	"UIConfig":       true,
	"SyntaxColors":   true,
}

// ConfigFieldsWithoutReader is the debt: fields nobody reads outside the config
// package, with the reason each one is still there.
//
// A regra é inútil se falhar com o que já existe: um teste que nasce vermelho em
// 15 casos é um teste que ninguém roda, e um teste que ninguém roda é pior do que
// nenhum. A lista transforma dívida silenciosa em dívida **nomeada** — e a regra
// falha no instante em que um campo sai daqui sem ganhar leitor.
//
// Uma entrada é removida quando o leitor chega. Uma entrada **não** é acrescentada
// para calar a regra: acrescentar é dizer que a chave é inerte de propósito, e o
// motivo tem que dizer por que isso é aceitável.
var ConfigFieldsWithoutReader = map[string]string{
	"EditorConfig.FormatOnSave":      "M4 — formatação ao salvar precisa de um formateador; o campo é lido quando ele existir",
	"EditorConfig.UseEditorconfig":   "M4 — .editorconfig é resolvido no carregamento, mas o campo decide se isso acontece",
	"EditorConfig.CompletionAuto":    "M4 — auto-complete depende do cliente LSP subir",
	"EditorConfig.CompletionMinChar": "M4 — idem",
	"TreeConfig.GitStatus":           "M4 — o rótulo de status no painel SCM, que ainda não existe",
	"UIConfig.GutterWidth":           "M4 — a largura da gutter é derivada do número de linhas pela própria editorview; o campo documenta um valor que nada usa",
	"TerminalConfig.Shell":           "M4 — o painel de terminal, que ainda não existe",
	"TerminalConfig.DefaultHeight":   "M4 — idem",
	"LSPConfig.OriscriptCommand":     "M4 — cliente LSP sobe no M4",
	"LSPConfig.Servers":              "M4 — idem",
	"LSPConfig.TimeoutMS":            "M4 — idem",
	"MarkdownConfig.TerminalImages":  "M5 — imagens no preview Markdown, que depende de glamou/x-image",
	"LanguageConfig.LineComment":     "M4 — toggle_comment, que usa o comentário de linha da linguagem",
	"LanguageConfig.BlockCommentEnd": "M4 — comentário de bloco em toggle_comment",
	"LanguageConfig.LSPCommand":      "M4 — cliente LSP sobe no M4",
	"LanguageConfig.CompletionWords": "M4 — palavras de completion vêm do LSP",
}

// ConfigurationFitness reports configuration fields that nothing reads.
//
// The scan is lexical on purpose: it looks for the field's name outside the config
// package, which is the same question a reader asks — "who uses this?" — and it
// costs one pass over the repository instead of a type-checker. A field read only
// inside `internal/config` is, by definition, read by nobody.
func ConfigurationFitness(root string) ([]string, error) {
	fields, err := collectConfigFields(root)
	if err != nil {
		return nil, err
	}

	readers, err := fieldOccurrencesOutsideConfig(root)
	if err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		// Um repositório sem tabela de configuração não deve nada a esta regra, e
		// o mapa de dívida é do **Oride**: aplicá-lo a outro repositório acusaria
		// dívida que não é dele.
		return nil, nil
	}

	// As cores de `[ui]` que **semeiam** a camada de tokens são leitoras, mesmo
	// que a leitura aconteça dentro do config: elas são o valor de omissão de um
	// tema que não declara `[tokens]`, e `internal/config.TokenDefaultFields` — a
	// lista ao lado da função que faz a leitura — é quem as nomeia. Um campo fora
	// dessa lista continua sendo um campo sem consumidor.
	seeded := map[string]bool{}
	for _, name := range config.TokenDefaultFields() {
		seeded[name] = true
	}

	var out []string
	for _, field := range fields {
		key := field.Table + "." + field.Name
		if reason, excused := ConfigFieldsWithoutReader[key]; excused {
			if reason == "" {
				out = append(out, fmt.Sprintf("%s está na dívida sem motivo escrito", key))
			}
			continue
		}
		if readers[field.Name] == 0 && !(seeded[field.Name] && field.Table == "UIConfig") {
			out = append(out, fmt.Sprintf("%s não tem leitor fora de internal/config", key))
		}
	}

	// O inverso também conta: uma entrada de dívida cujo campo **não existe mais** é
	// uma dívida de mentira, que alguém continua "paying" sem saber.
	//
	// A verificação é por **tabela presente**: o mapa de dívida é deste projeto e
	// uma fixture — ou outro repositório — pode ter outras tabelas, e acusar
	// dívida que não é dele seria um falso positivo em toda regra de código.
	for key := range ConfigFieldsWithoutReader {
		table, _, found := strings.Cut(key, ".")
		if !found || !hasTable(fields, table) {
			continue
		}
		if !contains(fields, key) {
			out = append(out, fmt.Sprintf("dívida %q não corresponde a nenhum campo de configuração", key))
		}
	}

	sort.Strings(out)
	return out, nil
}

// collectConfigFields reads the field list of every configuration table.
func collectConfigFields(root string) ([]configField, error) {
	dir := filepath.Join(root, "internal", "config")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Um repositório sem pacote de configuração não deve nada a esta regra.
			return nil, nil
		}
		return nil, err
	}

	var out []configField
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		out = append(out, fieldsIn(string(source), path)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path+out[i].Name < out[j].Path+out[j].Name })
	return out, nil
}

// fieldsIn extracts the configuration fields of one source file.
func fieldsIn(source, path string) []configField {
	var out []configField
	for _, match := range tableHeaders(source) {
		// `type TreeConfig struct {` → `TreeConfig`: o prefixo `type` e o
		// sufixo `struct {` são o que faz desta linha uma declaração, e nenhum
		// dos dois é parte do nome que a dívida cita.
		table := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(match), " struct {"), "type ")
		if !configTables[table] {
			continue
		}
		body := bodyOf(source, match)
		for _, field := range fieldPattern.FindAllStringSubmatch(body, -1) {
			out = append(out, configField{Table: table, Name: field[1], Path: path})
		}
	}
	return out
}

// tableHeaders finds the `type X struct {` lines of a source file.
func tableHeaders(source string) []string {
	var out []string
	for line := range strings.SplitSeq(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "type ") && strings.HasSuffix(trimmed, " struct {") {
			out = append(out, trimmed)
		}
	}
	return out
}

// bodyOf returns the text between a struct's opening brace and its closing one.
func bodyOf(source, header string) string {
	start := strings.Index(source, header)
	if start < 0 {
		return ""
	}
	rest := source[start+len(header):]
	if end := strings.Index(rest, "\n}"); end >= 0 {
		return rest[:end]
	}
	return rest
}

// fieldOccurrencesOutsideConfig counts how often each field name is **read** outside
// the configuration package.
//
// A leitura é contada como seletor (`.Shell`) e só em arquivo que importa
// `internal/config`. É lexical de propósito — o mesmo trade-off de `Load`, que
// parseia imports sem verificar tipos: um campo cujo nome colide com o de outro
// tipo, **no mesmo arquivo que importa o config**, ainda passa. A imprecisão está
// escrita aqui para que ninguém acredite que a regra prova mais do que prova.
func fieldOccurrencesOutsideConfig(root string) (map[string]int, error) {
	counts := map[string]int{}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return skipDirectory(root, path, entry.Name())
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		if strings.HasPrefix(filepath.ToSlash(relative), "internal/config/") {
			return nil
		}

		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		// Só conta o arquivo que **importa** o config. Sem isso, `Region.Width`
		// em internal/tui/layout pareceria um leitor de `TreeConfig.Width`, e a
		// regra ficaria silenciosa com um campo morto — que é exatamente o
		// defeito que ela existe para achar.
		if !strings.Contains(string(source), configImport) {
			return nil
		}
		for _, selector := range selectorPattern.FindAllStringSubmatch(string(source), -1) {
			counts[selector[1]]++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return counts, nil
}

// hasTable reports whether the field list has any field of a table.
func hasTable(fields []configField, table string) bool {
	for _, field := range fields {
		if field.Table == table {
			return true
		}
	}
	return false
}

// contains reports whether the field list has this table.name pair.
func contains(fields []configField, key string) bool {
	for _, field := range fields {
		if field.Table+"."+field.Name == key {
			return true
		}
	}
	return false
}
