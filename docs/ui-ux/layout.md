# Layout e classes de largura

## Classes

| Classe | Largura | O que muda |
|---|---|---|
| `compact` | < 80 | A árvore entra sobreposta em vez de lado a lado; a statusline encurta rótulos; o painel de terminal ocupa a largura toda |
| `standard` | 80–119 | Layout completo: árvore, editor e statusline lado a lado |
| `wide` | ≥ 120 | Como `standard`, com largura extra para a árvore e para a coluna de diagnóstico |

Os limites são **inclusivos embaixo**: 80 é `standard`, 120 é `wide`.

## Regras

1. **A largura é decidida uma vez por frame**, a partir do tamanho que o runtime
   entrega. Nenhuma superfície recalcula a classe por conta própria.
2. **Nada é cortado em silêncio.** O que não cabe é truncado com marcador
   visível; um rótulo que some sem sinal é indistinguível de um rótulo que nunca
   existiu.
3. **Largura é conservadora.** Caractere de largura ambígua é tratado como largo.
   Errar para largo custa uma coluna; errar para estreito desalinha a linha
   inteira.
4. **Zero é um tamanho válido.** Um terminal ainda não medido não pode fazer o
   editor entrar em pânico nem dividir por zero.

## Quebra de linha e linha atual

O wrap é **ligado por padrão** (`soft_wrap = false` desliga), como no Zed. Uma
linha de 121 caracteres num terminal de 60 é o caso comum de código, e cortá-la
sem caminho para o resto é pior que qualquer outro defeito de tela.

| Regra | Verificada por |
|---|---|
| Uma linha visual **nunca** passa da largura — exceto quando um único cluster é mais largo que a tela, e partir um caractere largo é pior | `TestWrapNeverExceedsTheWidth` |
| A quebra é por **palavra** quando dá, e por **cluster** quando a palavra não cabe (URL, caminho, identificador) | `TestWrapBreaksByWordWhenItCan`, `TestWrapBreaksInsideAWordThatCannotFit` |
| O espaço que cai na fronteira é **consumido**: a linha reassume a largura inteira | `TestWrapTilesTheLineExactly` |
| O número da linha aparece **uma vez**, e não em cada linha de tela | `TestTheLineNumberIsShownOncePerLine` |
| A janela segue o caret em **linhas visuais**, não em linhas do documento | `TestTheCaretStaysVisibleOnAWrappedLine` |
| O caret numa linha vazia e no fim de uma linha quebrada **aparece** | `TestTheCaretOnAnEmptyLineIsStillMarked` |
| Seleção e realce atravessam a quebra: pintam as linhas de tela que cruzam, e só elas | `TestTheSelectionCrossesWrappedRows`, `TestRolesFollowTheWrappedRow` |
| Toda linha de tela mantém a largura **exata** | `TestEveryRowKeepsTheExactWidth` |

### A linha atual tem duas pistas

1. **O marcador `>` na gutter** — texto, e portanto a pista que sobrevive a um
   terminal sem cor.
2. **O fundo do token `line_highlight`** — decoração, e o token vem **vazio** por
   padrão. Um tema que declara `line_highlight` vê a linha do caret com um fundo
   sutil; um tema que não declara não vê nada, e nada se perde.

A segunda pista é um **fundo**, nunca uma frente: um realce de linha atual que
trocasse a cor do texto apagaria o texto que ele deveria destacar. E ela some
inteira no perfil `NoColor` (`TestTheCurrentLineIsNotMarkedWhenTheThemeIsSilent`).

### A conversão mora em um lugar

Offset de **byte**, coluna de **escalar** e posição em **células** são três
unidades, e a quebra precisa das três. `layout.Wrap` devolve cada linha visual com
o intervalo de bytes que cobre, e `layout.RowFor` diz onde um offset cai. Duas
funções, uma para cada direção — `layout.Offset` e `layout.Cells` — e nenhuma
conta repetida dentro da superfície, que é onde ela estava antes do wrap existir.

Os testes de `wrap` pegaram **dois** bugs dessa classe, e os dois eram o que este
documento promete não ter:

- o índice de um `range` sobre fatia de clusters usado como **offset de byte**,
  que cortava caractere multibyte ao meio;
- `RowFor` passando um offset **absoluto** para uma função que recebe a **fatia**,
  dentro da função que existe para impedir a troca de unidade.

### A divergência de default é declarada

O wrap vir ligado é uma **decisão de produto** e não uma paridade: o oráculo
entrega `soft_wrap = false`. Está registrada como **B19** no ledger de paridade, e
`conformance.DivergentByDecision` remove o **caminho** `config.soft_wrap` da
comparação — não a chave `config` inteira, que carrega sete campos que concordam e
continuam sendo comparados.
