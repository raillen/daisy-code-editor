# Current Project State

- Project: **Oride Terminal IDE**
- Framework: **Prumo v0.6.0** (Protocolo v3)
- Current phase: **M4 — Paridade de features** (M0–M3 concluídos)
- Current goal: **Migração Rust → Go com paridade diferencial provada**
- Context methodology: **Lean Progressive Context (LPC)**
- Last updated: `2026-09-25T00:00:00Z`

## Plano ativo

Refatoração, saneamento e dependências: [`docs/migration/refactoring-plan.md`](docs/migration/refactoring-plan.md).
Próximos passos de M4, M5 e M6 no mesmo documento.

## M3 — Shell TUI

### M3.0 — Fundamentos verificáveis · **completo**

- ✅ **`internal/architecture`** — o contrato de arquitetura como teste. Sete
  regras (R1–R7) extraídas de `docs/architecture/clean-code-contract.md` §3:
  ausência de ciclos, não importar o consumidor, superfícies de `tui` não
  importarem o modelo, folhas puras permanecerem folhas, nomes de pasta
  proibidos, README por pasta, e o harness não virar dependência de produto.
  Cada regra é dado (`Rule`), então acrescentar uma não significa escrever outro
  teste. **Contraprova:** `TestRulesCanFail` prova que cada regra sabe falhar e
  que sabe silenciar — uma regra que nunca falha não mede nada.
  O R6 obrigou **20 READMEs** que o contrato exigia e o repositório não tinha.
- ✅ **`internal/docdrift`** — contratos de documentação contra o código.
  `docs/ui-ux/keymap.md` é **gerado** das 98 bindings e comparado nos dois
  sentidos: tecla no código e fora do doc, ou no doc e fora do código, falha
  nomeando a diferença.
- ✅ **`docs/ui-ux/`** — o contrato que o gate do M3 exige e que não existia:
  `keymap.md` (canônico, gerado), `focus-graph.md`, `states.md`, `layout.md`
  (classes de largura), `degradation.md` (cor, mouse, keyboard enhancements) e a
  projeção EN apontando para a fonte canônica em vez de duplicá-la.
- ✅ **`docs/architecture/overview.md`** (PT + EN) corrigido: descrevia `Ropey` e
  `Ratatui`/`Crossterm`, nenhum dos três usado pelo produto em Go.

### M3.1 — Registry de comandos · **completo**

**Correção ao plano:** os 63 campos planos são do `App` **Rust**. O `App` em Go
tem 11 campos e 657 linhas — então esta fatia não foi decompor um God struct, e
sim **estruturar antes de crescer**. Criar sub-estados vazios agora seria
abstração especulativa (§1.6); eles nascem na fatia que os faz ter comportamento.

- ✅ O `switch` de 31 braços virou **tabela por domínio** (`command_file.go`,
  `command_edit.go`, `command_movement.go`, `command_selection.go`,
  `command_view.go`), composta em `command.go`. **33 ações** registradas.
- ✅ `NeedsDocument` é declarado, não inferido: a palette precisa da mesma
  resposta, e num lugar só o despacho e a palette não podem discordar.
- ✅ Os 16 movimentos vêm de 8 direções × 2 variantes — mesma operação com flag de
  seleção, não coincidência estrutural.
- ✅ **Gate da fatia verde:** `TestGoMatchesTheOracle` e os 10 casos byte a byte
  inalterados. A refatoração não mudou comportamento observável.
- ✅ Guardas contra regressão silenciosa: a lista das 33 ações que o switch
  atendia, canonicalidade de cada id registrado, e as duas direções de
  `NeedsDocument`.

### M3.2 — Esqueleto Bubble Tea · **completo**
### M3.3 — Superfícies · **completo**

**Nove superfícies**, cada uma num pacote próprio que **não importa `internal/app`**
(regra R3): o composition root monta a view e a superfície pinta. É o que faz uma
superfície ser descartável — apagar a pasta e a linha que a chama.

`layout` (geometria pura, classes de largura) · `component` (janela de scroll e
linha de lista, com **dois** casos comprovados) · `focus` (grafo como dado) ·
`editorview` · `tree` · `tabs` · `statusbar` · `menubar` · `help` e `modal` ficam
para M3.4.

**O `menubar` é o primeiro consumidor que o `internal/i18n` já teve** — os
catálogos estavam completos e sem uso desde o port.

**Quatro regras em todas as superfícies**, porque cada uma já foi defeito nesta
migração: a coluna do cursor é reservada selecionando ou não; nenhum estado é
comunicado só por cor (sujo, aba ativa, menu aberto e linha selecionada são
escritos também); toda linha volta com largura exata; e vazio é distinguível de
quebrado.

**A quinta ocorrência da mesma classe de bug** foi fechada como classe, não como
instância: um valor medido numa unidade e afirmado noutra apareceu quatro vezes
neste trabalho — a largura do `…` no `layout` e no `docdrift`, a largura do `▶` no
`component`, e um teste comparando bytes onde o código media células.

