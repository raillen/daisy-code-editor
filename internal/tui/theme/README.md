# internal/tui/theme

Traduz as cores da configuração em estilos.

**É o único lugar que sabe que cor existe.** As superfícies pedem um **papel** —
seleção, gutter, status — e recebem um estilo; trocar a paleta não alcança o
código que desenha.

**O perfil é parâmetro, não ambiente.** Um teste precisa pedir `NoColor` sem
escrever no ambiente do processo, que é estado compartilhado — a mesma regra que
o `bubbletea-v2` enuncia como *capability is injected by option, never by writing
to the process environment*.

**`NoColor` devolve o estilo zero**, que não aplica atributo nenhum. Esse é o
invariante inteiro do modo sem cor, e ele mora num lugar só: um tema que afirma
suprimir cor e pinta uma é um erro, não uma escolha.

**Os tokens semânticos são carregados, não declarados.** `Gutter()` resolve o
fundo por `surface` e a tinta por `text_muted`; `StatusDirty()` pinta de `warning`.
Um tema que declara `text_muted = "#ff0000"` muda a gutter sem tocar em `[ui]` —
e é `TestTheTokenLayerChangesWhatTheProductPaints` que olha a **saída** para
provar. Uma camada que ninguém lê é a mesma configuração inerte que o plano
descreveu do `internal/config` inteiro.

**Um token `none` é o estilo zero**, e não um estilo sem cor: ele não desenha
fundo nem muda a frente, que é o que "não tenho cor" quer dizer.

**`""` e `"reset"` significam "sem cor"**, e ambos são normais nos temas que o
produto entrega. Filtrar aqui evita entregar à biblioteca de estilo um valor que
ela não resolve.

**Um acessor para os vinte papéis**, e não vinte acessores. Os papéis são dado, o
campo que cada um lê é dado, e um `switch` escrito aqui seria um segundo lugar
para um papel faltar. `Theme.Syntax(papel)` delega a leitura ao `internal/syntax`,
que é quem conhece a tabela `[syntax]`; um papel sem cor resolvida devolve o estilo
zero, que não pinta — a mesma promessa que o modo sem cor faz.

**O `chroma` é o motor e o `theme` não sabe disso.** A superfície pede um papel, o
realce entrega um papel, e a cor é resolvida num lugar só.

**Depende de `layout.Width` medir texto com ANSI**, o que é o que permite uma
linha estilizada continuar ocupando exatamente as células que ocupa sem estilo.
