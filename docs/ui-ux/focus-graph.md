# Grafo de foco

O foco é **dado**, não uma cascata de condições. Um grafo declarado permite
responder três perguntas sem ler o código: quem recebe esta tecla, para onde vai o
`Tab`, e se o foco pode parar num lugar invisível.

## Nós

A tabela **é** a ordem do `Tab`: `TestOrderMatchesTheDocumentedGraph` a fixa contra
este documento, porque reordenar aqui muda para onde o `Tab` leva **todo** mundo.

| Nó | Superfície | Pai | Visível quando |
|---|---|---|---|
| `editor` | editor | — (raiz) | sempre |
| `tree` | árvore de projeto | — | `show_tree` |
| `scm` | painel SCM | — | `show_scm` (M4) |
| `terminal` | painel do rodapé (aba Terminal) | — | `bottom.Visible` |
| `menubar` | barra de menus | — | sempre |
| `tabs` | barra de abas | — | sempre |
| `statusbar` | linha de status | — | sempre |

`—` significa nó de primeiro nível: são os alvos do `Tab`.

**Os painéis vêm antes do chrome, e isso é uma decisão medida.** A ordem anterior
era editor → menu → abas → árvore, e o preço era concreto: **quatro `Tab`s** para
alcançar a árvore de um projeto — o caminho mais usado de um editor de código. Com
os painéis primeiro, é **uma** pressão. Menu, abas e status continuam alcançáveis
(no fim da volta, e com o mouse) para quem vive neles pelo teclado.

## Regras

1. **Uma sobreposição ativa captura toda entrada.** Enquanto houver sobreposição,
   nem o nó com foco nem o keymap global são consultados. É o que impede uma tecla
   de editar o buffer por trás de um modal.
2. **Sem sobreposição**, o nó com foco resolve primeiro pelas suas chaves locais
   (setas na árvore, por exemplo) e depois pelo keymap global
   (`docs/ui-ux/keymap.md`).
3. **`Tab` e `Shift+Tab` percorrem os painéis** visíveis na ordem desta tabela, e
   dão a volta. Um painel oculto é pulado: o foco nunca para onde não se vê.
   **Menu, abas e barra de status não entram no ciclo** — são chrome, e chrome é
   alvo de clique. O custo de focá-los pelo teclado era medido: quatro `Tab`s para
   alcançar a árvore, e outras quatro de volta. Eles continuam no grafo (um clique
   neles focusa o editor) e continuam alcançáveis pelo teclado, pela paleta.
4. **O foco nunca fica num nó invisível.** Esconder o painel com foco move o foco
   para o `editor`, em vez de deixá-lo apontando para o nada.
5. **Perder o foco não muda o documento.** Nenhuma transição de foco edita,
   fecha ou salva — foco é navegação.

## Por que a regra 1 é a que importa

A referência Rust resolvia isto por ordem de `if` numa função de 2.211 linhas.
A ordem existia, mas não era verificável: nada impedia um `if` novo de ser
inserido antes dos outros e roubar uma tecla do modal. Como dado, a captura é uma
propriedade do grafo e o teste a afirma diretamente.