### M3.4 — Sobreposições · **completo**

- ✅ `palette`, `whichkey`, `help` e os modais **pelo componente `list`**, não por
  um overlay escrito à mão. O manual saiu; o que ficou é testado.
- ✅ A regra de captura do grafo está provada nos **dois** overlays — o da TUI e o
  do modelo. Checá-la num só foi um defeito real: abrir a buscaava digitação para
  o documento atrás da barra.
- ⚠️ **Os overlays não entram na matriz de golden**, e o motivo está escrito em
  `golden_test.go`: o campo de filtro do `list` pisca, e a saída depende do
  relógio. Um golden passaria numa máquina rápida e falharia numa lenta.

## M4 — Paridade de features (em andamento)

**Feito:** find no buffer (modelo + barra + realce + digitação ao vivo) · tema com
perfis e degradação · seleção pintada · sessão persistida · sobreposições pelo
`list` · **realce de sintaxe (S1)**.

### S1 — Realce de sintaxe · **completo**

- ✅ **`internal/syntax`** — os vinte papéis da tabela `[syntax]`, que eram
  parseados, testados e **lidos por ninguém** desde o primeiro marco. O motor é o
  `chroma/v2`, sem CGO; o `tree-sitter` continua fora do padrão.
- ✅ O mapeamento é **por família de linguagem**: o lexer de Markdown emite
  `Keyword` para marcador de lista e para citação, e `NameTag` para o texto de um
  link — onde em HTML são tag e atributo. Três colisões, cada uma com um papel só.
- ✅ **`[[languages]]` finalmente tem leitor**: as entradas do usuário respondem
  primeiro, por nome e por extensão, e o registro do `chroma` pelo resto.
- ✅ A pintura mora na borda: `layout.Cells` converte offset de byte em coluna de
  célula, e é a **única** conversão do pacote. Antes, uma seleção depois de um
  caractere largo era uma coluna curta — em silêncio, e só nas linhas com caractere
  largo antes.
- ✅ `editorview.Lines` passou de `[]string` para `[]Line`, com o texto e os papéis
  dele juntos. Dois arrays alinhados por índice são um acoplamento que nada
  verifica.
- ✅ O realce é **assíncrono e cacheado por `(documento, versão, caminho)`**. O
  caminho entra na chave porque renomear muda a linguagem sem mudar um byte da
  versão; a versão porque uma resposta que chega depois da próxima digitação
  descreve um documento que não existe mais.
- **Contraprova:** um `.txt` tem que sair **sem cor no texto**, e um `.go` tem que
  sair com. Sem o par, um frame em que a barra demenu pinta algo satisfaria "há um
  escape" e o motor poderia não estar fazendo nada.

**Desvios declarados** (§6 do plano manda dizer quando acontece): o S1 levou duas
coisas que o plano não nomeava — a conversão byte→célula e a mudança do tipo de
`Lines`. As duas são consequência direta de "os vinte papéis têm consumidor", e
nenhuma delas é adiável sem refazer o mesmo trabalho.

### S2 — Medida de largura · **completo**

- ✅ **Uma medição só.** `layout` delega a `clipperhouse/displaywidth` — a engine
  que os componentes `bubbles` usam. `go-runewidth` saiu da dependência direta.
  Não há três medidas para comparar: há uma, e a ponte é onde ela é configurada.
- ✅ **A regra conservadora estava errada, e a medição provou.** Ambíguo agora
  conta como **estreito**: a tabela ambígua do `runewidth` inclui "é" e não
  inclui "ã", então "são" mediava 4 células num terminal que desenha 3 — e **cada
  linha acentuada saía uma coluna mais larga que a vizinha**. Desvio declarado no
  plano, que recomendava manter a regra antiga.
- ✅ **Cluster de grafema, não rune.** `Slice`, `Truncate` e `Cells` mediam por
  rune; acento combinado, emoji com ZWJ e bandeira são um caractere na tela e
  vários runes no texto. Cortar no meio de um cluster produz um caractere que o
  terminal não sabe desenhar.
- **Contraprova:** `TestTheThreeMeasuresAgree` exige acordo entre as três medidas
  que existiam, sobre texto ambíguo, largo, com acento combinado, com emoji e com
  ANSI. E os **dez golden frames** foram regenerados com o diff lido: uma linha
  por arquivo, sempre a do `(sem título)` ou a da mensagem truncada, cada uma um
  caractere a menos de enchimento — o bug corrigido aparecendo no snapshot.

### S5 — Teste de fumaça · **completo**

- ✅ **`cmd/oride/smoke_test.go`** dirige a sessão inteira como um usuário: abre,
  Tab até a árvore, expande diretório, abre `notas.md`, volta ao editor, digita,
  `Ctrl+F`, busca ao vivo, `Esc`, `Ctrl+Q`. Cada passo afirma **estado**, nunca
  frame.
