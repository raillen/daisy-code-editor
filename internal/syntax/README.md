# internal/syntax

Dá aos bytes de um documento os papéis que o tema pinta.

**Os vinte papéis são a tabela `[syntax]`** da configuração, e até este pacote
existir dezenove deles não tinham leitor nenhum: eram parseados, testados e nunca
consultados — uma configuração que descreve um produto que não existe. Um papel
aqui é o que um desses campos significa; virar cor é tarefa do `theme`, não
deste.

**O `chroma` é o motor, e o padrão é sem CGO.** O `tree-sitter` exigiria CGO e o
plano fixou `CGO_ENABLED=0` em todos os alvos, então o realce preciso fica como
opção com `-tags treesitter` e o padrão é este.

**O mapeamento é por família de linguagem, e não é vaidade.** O lexer de Markdown
emite `Keyword` para um marcador de lista e para uma citação, e emite `NameTag`
para o texto de um link — onde em HTML esses mesmos tipos são uma tag e um
atributo. Uma tabela única pintaria cada `func` de Go como marcador de lista.
Três colisões, e cada uma tem um papel só dela.

**As linhas entram, o texto não.** O `buffer` é dono da definição de quebra de
linha — `\n`, `\r\n`, `\r` e três separadores Unicode — e uma segunda definição
aqui botaria os tokens de um arquivo CRLF na linha errada sem falhar nada. As
linhas voltam a ser `\n` para ser lexadas e são cortadas de novo nos deslocamentos
que a junção produziu, então o mapa é exato qualquer que seja o fim de linha do
arquivo.

**Bytes, não células.** O deslocamento do buffer é o índice canônico, e a
conversão para célula acontece na borda, como já acontece com a seleção e com a
busca: o modelo guarda bytes, a viewport pinta células.

**`[[languages]]` tem leitor aqui.** As entradas do usuário são consultadas
primeiro, por nome exato e depois por extensão, e o registro do `chroma` responde
pelo resto. Sem isso, a tabela era parseada por ninguém.

**Falha fechada.** Um arquivo cuja linguagem ninguém nomeia é texto puro, e um
lexer que não lê o fonte devolve nenhum token em vez de um erro: um arquivo
malformado ainda é um arquivo que o usuário está editando.

**Custo medido:** o binário foi de 7,7 MB para 12,0 MB. O pacote `lexers` embute os
~250 lexers em XML e o `embed.FS` é privado, então um registro curado não é
construível de fora. O número está anotado em `docs/migration/refactoring-plan.md`
§3.1, e a decisão — se o tamanho pesa no M5 — é do M5.
