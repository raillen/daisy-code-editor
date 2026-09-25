# internal/tui/editorview

O viewport do texto.

**As linhas chegam já extraídas**, não como buffer: a superfície não alcança o
documento, e um viewport não precisa do arquivo inteiro para desenhar uma tela.

**O caret é marcado na gutter** (`>`), não pintado sobre um caractere. Um cursor de
terminal é coisa que o runtime desenha; uma superfície que desenhasse o seu também
mostraria dois. E o marcador é texto, então sobrevive ao terminal sem cor — onde um
caret tingido seria invisível e a posição, desconhecida.

**Numeração de linha com largura derivada**, não fixa: num arquivo de 10 mil linhas
uma gutter fixa desalinharia todo o texto.

**A gutter cede primeiro.** Se ela não deixar largura útil para o texto, ela sai —
uma coluna de números sem texto ao lado não é um editor menor, é um editor quebrado.

**O texto e os papéis dele andam juntos**, num `Line` só: dois arrays alinhados
por índice são um acoplamento que nada verifica, e uma linha que perdesse os
tokens ainda seria uma linha — pintada sem cor, sem nada dizendo por quê.

**A precedência é decidida por célula**, não pela ordem em que os estilos são
aplicados: seleção > resultado da busca > papel. O que o leitor pediu vence o que
o tema escolheu.

**Offset de byte vira coluna de célula dentro da pintura**, e a conversão mora em
`layout.Cells`. `buffer` conta escalares, `layout` conta células, e as duas só são
a mesma distância enquanto todo caractere tem uma célula — uma seleção depois de
um caractere largo era uma coluna curta, em silêncio, só nas linhas que tinham um
caractere largo antes dela.

**Linha sem papel sai sem estilo**, que é o caso comum: um documento sem lexer não
paga uma sequência de escape em cada linha.

**Toda linha volta com largura exata**, e o viewport volta com a altura pedida —
uma tela curta deixaria o que está abaixo subir.
