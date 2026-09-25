# internal/tui/panel

O painel inferior com abas: **Terminal │ Problemas │ Busca**.

**Uma região, três conteúdos.** Antes eram duas superfícies — o terminal embaixo e
a barra de busca acima — disputando as mesmas linhas: abrir a busca fazia o
terminal sumir, e o leitor que queria o terminal o perdia para a tecla que não
era uma troca. Uma região com uma faixa de abas é o modelo do Lapce e do Zed, e
é a razão de `layout.FindBar` ter deixado de existir.

**A faixa come uma linha em todas as abas**, e é por isso que `bottomHeight` pede
o **piso do conteúdo mais a faixa**: pedir duas linhas para uma faixa de uma
devolve um painel com abas e nada dentro, e o leitor vê um menu.

**A aba ativa é marcada em texto** (`▍` mais negrito), nunca só na cor — a regra
do projeto é que nenhum estado é comunicado apenas por cor, e a faixa de abas é
onde o estado é mais lido.

**A contagem de problemas só aparece quando alguém mediu.** Com o LSP desligado, a
aba **diz** que o LSP está desligado em vez de oferecer um "0 problemas", que é
uma saúde que ninguém verificou.
