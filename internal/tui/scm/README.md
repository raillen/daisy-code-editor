# internal/tui/scm

O painel de versionamento, à direita do corpo.

**Uma lista, com a letra.** Cada entrada é um caminho com o badge (`M`, `A`, `?`…)
do `internal/git`, e a seleção é do teclado. A letra é a pista principal e a cor
é a segunda, que é a regra do projeto sobre nunca comunicar estado apenas por
cor — e `Theme.FileStatus` pinta por **intenção** (`warning` para modificado,
`success` para novo), não por nome de git.

**As teclas agem sobre a linha selecionada**, como na referência: `s` adiciona ao
índice, `u` tira, `d` mostra o diff, `r` relê o disco, `Enter` abre o arquivo e
`Esc` devolve o foco ao editor. Uma falha em qualquer uma é o **motivo na barra**,
nunca o silêncio: um `s` que não adiciona e não diz nada é a mesma tecla morta que
este projeto passou a caçar.

**(vazio)** não é erro: é o texto que distingue um repositório limpo de um painel
que não carregou.