- ✅ **Vive no pacote `main` de propósito**: a sessão é montada pelo mesmo código
  que o binário executa (configuração mesclada, keymap, árvore, restauração). Um
  teste em `internal/tui` montaria a aplicação à mão e passaria com o
  `openEditor` quebrado.
- **Contraprova:** sem o binding de `Ctrl+Q` o modelo não fecha. E a primeira
  versão do teste tinha uma afirmação errada — "pediu para sair" via `Cmd` no
  retorno de `Update` — que **só começou a falhar depois do S1**: com o realce
  assíncrono, todo `Update` devolve um `Cmd`, e a afirmação media a existência do
  realce. Passar pela razão errada é a mesma classe de um gate que mede o que ele
  mede.

**Ordem trocada, e declarada:** S5 veio antes de S3, contra a numeração do plano.
A rede de proteção tem que existir antes da refatoração, não depois.

### S3 — Componentes da biblioteca · **1 de 4**

- ✅ **`findbar` → `textinput`.** `field()`, `tail()`, `appendFind()` e
  `backspaceFind()` não existem mais: inserção, apagar por rune, placeholder e
  cauda são do componente. A barra recebe o campo **já desenhado**.
- ✅ **Os dois testes** que mediam cauda e placeholder foram **apagados** (mediam a
  biblioteca); o que é nosso ganhou teste.
- ✅ **Um único caminho escreve a consulta** (`setFindQuery`). O teste do
  backspace agora digita em vez de atribuir, e afirma os dois lados — porque
  escrever no modelo sem escrever no campo mediria uma barra que o usuário nunca
  vê.
- **Evidência do aceite:** os **quatro goldens da barra passaram sem regerar** —
  quadro byte a byte igual, o que prova que a troca não mudou comportamento.
- ⚠️ **Duas consequências declaradas:** o cursor virtual do componente **pisca**,
  então foi desligado (um golden não pode depender do relógio) — e a barra de
  busca **não tem cursor visível**, dívida de M4. Os estilos do componente foram
  **zerados**, porque no perfil sem cor ele emitia escapes que o perfil promete não
  emitir.

### S3 — medido: 1 aplicado, 3 sem o que remover · **concluído**

A lista dizia que três componentes ainda estavam escritos à mão. A medição disse
outra coisa:

- **which-key / ajuda: já estava feito, e o plano se contradiz.** A primeira linha
  da tabela S3 marca `overlay (palette, which-key, ajuda)` como ✅ substituído pelo
  `list`; a quinta os lista de novo como "à mão". O que sobrou é
  `bindingItems` — 18 linhas que leem o keymap e viram itens de lista. E
  `bubbles/help` desenha duas colunas estáticas, **pior** para 98 bindings
  filtráveis do que a lista com scroll.
- **`editorview` → `viewport`: não há o que remover.** O que existe é
  `component.Window`, 53 linhas de função pura com a regra "a seleção está sempre
  visível e a lista não salta", e com dois usos provados. **Não existe estado de
  scroll** — o campo `Scroll` da `View` da árvore nunca foi preenchido, e foi
  removido com a medição anotada. `viewport` adicionaria um recurso ausente, não
  removeria duplicação.
- **`tree` → `tree`: a biblioteca repetiria o modelo.** `bubbles/tree.New` **possui**
  a árvore, e adotá-la produziria três defeitos medidos: duplicar `internal/fs` (que
  o harness de paridade compara com o oráculo), trazer de volta `▼ ▶ →` (largura
  ambígua — a mesma classe de bug do `▶` e do `…`) e trazer um segundo `viewport`
  (o que o S2 acabou de consolidar).

### S4 — regra de aptidão das teclas · **feito, e 9 teclas religadas**

- ✅ **A regra:** toda ação com tecla tem tratamento, ou está na dívida nomeada com
  motivo, nos dois sentidos. E toda binding anunciada é uma tecla que o terminal
  manda.
- **O número agora:** 81 ações com tecla · **49 tratadas** · 4 interceptadas pela
  TUI · **28 mortas** (era 37 tratadas e 40 mortas).
