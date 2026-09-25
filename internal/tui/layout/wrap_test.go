package layout

import (
	"strings"
	"testing"
)

// A quebra de linha é a primeira coisa que faz a largura deixar de ser um número
// e virar uma **decisão**, e é onde mora a classe de bug que este projeto mais
// vezes paga: um valor medido numa unidade e afirmado noutra. Estes testes fixam
// as três medidas que precisam concordar — a célula, o offset de byte e a linha
// visual — e o caso que ninguém pensa: o caret no espaço que a quebra consumiu.

// TestWrapNeverExceedsTheWidth is a invariante que vale mais que a aparência: uma
// linha visual mais larga que a tela empurra o que está à direita, e num editor
// isso é a coluna de status.
func TestWrapNeverExceedsTheWidth(t *testing.T) {
	texts := []string{
		"",
		"curta",
		"uma linha de código bastante comprida para não caber em vinte colunas de jeito nenhum",
		"identificador_muito_longo_que_nao_cabe_em_nenhuma_palavra_aaaa_bbbb",
		"日本語のテキストは折り返す必要があります",
		"https://example.com/um/caminho/muito/longo/que/nao/quebra/em/palavra/aaaa/bbbb",
		"   espaços   no   começo   e   no   fim   ",
		"emoji 👨‍👩‍👧 e acento e á e ç",
	}
	for _, text := range texts {
		// A largura zero não é uma tela: `Wrap` devolve a linha inteira, e quem
		// chama com largura zero não pinta nada. A partir de 1 a invariante vale,
		// com uma exceção declarada: um cluster mais largo que a tela não pode ser
		// partido, e partir um caractere largo é pior do que estourar uma coluna.
		floor := 1
		for _, piece := range clusters(text) {
			if cells := Width(piece.text); cells > floor {
				floor = cells
			}
		}
		for width := 1; width < 25; width++ {
			for _, row := range Wrap(text, width) {
				if got := Width(row.Text); got > width && width >= floor {
					t.Errorf("linha %q com %d células numa tela de %d: %q", text, got, width, row.Text)
				}
			}
		}
	}
}

// TestWrapTilesTheLineExactly is a propriedade que faz a seleção e o realce
// acertarem: as linhas e os espaços consumidos ** ladrilham ** o texto, cada byte
// exatamente uma vez.
//
// Um byte coberto duas vezes pinta duas vezes, e um byte coberto nenhuma vez deixa
// um buraco sem cor. E o ladrilho tem duas peças: as linhas e os espaços que a
// quebra comeu, que são a única coisa que um editor de fato descarta.
func TestWrapTilesTheLineExactly(t *testing.T) {
	for _, text := range []string{
		"uma linha de código bastante comprida para não caber em vinte colunas",
		"palavra-muito-longa-sem-espacos-para-quebrar",
		"espaços   extras   com   espaço   no   meio",
		"日本語 mixed 日本語 text",
		"",
	} {
		for width := range 1 + 18 {
			rows := Wrap(text, width)
			if len(rows) == 0 {
				t.Fatalf("quebra produziu zero linhas para %q", text)
			}
			if rows[0].Start != 0 {
				t.Errorf("a primeira linha começa em %d, e não em 0", rows[0].Start)
			}
			if rows[len(rows)-1].End != len(text) {
				t.Errorf("a última linha termina em %d, e o texto tem %d", rows[len(rows)-1].End, len(text))
			}

			// Ladrilho: linhas + espaços consumidos, sem sobra e sem sobreposição.
			tiled := 0
			for index, row := range rows {
				if row.End-row.Start != len(row.Text) {
					t.Errorf("linha %d: %d bytes declarados para %d bytes de texto",
						index, row.End-row.Start, len(row.Text))
				}
				tiled += row.End - row.Start
				if index == 0 {
					continue
				}
				previous := rows[index-1]
				if row.Start < previous.End {
					t.Errorf("linhas %d e %d se sobrepõem: %d < %d", index-1, index, row.Start, previous.End)
				}
				gap := text[previous.End:row.Start]
				tiled += len(gap)
				if strings.TrimSpace(gap) != "" {
					t.Errorf("a quebra comeu %q, que não é espaço", gap)
				}
			}
			if tiled != len(text) {
				t.Errorf("as linhas e os espaços cobriram %d de %d bytes", tiled, len(text))
			}
		}
	}
}

