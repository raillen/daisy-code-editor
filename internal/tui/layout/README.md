# internal/tui/layout

Geometria da tela, pura.

**Papel:** responder **onde** cada superfície cai, e **como** o texto cabe nela. O
modelo diz o que quer mostrar; este pacote diz onde aquilo aterrissa.

**Puro, e de propósito:** sem terminal, sem Bubble Tea, sem modelo. É o que
permite conferir as regras de `docs/ui-ux/layout.md` em qualquer largura sem
renderizar um frame — e é por isso que as classes de largura têm teste direto.

**Contrato de entrada:** `Wants` é um valor, não quatro parâmetros — três deles
booleanos, que é a aridade que o contrato de clean-code chama de cheiro.

**Regras que o teste guarda:**
- as fronteiras de classe são inclusivas embaixo (80 é `standard`, 120 é `wide`);
- nada é cortado em silêncio — o truncamento marca a perda com `…`;
- largura é medida em **células**, e a medição é **uma só**: a engine do
  `clipperhouse/displaywidth`, que é a que os componentes `bubbles` usam.
  `TestTheThreeMeasuresAgree` compara as três medidas que existiam no projeto
  (`runewidth`, `displaywidth`, `x/ansi`) e exige acordo;
- **caractere ambíguo conta como estreito**, que é o contrário da regra antiga.
  A tabela ambígua do `runewidth` inclui "é" e não inclui "ã": com a regra
  anterior, "são" mediava quatro células num terminal que desenha três, e **cada
  linha acentuada saía uma coluna mais larga** que a vizinha. A devolução está
  em `TestAmbiguousWidthCountsAsNarrow`;
- a menor unidade é o **cluster de grafema**, não o rune: acento combinado, emoji
  com ZWJ e bandeira são um caractere na tela e vários runes no texto. `Slice`,
  `Truncate` e `Cells` percorrem clusters, e cortar no meio de um cluster é um
  defeito, não uma economia;
- `Pad` devolve **exatamente** a largura pedida, porque uma linha curta deixa o
  que está à direita deslizar para a esquerda;
- zero é tamanho válido: um terminal ainda não medido produz regiões vazias, não
  um panic nem largura negativa;
- o editor cede por último — a árvore vira sobreposição antes de espremer o
  editor abaixo do mínimo.

**`Wrap` é a quebra de linha, e ela devolve o intervalo de bytes de cada linha
visual.** O motivo de devolver offsets e não só texto é a conversão: a pintura
precisa saber onde a linha começa para saber onde o caret e a seleção caem, e
recalcular isso a partir do texto seria uma segunda medição.

**`Offset` e `Cells` são as duas direções, e `RowFor` é a conversão completa.**
Offset de byte, coluna de escalar e posição em células são três unidades, e a
quebra precisa das três. Os testes dessa área pegaram dois bugs — um índice de
`range` usado como offset de byte, que cortava caractere multibyte, e um offset
absoluto passado para uma função que recebe a fatia — e os dois eram exatamente o
que a regra do projeto diz para não existir.

**`Cells` é a única conversão de byte para célula.** O documento conta escalares e
bytes; o layout conta células; e os dois só são a mesma distância enquanto todo
caractere tem uma célula de largura. Existe porque seleção, resultado de busca e
papel de sintaxe chegam como coluna de escalar e são pintados como célula — uma
conversão implícita em cada um daria três medidas para discordar, que é a classe
de bug mais frequente deste projeto. Um offset que cai dentro de um cluster
arredonda para o começo dele.

**Dependência:** `mattn/go-runewidth`, já presente via lipgloss. É medida de
célula, não renderização.
