# internal/tui/findbar

A barra de busca no buffer.

**A altura é calculada, não fixa.** Revelar o campo de substituição faz a barra
crescer, e o layout precisa saber disso **antes** de dividir a tela — por isso
`Height()` é método, e `layout.Wants` pergunta quantas linhas a barra quer.

**A barra fica acima do terminal e abaixo do editor**, tirando linhas do editor
em vez de cobrir o texto procurado. Uma barra que cobre o resultado que ela mesma
produz é uma barra que atrapalha.

**Os modificadores são escritos, não tingidos** (`[Aa acentos regex]`). Nenhum
estado é comunicado apenas por cor, e um modificador que não se vê é um
modificador sobre o qual se lê o resultado errado.

**O erro de expressão aparece no lugar da contagem.** Um padrão que não compila
não casou nada, então mostrar `0/0` ao lado do erro seria uma mentira útil apenas
para confundir.

**O campo não é nosso.** Inserção, apagar por rune, o placeholder e a cauda da
consulta longa são do componente `textinput` — e a barra recebe a string **já
desenhada**, com a largura exata. Um segundo parágrafo fazendo o mesmo é a
duplicação que o S3 existe para remover. Os dois testes que mediam cauda e
placeholder foram **apagados**, não adaptados: mediam a biblioteca, que os testa
melhor.

**`FieldWidth` é a largura do campo, e um valor só.** A composição diz ao
componente o quanto medir, e a constante é exportada porque dois valores em dois
lugares é um valor que diverge — que foi o que aconteceu com o `…` e com o `▶`
aqui. O componente pede um a menos do que renderiza, porque reserva uma célula
para o cursor: pedir 28 e receber 29 deslocaria as flags da busca.

**O cursor não é desenhado aqui.** O cursor virtual do `textinput` **pisca**, e
uma saída que depende do relógio é uma saída que um golden fixaria por acidente
numa máquina e reprovaria noutra — por isso ele está desligado. O runtime é dono
do cursor do terminal, e uma barra que desenhasse o seu mostraria dois.

**Dívida que essa escolha deixa visível:** a barra de busca **não tem cursor
visível**. Arrumar é posicionar o cursor real do runtime sobre o campo, e é
trabalho de M4.