- ✅ **13 teclas religadas, nesta ordem:**
  - **Abas** — `next_tab`, `prev_tab`, `new_tab`, `close_tab`. O store já tinha as
    três operações; o que faltava era o despacho. Fechar a última aba abre um
    documento vazio e **diz** que fechou.
  - **Paginação** — `page_up`/`page_down` andam `altura − 1` linhas, que é a regra
    do oráculo. A altura é **injetada** pela composição (`App.EditorHeight`), nunca
    lida do terminal: um teste pede "uma página de dez linhas" sem tocar no
    ambiente.
  - **Área de transferência** — `copy`, `paste`, `cut`, com a regra do oráculo: a
    **seleção, ou a linha inteira** quando não há seleção. `internal/clipboard` é
    buffer de sessão + **OSC 52** (escrita, que sobrevive à saída e funciona sobre
    SSH), sem `fork` de `xclip` — o plano lista `atotto/clipboard` como
    não-candidata. O modelo guarda a sequência e a TUI a escreve, porque o modelo
    não é dono do stdout: uma escape no dump quebraria a paridade byte a byte.
  - **Painéis, foco e recarga** — `focus_toggle_tree_editor` (a árvore aparece
    quando o alvo estava escondido, porque atalho para superfície invisível é
    atalho para nada), `tree_refresh` (relê o disco: um arquivo novo aparece), 
    `toggle_mouse` (a mensagem diz por que o padrão é desligado) e `reload_file`
    (`Document.ReloadFromDisk` mantém o caret, descarta o undo **de propósito** e
    limita o caret ao fim do texto novo — o mesmo invariante do oráculo). Falha de
    recarga é **status, não erro**: recarregar um arquivo que sumiu é situação
    normal.
- **Um achado de teste, e vale mais que o código:** o clipboard é **estado do
  processo**, então dois testes que colam compartilham o buffer e o segundo falha
  por causa do primeiro. Todo teste de cópia agora esvazia o buffer explicitamente,
  com o motivo escrito — o oráculo usa um lock pelo mesmo motivo.

- **Correção que quase entrou, e o ledger teria registrado errado:** trocar o
  canônico `pagedown` → `pgdown` (a grafia do Bubble Tea) quebrou a paridade do
  dump e revelou que `ApplyKey` **já** traduz por `keymap.Parse`. **Não havia
  binding morta por grafia**; a segunda tradução só criaria uma segunda verdade.
  Revertido, e ficou um teste que mede a propriedade real.
- ✅ **A regra do enunciado do S4 também está feita: R8** em
  `internal/architecture`. Ela varre as tabelas de `internal/config`, conta as
  leituras em cada arquivo que **importa** o config, e falha o build quando um
  campo não tem leitor e não está na dívida nomeada (16 entradas). Falha também
  quando a dívida ganha leitor, ou quando um campo novo aparece.
- **A R8 encontrou uma chave morta que o grep não via:** `[ui].gutter_width`. A
  largura da gutter é derivada do número de linhas pela `editorview`, e o campo
  documenta um valor que nada lê.
- **O que a R8 não prova, escrito na própria regra:** a contagem é lexical. Um
  campo cujo nome colide com o de outro tipo, no mesmo arquivo que importa o
  config, passa. É o mesmo trade-off de `Load` (imports sem tipos).

### Item 2 — Pickers e execução de overlay · **completo**

- ✅ **Enter executa a linha destacada.** Era a peça que faltava: a paleta
  **listava** comandos e não rodava nenhum.
- ✅ **A paleta mostra rótulos, não ids.** `i18n.Catalog.Label(actionID)` traduz os
  26 comandos que os menus nomeiam ("Salvar", "Alternar árvore") e devolve o id para
  o resto.
- ✅ **Enter aplica com o filtro aberto também**, interceptado antes da lista: o
  componente gasta o Enter em "aceitar o filtro" (`AcceptWhileFiltering`), e num
  seletor isso é uma tecla jogada fora entre digitar e escolher.
- ✅ **Tecla digitável abre o filtro**, sem o `/` da convenção de lista, e o texto
  entra **semeado** — a primeira letra de um filtro recém-aberto é o que a lista
  ainda não viu, e mandá-la pelo `Update` depois de trocar o estado a perde.
- ✅ **Os dois seletores**: `select_theme` lista os temas do registro com o ativo
  marcado **em texto**, e guarda o nome **normalizado** (aplicar "Tokyo Night"
  resolveria um tema que não existe, e o sintoma é um seletor que "funciona" e não
  muda nada). `select_locale` recarrega o catálogo.
- ✅ **Os nomes da sobreposição saem do modelo** (`app.OverlayThemePicker` etc.) e a
  TUI deriva os dela — duas cópias da mesma string em dois pacotes é a divergência
  esperando o dia.
- ⚠️ **Dívida que fica, e é de forma:** o dump do overlay no oráculo é um union rico
  (`ThemePicker{query, selected, themes, locales}`) e o Go emite uma string. Nenhum
  caso de paridade exercita os seletores, então a suíte está verde — e quando um
  caso existir, a forma do dump tem de acompanhar.

### Item 4 — Painel de terminal · **completo**

- ✅ **Estado no modelo** (`ShowTerminal`, `TerminalHeight`, `TerminalAttached` no
  dump) e quatro ações religadas: `toggle_terminal`, `focus_terminal`,
  `terminal_grow`, `terminal_shrink`. **52 tratadas, 25 mortas.**
- ✅ **`[terminal].shell` e `[terminal].default_height` finalmente têm leitor** — a
  primeira vez que a tabela inteira do terminal faz alguma coisa.
