# Direção da interface

> **Ienção, não contrato.** Este documento diz para onde vamos e o que
> **recusamos** copiar. Ele **não** descreve comportamento: contrato de
> comportamento é `keymap.md`, `focus-graph.md`, `states.md`, `layout.md` e
> `degradation.md`, e cada um tem um teste que o compara com o código. Um
> documento de direção que descrevesse o que o produto faz seria a mesma mentira
> que o ledger já registrou doze vezes.

## A ambição

Um **workbench de IDE dentro de um terminal**: a densidade e a informação do Zed, a
organização em painéis do Lapce, e a resposta instantânea de um programa de
linha de comando. Não um editor de texto com janelas.

Isto **substitui** a inspiração declarada no `docs/design.md` (Micro e Helix).
Micro e Helix continuam sendo a referência de **ergonomia de tecla** — um chord, um
gesto, sem mouse — mas o alvo visual e organizacional passa a ser o par Zed/Lapce.

## O layout alvo

```
┌ menu ───────────────────────────────────────────────────────────┐
├ abas ───────────────────────────────────────────────────────────┤
│ projeto │                      editor                            │
│ (árvore)│                      (soft wrap, papel de sintaxe)     │
│         ├───────────────────────────────────────────────────────┤
│         │ painel inferior: Terminal │ Problemas │ Busca         │
├ status ────────────────────────────────────────────────────────┤
│ main.go* · rust · Ln 42, Col 8 · 2 erros · ●Branch · LSP ✓      │
└──────────────────────────────────────────────────────────────────┘
```

Cinco Moves, e cada um tem **slot no código hoje** — a região existe, o que falta
é o conteúdo:

| Faixa | O que é | Onde já existe |
|---|---|---|
| **projeto** | árvore com estado do git por arquivo | `layout.Regions.Tree`, `internal/fs`, `internal/git` |
| **editor** | soft wrap, papel de sintaxe, linha atual, seleção | `layout.Regions.Editor`, `internal/syntax` |
| **painel inferior** | Terminal │ Problemas │ Busca, com abas | `layout.Regions.Terminal`, `internal/term`, `internal/lsp` |
| **status** | arquivo, modo, posição, erros, branch, LSP | `layout.Regions.StatusBar`, `internal/git` |
| **abas** | arquivo, sujo, fechar por aba | `internal/tui/tabs` |

## O que tomamos do Zed

- **A camada de token no tema.** `surface`, `panel`, `border`, `text`,
  `text-muted`, `accent`, `success`, `warning`, `error`, `focus` — o tema nomeia
  **tokens**, e a resolução de token para cor acontece em um lugar. É o que torna
  `no-color` honesto: um tema que **alega** suprimir cor pintando cor é erro, não
  aviso.
- **Soft wrap ligado por padrão** para código. Linha longa é o caso comum, e cortar
  o fim de uma linha sem caminho para ele é o pior defeito de leitura que um
  editor pode ter.
- **Linha atual marcada**, bracket match, e a **densidade de informação** dele: informação
  máxima por célula, respiro zero.
- **Paleta de comandos como porta de entrada**, não como lista: busca sobre tudo.

## O que tomamos do Lapce

- **Painéis com título e abas**, e o painel inferior como **um** lugar com
  Terminal, Problemas e Busca — em vez de três surfaces disputando as mesmas
  linhas de baixo.
- **Estado do git por arquivo** na árvore, e o SCM como painel com staged e
  unstaged separados.
- **Barra de status que é superfície de informação**, não um rótulo: modo da
  linguagem, contagem de erros, branch, estado do LSP.

## O que recusamos, e por quê

| Não copiamos | Por quê |
|---|---|
| Botão de fechar por aba, cantos arredondados, moldura em volta de cada painel | Custa **células** e quebra em terminais sem suporte. A informação que o botão carrega (aba suja) já cabe em um marcador, e ela já existe. |
| Ícones de arquivo por tipo | Depende de Nerd Font. Onde falta a fonte, o glifo vira caixa — e a leitura piora. Ícone é M5, com detecção de fonte, não antes. |
| Minimap | Em 60 colunas vira 3 colunas de ruído. Uma coluna de diagnósticos vale mais. |
| Mouse como caminho primário | Já é a posição do produto: o teclado é o caminho completo, o mouse é aceleração opcional. |

O critério é um só: **a mediana de um terminal de 80 colunas**. Toda decisão de
layout é julgada por se ainda dá para ler código e o status ao mesmo tempo.

## O que isto **não** é

- Não é um clone. Onde o Zed usa o mouse, o Oride usa chord; onde o Lapce usa
  painel flutuante, o Oride usa sobreposição com captura de tecla.
- Não é um POM. Nenhuma fatia deste trabalho volta a criar "chave sem tratamento" —
  a regra de aptidão (S4) é o que impede a UI/UX de ser uma coleção de promises.

## A ordem que este documento implica

1. **Tokens e `no-color`** — a base. Sem token semântico, toda decisão visual
   seguinte é uma cor solta.
2. **Soft wrap + linha atual** — o que o leitor sente amanhã, e o que faz a
   diferença entre "editor" e "visualizador de arquivo".
3. **Status como superfície** — modo, erros, branch, LSP: quatro dados que já
   existem e nenhum tem onde aparecer.
4. **Painel inferior com abas** — Terminal, Problemas e Busca em um lugar, com
   título, e a sobreposição da busca virando uma aba dele.
5. **Git por arquivo na árvore** — o badge existe em `internal/git` e não aparece.
6. **Splits e SCM** — as features que o layout já tem slot e que são caro.

Cada item ganha o seu contrato no `docs/ui-ux/` **no mesmo commit** que o entrega.
