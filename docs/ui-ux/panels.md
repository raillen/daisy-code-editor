# Painéis

Um painel, três conteúdos, uma faixa de abas.

## O painel inferior

| Aba | O que mostra | Atalho |
|---|---|---|
| **Terminal** | o shell embutido, com o prompt e a saída | `Ctrl+`` |
| **Problemas** | os diagnósticos do LSP, `arquivo:linha:coluna · mensagem` | — |
| **Busca** | a busca no buffer, com campo de substituição opcional | `Ctrl+F` |

O painel fica **entre o editor e a barra de status** e cresce para cima: abrir a
busca tira linhas do editor em vez de cobrir o texto que está sendo procurado.

## Regras

| Regra | Verificada por |
|---|---|
| **Uma região, três abas.** Terminal, Problemas e Busca não são três superfícies: são três conteúdos de um lugar | `TestTheBottomPanelIsOneRegionForThreeThings` |
| A faixa come **uma linha em todas** as abas, e a altura pedida é o piso do conteúdo **mais** a faixa | `TestComputeBottomPanelTakesFromTheBody`, `TestAPanelShorterThanItsStripIsEmpty` |
| A aba ativa é marcada **em texto** (`▍` + negrito), nunca só na cor | `TestTheActiveTabIsMarkedInText` |
| Toda linha do painel tem a largura **exata** | `TestEveryRowHasTheExactWidth` |
| A contagem de problemas só aparece **quando alguém mediu** | `TestTheCountAppearsOnlyWhenMeasured` |
| A lista de problemas **nomeia a fonte** do dado e diz o que fazer | `TestProblemsNamesItsSource` |
| A PTY só nasce quando a aba **do terminal** está à frente | `TestThePanelNeverSpawnsAShellNobodySees` |
| O foco do terminal é recusado quando outra aba está à frente | `TestTheTerminalKeepsTheKeyboardOnlyWhenVisible` |

## O que a barra **não** faz

Uma lista vazia sem legenda parece programa quebrado, e um `0 problemas` com o LSP
desligado é uma saúde que ninguém mediu. Por isso a aba de problemas **diz**:

| Estado | O que aparece |
|---|---|
| cliente anexado, nenhum erro | `nenhum problema` |
| ligado no `config.toml`, sem cliente | `LSP desligado · cliente não iniciado` |
| desligado no `config.toml` | `LSP desligado · ligue em [lsp].enabled` |

A lacuna do meio é uma **dívida de produto** (o cliente existe em `internal/lsp` e
a aplicação nunca o sobe) e ela é do `PROJECT_STATE` e do ledger.

## Fechar

`Esc` na busca **recolhe o painel** quando a busca era a aba da frente: um painel
que fica vazio depois de `Esc` é um painel que mente sobre o que o programa está
fazendo. `Ctrl+`` fecha a aba do terminal e devolve o foco ao editor.

## A coluna de estado da árvore

Cada arquivo mostra a **letra** do seu estado no git, na coluna da direita:

| Letra | Estado | Token que a pinta |
|---|---|---|
| `M` | modificado | `warning` |
| `A` | novo (staged) | `success` |
| `D` | apagado | `error` |
| `U` | em conflito | `error` |
| `?` | não rastreado | `accent` |
| `R` | renomeado | `accent` |

Duas regras:

1. **A letra é a pista principal e a cor é a segunda.** Um terminal sem cor
   continua mostrando o estado — a regra do projeto é que nenhum estado é
   comunicado apenas por cor (`TestTheBadgeIsALetterAndTheColourIsTheSecondPista`).
2. **A coluna é medida, não colada.** O nome recebe o que sobra e o badge é o
   último caractere da linha; abaixo de quatro células para o nome, **a letra é
   descartada** e fica o nome truncado — o nome é o conteúdo, a letra é informação
   sobre ele (`TestANameNeverDisappearsForABadge`).

O estado é lido do disco **no mesmo tique de dois segundos que lê a branch**: `git`
é um processo, e dois `git` no mesmo tique seriam o dobro do trabalho pelo mesmo
dado. E o mapa é **relativo ao workspace**, porque o `git` reporta a partir da
raiz do repositório e um workspace dentro de um repositório maior traria caminhos
que não casam com nada da árvore — um badge que nunca apareceria, e ninguém saberia
se o defeito era do `git` ou do casamento.
