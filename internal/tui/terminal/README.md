# internal/tui/terminal

O painel do terminal embutido.

**Não é um emulador, e não finge ser.** Desenha as últimas linhas que o shell
escreveu — que é o mesmo que a referência faz. A diferença que importa é o
**estado vazio**: um painel sem shell ainda e um painel cujo shell morreu são
iguais sem isso, e o leitor não distingue "iniciando" de "quebrado".

**O foco é marcado em texto**, na primeira coluna, e nunca só por cor: a regra do
projeto é que nenhum estado vive só na cor, e um painel que só mostra o foco por
tingimento é um painel que quem não vê cor não sabe se está digitando.

**A cauda vence.** Quando o shell imprime mais do que cabe, as linhas antigas
saem pela frente — é o que o leitor está olhando, e o contrário faz o painel
crescer para cima dentro de um espaço fixo.

**Sem promessa de VT.** As linhas chegam **sem sequência de escape**: o painel
pinta texto, e `Term` é uma promessa de emulador que nenhuma das duas
implementações mantém (ledger **B5**).
