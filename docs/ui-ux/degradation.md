# Degradação de capacidade

Cada capacidade é **declarada no view** ou **ausente com elegância**. Nunca
alternada imperativamente a cada frame — é assim que um terminal fica quebrado
depois de um encerramento abrupto.

## Cor

| Capacidade | O que acontece |
|---|---|
| truecolor | cores exatas do tema |
| 256 | cada cor é reduzida ao índice mais próximo |
| 16 | reduzida à paleta do terminal |
| nenhuma | **todo token não primitivo resolve para nada** |

Um tema `no-color` que pinta qualquer coisa é um erro, não uma escolha. E
**nenhum estado é comunicado só por cor**: o que está tingido também está escrito.
Cor é reforço; nunca é a informação.

## Mouse

Desligado por padrão. Ligado, é estado do modelo e vira `MouseMode` no view — o
runtime cuida de capturar e soltar. Um editor que captura o mouse sem o usuário
pedir rouba a seleção de texto do terminal.

## Keyboard enhancements

Pedidos **apenas quando o programa usa**, e o terminal que não suporta é tratado
como caso normal, não como erro.

Consequência concreta: `Ctrl+Shift+<letra>` é ambíguo. Sem *enhancements*, o
terminal entrega a tecla sem distinguir o Shift. A ambiguidade é resolvida **pelos
campos da própria tecla** — nunca por uma suposição sobre qual terminal está do
outro lado.

## Barra de foco

Reportada como estado do view. Um terminal que a suporta não precisa que o editor
adivinhe quando perdeu o foco.

## A barra de status como superfície de informação

A barra é um lugar de **leitura por posição**, e por isso os campos dela têm uma
ordem: o que se procura por posição fica na borda, o que se procura por leitura
fica à esquerda.

| Posição | Campo | Regra |
|---|---|---|
| esquerda | arquivo + `*` de sujo | o marcador é texto |
| esquerda | **modo** da linguagem | vem do mesmo resolvedor que faz o realce |
| esquerda | foco | o nome da superfície |
| esquerda | mensagem | o status da última ação |
| direita | **branch** + `↑ahead ↓behind` | `●` é texto; sem upstream, sem distâncias |
| direita | **erros** | **só aparece com cliente LSP anexado** |
| direita | **LSP** | `LSP off` só quando o usuário desligou |
| direita, na ponta | `Ln x, Col y` | o valor que se lê por posição |

Duas regras que são o ponto inteiro:

1. **A barra nunca afirma uma saúde que ninguém mediu.** Sem cliente LSP, o número
   de erros **não aparece** — "0 erros" sem servidor é ausência de medição
   vestida de news (`TestTheBarNeverClaimsAHealthNobodyMeasured`).
2. **A lacuna real é dita fora da tela.** `[lsp].enabled` ligado sem cliente é uma
   dívida de produto (M4 sobe o cliente), e ela é do `PROJECT_STATE` e do ledger —
   não um `LSP —` permanente na barra, que faria o leitor parar de ler a barra
   inteira.

A branch é lida num `Cmd`, num tique de **dois segundos**: `git` é um processo, e
um `fork` por tecla seria um `fork` por tecla. `git` fora de um repositório não é
erro — é a barra sem branch, que é a verdade.
