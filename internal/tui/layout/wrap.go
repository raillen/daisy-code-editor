package layout

import "strings"

// WrappedRow is one visual row of a wrapped line, with the byte range it covers.
//
// Os offsets são a parte que não é óbvia: a pintura precisa saber **onde** cada
// linha começa para saber onde o caret e a seleção caem dentro dela. Recalcular
// isso a partir do texto seria uma segunda medição — a mesma classe de bug que o
// S2 fechou quando consolidou a largura, e o motivo de a quebra nascer aqui, pura,
// e não dentro da superfície.
type WrappedRow struct {
	Text string
	// Start is the byte offset of the row's first byte within the line.
	Start int
	// End is the byte offset just past the row's last byte.
	End int
}

// Wrap splits a line into the visual rows of `width` cells.
//
// A quebra é por **palavra** quando dá, e por **cluster** quando não dá: palavra
// mais longa que a tela é fato comum (uma URL, um caminho, um identificador), e
// truncá-la seria pior que quebrá-la no meio.
//
// O espaço que cai na fronteira é **consumido** e não aparece em linha nenhuma —
// é o que um editor faz, e é o que faz a linha reassumir a largura inteira em vez
// de começar com um espaço órfão.
func Wrap(text string, width int) []WrappedRow {
	if width <= 0 {
		return []WrappedRow{{Text: text, Start: 0, End: len(text)}}
	}
	if text == "" {
		return []WrappedRow{{Text: "", Start: 0, End: 0}}
	}

	rows := make([]WrappedRow, 0, 2)
	start, used := 0, 0
	// O último espaço que serviria de quebra: onde, quantas células havia antes
	// dele, e quantos bytes ele ocupa. Os três porque a decisão é "a palavra cabe
	// na linha?" — e responder exige a posição, a largura acumulada e o tamanho do
	// que seria consumido.
	breakAt, breakCells, breakLen := -1, 0, 0

	// `piece.start`, e **não** o índice do `range`: o índice é a posição na fatia de
	// clusters, e usá-lo como offset de byte corta caractere multibyte ao meio —
	// que é o defeito que os testes de cobertura de byte pegaram.
	for _, piece := range clusters(text) {
		offset := piece.start
		cells := Width(piece.text)
		if used+cells > width {
			switch {
			case breakAt >= 0 && breakCells <= width:
				// A palavra que vem a seguir cabe: quebra no espaço, e o espaço
				// some — é o que faz a linha reassumir a largura inteira em vez de
				// começar com um espaço órfão.
				rows = append(rows, rowBetween(text, start, breakAt))
				start = breakAt + breakLen
			default:
				// Não há palavra que caiba: quebra no cluster, porque truncar é
				// pior que partir.
				rows = append(rows, rowBetween(text, start, offset))
				start = offset
			}
			used = cellsOf(text[start:offset])
			breakAt, breakCells, breakLen = -1, 0, 0
		}

		if isSpace(piece.text) && used > 0 {
			breakAt, breakCells, breakLen = offset, used, len(piece.text)
		}
		used += cells
	}
	return append(rows, rowBetween(text, start, len(text)))
}

// RowFor is the visual row that holds a byte offset, and the cell column inside it.
//
// Três casos, e o terceiro é o que ninguém pensa e todo mundo sente:
//
//  1. o offset está dentro de uma linha → ela, e a coluna em células;
//  2. o offset está **no espaço consumido** pela quebra → a linha **anterior**, no
//     fim. É onde o cursor é desenhado num terminal, e pular para a linha seguinte
//     quando o caret passou do fim da anterior é o defeito mais irritante que
//     existe em quebra de linha;
//  3. o offset está além do fim → a última linha, no fim.
func RowFor(rows []WrappedRow, offset int) (row, column int) {
	if len(rows) == 0 {
		return 0, 0
	}
	if offset <= 0 {
		return 0, 0
	}
	if offset > rows[len(rows)-1].End {
		last := len(rows) - 1
		return last, Width(rows[last].Text)
	}

	for index, piece := range rows {
		if offset < piece.Start {
			// O offset caiu no espaço que a quebra consumiu: a linha anterior, no
			// fim — que é onde o cursor está.
			previous := index - 1
			if previous < 0 {
				return 0, 0
			}
			return previous, Width(rows[previous].Text)
		}
		if offset <= piece.End {
			// `offset - piece.Start`, e não `offset`: `Cells` recebe a **fatia**, e
			// um offset absoluto sobre uma fatia que começa depois mede a linha
			// errada — a mesma troca de unidade que esta função existe para
			// impedir, cometida dentro dela.
			return index, Cells(piece.Text, offset-piece.Start)
		}
	}
	last := len(rows) - 1
	return last, Width(rows[last].Text)
}

// rowBetween is the row showing text[start:end], measured.
func rowBetween(text string, start, end int) WrappedRow {
	if start > end {
		start, end = end, start
	}
	return WrappedRow{Text: text[start:end], Start: start, End: end}
}

// cellsOf measures a slice of the line in cells.
func cellsOf(text string) int {
	total := 0
	for _, piece := range clusters(text) {
		total += Width(piece.text)
	}
	return total
}

// isSpace reports whether a cluster is a space that may host a break.
func isSpace(text string) bool {
	return strings.TrimSpace(text) == ""
}
