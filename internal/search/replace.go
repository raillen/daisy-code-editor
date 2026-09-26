package search

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ReplaceSummary counts what a project replace did.
//
// Dois números, e não um: "3 em 2 arquivos" é a informação que o leitor precisa
// para desfazer — ou para desconfiar, quando o número não é o que ele esperava.
type ReplaceSummary struct {
	FilesModified     int
	ReplacementsCount int
}

// ModifieldPaths are the files the replace touched, relative to the root.
type ModifiedPaths = []string

// ReplaceInFile replaces every occurrence in one file, atomically on disk.
//
// A semântica é a do oráculo (`crates/oride-search/src/replace.rs`), e o detalhe
// é literal por padrão: sem regex, `$100` é **texto** e não grupo — que é o que o
// `NoExpand` do Rust faz, e o que Go's `ReplaceAllLiteral` faz aqui.
func ReplaceInFile(path string, query Query, replacement string) (int, error) {
	matcher, err := replaceMatcher(query)
	if err != nil {
		return 0, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	matches := matcher.FindAllStringIndex(string(content), -1)
	if len(matches) == 0 {
		return 0, nil
	}

	modified := replaceAll(string(content), matcher, query, replacement)
	if err := os.WriteFile(path, []byte(modified), 0o644); err != nil {
		return 0, err
	}
	return len(matches), nil
}

// ReplaceInProject replaces every occurrence in every text file the search would
// consider.
//
// O **mesmo** caminhador da busca, com as mesmas regras — ocultos, `gitignore`
// pelo `rg`, os mesmos diretórios ignorados no fallback, e o mesmo filtro de
// glob. Uma substituição que olha arquivos que a busca não olha é uma
// substituição que o leitor não consegue prever, e vice-versa.
func ReplaceInProject(root string, query Query, replacement string) (ReplaceSummary, ModifiedPaths, error) {
	matcher, err := replaceMatcher(query)
	if err != nil {
		return ReplaceSummary{}, nil, err
	}

	var summary ReplaceSummary
	var modified ModifiedPaths
	err = forEachFile(root, query, func(path string) error {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			// Um arquivo ilegível no meio do projeto não é falha do replace:
			// é o mesmo caminho da busca, que pula e segue.
			return nil
		}
		matches := matcher.FindAllStringIndex(string(content), -1)
		if len(matches) == 0 {
			return nil
		}

		substituted := replaceAll(string(content), matcher, query, replacement)
		if writeErr := os.WriteFile(path, []byte(substituted), 0o644); writeErr != nil {
			return writeErr
		}
		summary.FilesModified++
		summary.ReplacementsCount += len(matches)
		modified = append(modified, mustBeRelative(root, path))
		return nil
	})
	if err != nil {
		return summary, modified, err
	}
	return summary, modified, nil
}

// replaceMatcher is the compiled pattern with the empty-pattern refusal, because a
// pattern that matches everything would rewrite the repository.
func replaceMatcher(query Query) (*regexp.Regexp, error) {
	if strings.TrimSpace(query.Text) == "" {
		return nil, ErrEmptyQuery
	}
	return compileMatcher(query)
}

// replaceAll is the substitution with the Rust's own semantics: literal by
// default (`$100` stays text), and group expansion only when the query asked for
// regex — que é `NoExpand` contra `replace_all` do outro lado.
func replaceAll(content string, matcher *regexp.Regexp, query Query, replacement string) string {
	if query.UseRegex {
		return matcher.ReplaceAllString(content, replacement)
	}
	return matcher.ReplaceAllLiteralString(content, replacement)
}

// mustBeRelative is the path relative to the root, with forward slashes — o
// mesmo formato de `Hit.Path`, porque é o mesmo leitor que lê os dois.
func mustBeRelative(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}
