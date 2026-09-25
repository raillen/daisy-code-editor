# internal/tui/component

Primitivas que mais de uma superfície usa.

**Existe porque a duplicação foi medida, não imaginada.** Rolar a linha
selecionada para dentro da janela e pintar a linha selecionada aparecem na árvore
e no editor — e apareciam, respectivamente, cinco e três vezes na UI Rust. O
contrato §1.6 permite abstrair a partir de **dois** casos comprovados e proíbe com
um só, então nada entra aqui na esperança de que uma terceira superfície queira.

**`Window`** calcula a fatia visível de uma lista. A regra é a de toda lista: a
seleção está sempre visível, e a janela não pula enquanto a seleção se move dentro
do que já aparece. O re-clamp depois de seguir a seleção existe porque segui-la
pode empurrar a janela além do fim, deixando linhas em branco abaixo do último
item.

**`Row`** desenha uma entrada com largura exata, reservando a coluna do cursor
esteja ou não selecionada — selecionar não pode empurrar o texto para o lado, e
uma lista que pula a cada seta é exaustiva de ler.

**`Mark`** é o cursor da linha selecionada, e é **texto, não cor**: nenhum estado
é comunicado só por cor, e uma seleção tingida mas não marcada é invisível para
quem não vê o tom — e para quem está num terminal sem cor.

**`RowWithBadge` desenha a coluna de estado à direita, e ela é medida.** O nome
recebe `largura − cursor − badge` e o badge é o **último** caractere da linha —
colado com um espaço, o badge apareceria em colunas diferentes conforme o nome
cresce ou encolhe, e uma coluna que anda não é coluna.

**Duas colunas que não cabem: some a letra, fica o nome.** Abaixo de quatro células
para o nome, o badge é descartado. O nome é o conteúdo da lista; a letra é
informação sobre ele — e `M` ao lado de um `…` não informa nada.