- ✅ **A PTY nasce numa `Cmd`, nunca no modelo.** Um processo criado no caminho do
  `Update` abriria um shell a cada tecla num harness headless; quem cria é a raiz
  de composição, e a resposta volta por mensagem — o mesmo caminho do realce.
- ✅ **A superfície é `internal/tui/terminal`**, no mesmo formato das outras: cauda
  que vence, largura exata, foco marcado em texto, e os três estados distinguíveis
  — iniciando, encerrado, e o erro do shell ao lado.
- ✅ **As teclas vão para o shell**, com a conversão que o shell entende: Enter é
  `\r`, backspace é `DEL`, e as de navegação saem como as sequências CSI que o
  editor de linha do outro lado lê. `Esc` volta para o editor **com ou sem shell**,
  e `Ctrl+C` não é roubada — no shell ela interrompe o que estiver rodando.
- ✅ **Prova de ponta a ponta com um shell de verdade**: o teste cria a PTY, digita
  `echo oride 1` **por teclado** e espera a saída aparecer no painel. Sem shell no
  ambiente, ele pula — e diz que pulou.

**Dois bugs reais que o painel expôs, e que eram de antes dele:**
- **O foco do modelo e a superfície da apresentação podiam discordar.** Uma ação
  muda `App.Focus`; a apresentação só acompanhava na tecla de Tab. Com o painel,
  `Ctrl+`` punha o foco no terminal do modelo e as teclas continuavam no editor —
  o painel segurava o teclado do shell enquanto a barra de status prometia outra
  coisa. Agora `syncSurface` fecha o caminho inverso, **preservando** o passo do Tab
  entre menu, abas e status, que o modelo não distingue.
- **O painel entrava antes da composição da árvore** e levava o prefixo dela: 130
  células numa tela de 100. Ele entra por último, em largura cheia, porque fica
  *abaixo* da árvore e não ao lado.

**Um teste de conformidade deixou de valer, e isso é informação:** ele usava
`toggle_terminal` como exemplo de "ação que o Go não implementa". O painel
ganhou tratamento e o teste começou a falhar — trocar por `split_vertical` deixou a
intenção explícita, com o motivo escrito.

## A direção da interface

[`docs/ui-ux/direction.md`](docs/ui-ux/direction.md) declara a ambição de
**workbench de IDE dentro de um terminal**: densidade do Zed, painéis do Lapce,
ergonomia de tecla do Micro e do Helix. O documento diz também **o que recusamos
copiar** e por quê — botão de fechar por aba, cantos arredondados, moldura por
painel, ícone de arquivo, minimap, mouse como caminho primário — e fixa o critério
de toda decisão de layout: **a mediana de um terminal de 80 colunas**. O
`docs/design.md` foi corrigido, porque ainda declarava a inspiração antiga.

**A ordem que a direção implica, e o que ela destrava:**

1. **Tokens semânticos + tema `no-color`** — a base. `[ui]`/`[syntax]` nomeiam cor
   direta hoje; sem token, cada decisão visual seguinte é uma cor solta e o
   `no-color` continua sendo um perfil em vez de um tema.
2. **Soft wrap + linha atual** — o defeito de leitura mais visível que existe: uma
   linha de 121 caracteres mostra 59 e um `…`, e `Config.SoftWrap` não tem leitor.
3. **Status como superfície de informação** — modo, erros, branch e LSP: quatro
   dados que já existem e nenhum tem onde aparecer.
4. **Painel inferior com abas** — Terminal │ Problemas │ Busca em um lugar, modelo
   do Lapce, usando a região que o layout já reserva.
5. **Git por arquivo na árvore** — o badge existe em `internal/git` e não aparece.
6. Splits e SCM — caro, e é feature, não UI.

Cada item ganha o seu contrato em `docs/ui-ux/` **no mesmo commit** que o entrega:
o documento de contrato vem depois do código, nunca antes.

### Tokens semânticos + tema `no-color` · **completo** (item 1 da direção)

- ✅ **`[tokens]`** com as onze intenções do modelo do Zed (`surface`, `panel`,
  `panel_text`, `border_subtle`, `text`, `text_muted`, `accent`, `success`,
  `warning`, `error`, `focus`), e a **mesclagem é campo a campo** — um tema de
  arquivo que declara `text_muted` herda os outros dez. Com struct de strings
  normais, "não declarado" e "cor vazia" seriam o mesmo valor, e cor vazia é o que
  o tema `no-color` afirma: a diferença entre as duas coisas é a razão dos
  ponteiros.
- ✅ **A camada é carregada, não declarada.** `Gutter()` resolve o fundo por
  `surface` e a tinta por `text_muted`; `Status()` por `panel`/`panel_text`;
  `StatusDirty()` por `warning`. `TestTheTokenLayerChangesWhatTheProductPaints`
  olha a **saída** e prova que declarar um token muda o produto — porque uma
  camada que ninguém lê é a mesma configuração inerte que o plano descreveu do
  `internal/config` inteiro.
- ✅ **Inerte por padrão, e isso é testado.** Os dez temas vêm do oráculo por um
  arquivo gerado; os tokens herdados são comparados com as cores diretas campo a
  campo, e **os dez goldens passaram sem regerar**.
- ✅ **Tema `no-color`**, registrado à mão fora de `themes.go` (que é gerado e
  reescrito pelo `--check`): um tema que o oráculo não tem, porque lá a mesma coisa
  é um perfil do terminal. A diferença é **quem afirma** — o perfil desliga a cor
  porque o terminal não sabe; o tema afirma que a paleta não tem cor nenhuma.
  `TestNoColourThemeReallyPaintsNothing` falha o build se um token dele pintar.
- ⚠️ **A R8 pegou a minha própria mudança**, e está certo: `StatusBG` perdeu o
  leitor quando `Status()` passou a resolver por token. A verdade é que a cor
  continua lida — como **omissão** de um tema sem a camada — e a regra passou a
  conhecer `config.TokenDefaultFields()`, a lista que mora ao lado da função que
  faz a leitura (um campo renomeado quebra a compilação) e que
  `TestTokenDefaultFieldsMatchWhatIsRead` impede de envelhecer. A premissa da regra
  ("lido só dentro do config é lido por ninguém") estava errada depois desta
  mudança, e está corrigida no código e na regra.

### Soft wrap + linha atual · **completo** (item 2 da direção)

- ✅ **O wrap vem ligado por padrão**, como no Zed, e `soft_wrap = false` desliga.
  Uma linha de 121 caracteres num terminal de 60 era **o defeito de leitura mais
  visível do produto**: 59 caracteres e um `…`, sem comando que levasse ao resto.
- ✅ **A janela passou a contar linhas visuais**, e com ela três casos que só
  existem por causa da quebra: o número da linha aparece **uma vez** (e não quatro
  numa linha quebrada), o marcador do caret **acompanha** a linha visual, e
  seleção/realce são **recortados** para a linha de tela antes de pintar.
- ✅ **`layout.Wrap` + `layout.RowFor` + `layout.Offset`**, com as três unidades
  (byte, escalar, célula) convertidas em um lugar só. Os testes dessa área pegaram
  **três bugs** que eram a mesma classe que o projeto já contabiliza: índice de
  `range` como offset de byte (cortava multibyte), offset absoluto em função que
  recebe fatia, e o caret sumindo em linha vazia / pulando no fim de linha
  quebrada.
- ✅ **A linha atual tem duas pistas**: o marcador `>` da gutter (texto, sobrevive
  sem cor) e o fundo do token `line_highlight`, **vazio por padrão** — decoração é
  decoração. Um teste caça o defeito clássico de um estilo feito a partir de nada,
  que é virar preto.
- ✅ **A divergência de default está nomeada (B19)**, e o harness ganhou
  `conformance.DivergentByDecision`: remove o **caminho** `config.soft_wrap`, não a
  chave `config` inteira — tirar a chave pararia a suíte de verificar os outros
  sete campos, que é como uma suíte de paridade vira decorativa.
- ✅ O contrato está em `docs/ui-ux/layout.md`, com **cada regra ligada ao teste que
  a verifica** — no mesmo commit da entrega.

**Próximo, na nova ordem: o status como superfície de informação** — modo da
linguagem, contagem de erros, branch e estado do LSP, que são quatro dados que já
existem no modelo e nenhum tem onde aparecer.

## Camada de documentação do Prumo

O Prumo instalado é o build da `main` com os assets **embutidos** (v0.6.0), e os
gates de governança funcionam a partir deste repositório — o P14 do ledger está
**fechado**, e o procedimento está em [`docs/PRUMO.md`](docs/PRUMO.md).

- ✅ **`docs/contracts/bindings.json`** — 13 contratos vinculados às suas fontes
  reais e 4 dispensados com o motivo escrito. `prumo docs audit` passou a rodar;
  antes ele nem tinha o que ler.
- ✅ **`docs/AUTHORITY_MAP.json`** — 83 documentos classificados: 40 canônicos,
  34 projeções (`docs/en/**`, `docs/guides/en/**`) e 8 registros (`docs/planning`,
  `docs/migration`, ADR). `prumo docs authority` passa limpo.
- ✅ **`docs/lifecycle.json`** — versão, gates de release e a depreciação de
  `docs/en/design.md`, que é uma projeção abreviada da fonte pt-BR.
- ✅ **Oito links quebrados que o gate achou e ninguém tinha visto:** dois índices
  apontando para `ATLAS.md` (virou `PRUMO.md` no rebranding do Prumo 0.5), a
  projeção `docs/en/ui-ux/` apontando para cinco arquivos nunca escritos, e um
  caminho relativo quebrado em `docs/guides/pt/markdown.md`.
- ✅ **`docs/development/testing-strategy.md` e `coding-standards.md` reescritos**:
  os dois nomeavam só `cargo`, `Ratatui`, `Crossterm` e `Ropey` — a estratégia de
  testes descrevia um produto que não existe desde o primeiro marco em Go. Agora
  nomeiam os checks que bloqueiam de verdade, e declaram o que **não** é gate
  (cobertura de 85% não é medida por nada; golden só para o que é função pura do
  estado).
- ⚠️ **Um achado permanece, e é deliberado (P15):** a política de versão do Prumo
  compara a versão do produto com a versão do CLI. Escrever `0.6.0` deixaria o
  gate verde e a afirmação falsa; o `version-policy-drift` fica visível.
- ⚠️ **O `audit` é lexical** e o vocabulário dos contratos é inglês contra uma
  fonte pt-BR: todo contrato vinculado aparece `partial` mesmo quando o
  conhecimento está lá. O caminho para `verified` é semântico e é trabalho do M6.

## Onde o projeto está

A implementação alvo é **Go** (`cmd/`, `internal/`). A implementação **Rust**
(`crates/`) está congelada como **oráculo diferencial**: ela responde o que o
produto faz hoje, e o porte é comparado contra ela sem tolerância. Quando a
paridade for atingida, `crates/` é arquivado em tag.

Paridade **não** é uma afirmação — é um número que o harness produz, em
`conformance/`, e cada divergência intencional está em
[docs/migration/parity-ledger.md](docs/migration/parity-ledger.md).

## Concluído

### M0 — Fundação, Oráculo e Prumo

- ✅ **Oráculo scriptável.** `oride conformance run --case <FILE> --workspace <DIR> --json`
  dá ao Rust um modo de entrada headless determinístico. Antes disso `--headless`
  era um stub que só emitia erro — sem ele, "paridade 100%" não passava de intenção.
- ✅ **Harness de conformance em Go** com duas garantias provadas por teste: o
  oráculo é determinístico (mesmo caso, dois diretórios, relatórios idênticos) e o
  comparador detecta divergência (um comparador que sempre diz "igual" faria a
  paridade subir a 100% sozinha).
- ✅ **Ledger de paridade** com D1–D11 (dissonância doc↔código) e B1–B18 (bugs)
  triados por veredito: *spec vence*, *código vence*, *removido* ou *melhoria*.
- ✅ **Correções aplicadas e guardadas por teste:** `mouse` desligado por default;
  `[editor]` duplicado no exemplo de config; Enter fechando o grupo de undo;
  dependência morta removida; digest de sessão estável (FNV-1a com valor golden
  que o Go precisa reproduzir).
- ✅ **Fundação Go:** módulo, `cmd/oride`, `internal/buildinfo`, `conformance/`,
  CI de três jobs (`rust`, `go`, `conformance`), `CGO_ENABLED=0` verificado a cada
  mudança.
- ✅ **Governança Prumo instalada:** `prumo.json` (Protocolo v3), `docs/PRUMO.md`
  como router, `.prumo/history/`, `project-profile.json`, workforce resolvido
  deterministicamente (14 agents, 58 skills do catálogo + 8 locais, 6 recipes),
  roles de agente geradas dos contratos canônicos, adaptador Antigravity
  recompilado. `prumo validate` e `prumo doctor` passam limpos.
- ✅ **Skills locais** para o que o catálogo não cobre: `bubbletea-v2`,
  `tui-editor-engine`, `conformance-parity`, `doc-drift`, `pty-vt`, `lsp-client`,
  `cross-platform-terminal`, `release-go`. Tudo regerável por
  `scripts/workforce.sh`.
- ✅ **Dissonância documental corrigida:** dos 12 itens fechados no ledger, oito
  eram doc↔código — incluindo uma chave de config que não existe, um comando
  nunca implementado e um default documentado ao contrário do código.

### M1 — Domínio puro (fechado no essencial)

- ✅ **`internal/action`** — 96 ids canônicos gerados do Rust, com guard de drift
  no CI (verificado injetando um typo).
- ✅ **`internal/keymap`** — `Chord` com o código sendo o próprio token canônico,
  o que faz `Parse`/`String` serem inversos por construção. 98 bindings.
- ✅ **`internal/buffer`** — a semântica de índice que o `ropey` dava de graça,
  verificada contra o oráculo em bytes, scalars, linhas, caret e conversão inversa.
- ✅ **`internal/editor`** — seleção, undo com rótulos exatos, documento,
  movimentos, multi-cursor, store. Corrige **B2, B3 e B4** do ledger.
- ✅ **`internal/config`** — modelo, TOML em camadas (padrão ← usuário ← projeto),
  todos os clamps, registry de temas (10 temas, 252 cores, geradas do Rust por
  `scripts/gen-tables.py`, com temas de disco) e resolução de `.editorconfig`.
- ✅ **`internal/i18n`** — catálogos TOML embutidos e de disco, com herança do
  catálogo padrão para traduções parciais.
- ✅ **`internal/fs`** — árvore de projeto: expansão preguiçosa, ordenação,
  filtros, comparada contra o oráculo.
- ✅ **`internal/git`** — status por porcelain `-z`.
- ✅ **`internal/search`** — matcher do buffer: dobra de caso e acento, fronteira
  de palavra, regex, offsets em bytes.

**M1 está completo.** Todo módulo do domínio puro tem testes próprios, e os que
são observáveis no dump são comparados byte a byte contra o oráculo.

### M1-e — O harness diferencial mede de verdade

- ✅ **`internal/app`** — driver headless que executa um caso e produz o mesmo
  dump do oráculo. `TestGoMatchesTheOracle` compara os dois.
- ✅ **Estado comparado:** keymap completo, config efetiva, texto, contagens,
  caret, seleção, cursores extras, rótulos de undo/redo, versão, abas e **árvore**.
- ⏳ **Ainda não produzido** (com o motivo de cada chave, em `NotYetImplemented`):
  `scm`, `find`, `split`, `vim`, `diagnostics`, `lsp_failures`, `status`.
- **5 divergências declaradas**, todas rastreando para B2/B3/B4 — os itens em que
  o ledger diz *spec vence*. O caso `multi-cursor` documenta o B4 ao vivo: o
  oráculo mantém cursores além do fim do buffer e o backspace seguinte não altera
  o texto.

### M2 — Adaptadores (em andamento)

- ✅ **`internal/session`** — persistência com digest FNV-1a 64. Os três valores
  golden do Rust são reproduzidos byte a byte, o que é paridade cross-language
  sem depender do oráculo: um digest divergente faria a sessão de uma
  implementação ser invisível para a outra.
- ✅ **`internal/plugin`** — manifestos `plugin.toml`, descoberta nos dois
  layouts, expansão de `$FILE`/`$DIR`/`$WORKSPACE`, e execução que **não** passa
  por shell: um nome de arquivo com ponto e vírgula é um nome de arquivo, não um
  segundo comando. Um manifesto quebrado é ignorado, não fatal.
- ✅ **`internal/lsp`** — cliente JSON-RPC sobre stdio com framing
  Content-Length. O transporte é um par de streams, **separado do processo**, o
  que deixa o cliente inteiro ser testado por um pipe sem servidor instalado. A
  conversão escalar↔UTF-16 recusa uma posição que parte um par substituto em vez
  de arredondar para o lado errado.
- ✅ **`internal/term`** — terminal PTY embutido (`creack/pty`). O miolo é
  `ApplyChunk`, função pura sobre o scrollback e a coluna do cursor, testada com
  as sequências exatas que zsh e fish emitem. **Não é um emulador de terminal** —
  a referência também não é, e inventar um faria as duas implementações
  mostrarem coisas diferentes.
- ✅ **Busca no projeto** — `rg --json` e fallback em Go.

### 0.2.0 — entregue antes da migração

- i18n dinâmico via catálogos TOML; tutorial de temas; motor modal Vim; task
  runner com substituição de variáveis; diagnóstico `:health`; Markdown rico;
  sync Git; busca e substituição no projeto.

## Próxima ação

- **S2** — consolidar a medida de largura. Três medidas convivem (`layout`,
  `bubbles`, `x/ansi`) e a classe de bug mais frequente deste projeto é um valor
  medido numa unidade e afirmado noutra: cinco vezes. Consolinar é pré-requisito,
  não tarefa paralela.
- **S3** — substituir os quatro componentes escritos à mão: `findbar` →
  `textinput`, `editorview` → `viewport`, `tree` → `tree`, which-key/ajuda → `key`
  + `help`. Um commit por componente, e os testes do que deixou de ser nosso são
  **apagados**, não adaptados.
- **S4 e S5** — a regra de aptidão (configuração sem consumidor falha o build) e o
  teste de fumaça que dirige a TUI como um usuário. O gate do M4 passa a exigir os
  dois.
- **Depois:** SCM + diff, splits, multi-cursor com gesto, preview Markdown,
  pickers de tema e locale, motor modal Vim, task runner, disk watch, bracketed
  paste — na ordem do plano.

## Ordem de recuperação

1. `AGENTS.md` (regras e invariantes do produto).
2. `prumo.json` (manifesto canônico) e `PROJECT_STATE.md` (este arquivo).
3. `docs/PRUMO.md` (router de documentação) — leia só o que a tarefa pede.
4. `docs/migration/parity-ledger.md` antes de mexer em qualquer comportamento.
5. Código: `internal/` (alvo) e `crates/` (oráculo).

Não carregue o repositório inteiro por padrão.