// TestRowForAgreesWithTheRowItPicks is a conversão, e conversões são o lugar onde
// este projeto mora: para cada linha, cada offset tem que cair na linha que
// `RowFor` diz, na coluna em células que `Cells` diz.
func TestRowForAgreesWithTheRowItPicks(t *testing.T) {
	text := "package main\nfunc processar(entrada string) (saida string, erro error) { return validar(entrada), nil }"
	for _, width := range []int{1, 7, 20, 43, 60, 200} {
		rows := Wrap(text, width)
		for offset := range len(text) + 1 {
			row, column := RowFor(rows, offset)
			if row < 0 || row >= len(rows) {
				t.Fatalf("largura %d offset %d: linha %d fora de %d linhas", width, offset, row, len(rows))
			}
			if column < 0 || column > Width(rows[row].Text) {
				t.Fatalf("largura %d offset %d: coluna %d fora da linha %q (%d células)",
					width, offset, column, rows[row].Text, Width(rows[row].Text))
			}
		}
	}
}

// TestRowForPutsTheCaretAtTheEndOfTheRowItCrossed é o caso que ninguém pensa e
// todo mundo sente: o caret logo depois de um espaço que a quebra consumiu pertence
// à linha **anterior**, no fim. Sem isso, digitar no fim de uma linha larga faz o
// cursor pular uma linha inteira.
func TestRowForPutsTheCaretAtTheEndOfTheRowItCrossed(t *testing.T) {
	const text = "uma palavra longa e outra palavra"
	rows := Wrap(text, 18)
	if len(rows) < 2 {
		t.Fatalf("a linha não quebrou em 18 colunas: %+v", rows)
	}

	// O espaço que a quebra consumiu é o byte `rows[0].End`: ele pertence à
	// primeira linha, no fim dela, que é onde o terminal desenha o cursor.
	if rows[1].Start != rows[0].End+1 {
		t.Fatalf("a consumido não é um espaço de um byte: %+v", rows)
	}
	row, column := RowFor(rows, rows[0].End)
	if row != 0 {
		t.Errorf("o caret no espaço consumido caiu na linha %d, e não na 0", row)
	}
	if column != Width(rows[0].Text) {
		t.Errorf("coluna %d, e o fim da linha é %d", column, Width(rows[0].Text))
	}

	// E o primeiro caractere da linha seguinte já é a linha seguinte: pular uma
	// linha inteira porque o caret passou do fim da anterior é o defeito mais
	// irritante que existe em quebra de linha.
	row, column = RowFor(rows, rows[1].Start)
	if row != 1 || column != 0 {
		t.Errorf("o primeiro caractere da linha 1 caiu na linha %d coluna %d", row, column)
	}
}

// TestWrapBreaksByWordWhenItCan is a preferência, e a mede é que a segunda linha
// começa pela palavra e não pelo meio dela.
func TestWrapBreaksByWordWhenItCan(t *testing.T) {
	rows := Wrap("alpha beta gamma delta", 12)

	if len(rows) < 2 {
		t.Fatalf("não quebrou: %+v", rows)
	}
	if !strings.HasPrefix(rows[1].Text, "gamma") {
		t.Errorf("a segunda linha começou por %q, e a palavra deveria caber inteira", rows[1].Text)
	}
}

// TestWrapBreaksInsideAWordThatCannotFit é o outro lado: uma palavra maior que a
// tela é fato comum, e truncá-la seria pior que quebrá-la.
func TestWrapBreaksInsideAWordThatCannotFit(t *testing.T) {
	const word = "identificador_que_nao_cabe"
	rows := Wrap(word, 8)

	if len(rows) < 2 {
		t.Fatalf("a palavra cabia em %d colunas: %+v", 8, rows)
	}
	for _, row := range rows {
		if strings.Contains(row.Text, " ") {
			t.Errorf("uma palavra quebrada ganhou espaço: %q", row.Text)
		}
	}
}

// TestWrapOfAShortLineIsItself: a linha que cabe não muda, que é o caso comum e o
// que os goldens fixam.
func TestWrapOfAShortLineIsItself(t *testing.T) {
	const text = "curta"
	rows := Wrap(text, 20)

	if len(rows) != 1 {
		t.Fatalf("uma linha de %d células quebrou em %d", Width(text), len(rows))
	}
	if rows[0].Text != text {
		t.Errorf("o texto mudou: %q", rows[0].Text)
	}
}

// TestWrapWithoutAWidthIsOneRow is a condição de borda honesta: largura zero não é
// uma tela, e devolver zero linhas faria a superfície pintar nada.
func TestWrapWithoutAWidthIsOneRow(t *testing.T) {
	for _, width := range []int{0, -1} {
		rows := Wrap("qualquer coisa", width)
		if len(rows) != 1 || rows[0].Text != "qualquer coisa" {
			t.Errorf("largura %d: %+v", width, rows)
		}
	}
}
