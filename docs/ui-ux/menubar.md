# A barra de menus

A tabela do Rust em `crates/oride-app/src/menus.rs`: seis colunas, ~48 itens, com
hotkey e atalho. A barra em si só desenha os seis rótulos; a coluna aberta é um
dropdown sobre o corpo.

## Abrir e usar

| Tecla | Efeito |
|---|---|
| `Alt+letra` (`f`, `e`, `v`, `g`, `i`, `h`) | abre a coluna, no item 0 |
| `←` / `→` | troca de coluna, no item 0 |
| `↑` / `↓` | anda pelos itens, dando a volta |
| letra | pula para o primeiro item que começa por ela |
| `Enter` | fecha o menu e executa o item |
| `Esc` | fecha, e só fecha |
| clique no título | abre a coluna |
| clique no item | executa |

Verificado por `TestAltOpensTheSameColumnTheOracleOpens`,
`TestArrowsWalkTheMenuAndEnterRuns`, `TestALetterJumpsToTheFirstItemThatStartsWithIt`,
`TestEscapeClosesTheMenu` e `TestTheMouseOpensAndRunsFromTheMenu`.

## Regras

1. **O menu captura tudo enquanto está aberto.** `Alt`, setas, `Enter` e `Esc`
   pertencem a ele; digitar com um menu aberto editaria o documento por trás, que é
   o defeito que a regra de captura existe para impedir.
2. **Fechar é mexer no modelo.** O estado aberto vive em `App.MenuOpen` e a tela o
   projeta — a mesma regra das sobreposições, e pelo mesmo motivo: fechar na tela
   e continuar aberto no modelo é a sobreposição que não fecha.
3. **Um item sem tratamento diz, e o que ele diz é a verdade.** `Enter` numa ação
   que o modelo não trata escreve o motivo na barra em vez de calar
   (`TestAnUnwiredMenuItemSaysSo`).
4. **O dropdown usa a mesma geometria que pinta.** `menubar.OpenX` diz onde o
   rótulo aberto começa; o dropdown cai logo abaixo, à esquerda dessa posição; a
   linha do item tem a largura da coluna mais larga. E o clique usa **a mesma**
   geometria: uma para pintar e outra para clicar é uma divergência esperando o dia.
5. **O item escolhido é marcado em texto** (`»`) e com a cor do foco: nenhum estado
   vive só na cor, e "qual item vai executar" é o estado que não pode depender dela.
