# Plano de refatoração, saneamento e dependências

**Status:** ativo · **Substitui:** nada · **Relacionado:** `docs/migration/parity-ledger.md`, plano de migração aprovado

Este documento nasceu de uma constatação desconfortável: o gate media frames,
testes e paridade, e **nenhum dos três pergunta se uma tecla faz alguma coisa**.
Uma TUI em que não se fecha o programa, não se navega a árvore e não se abre um
menu passava em todos os nove checks. O que segue é o que precisa mudar para que
isso não volte.

---

## 1. Diagnóstico

### 1.1 Configuração portada e inerte

O padrão se repetiu e ninguém notou, porque nada falha: **configuração é parseada,
testada e não tem consumidor**.

| Config | Papéis | Consumido por | Veredito |
|---|---|---|---|
| `Syntax.*` | **20** | o tema, que expõe **um** (`Comment`) | **19 sem motor.** Não existe realce de sintaxe no Go |
| `Markdown.*` | — | só `config/load.go` | parseado, ninguém age |
| `Terminal.*` | — | só `config/load.go` | idem; não há painel de terminal na TUI |
| `LSP.*` | — | só `config/load.go` | idem; `internal/lsp` existe e a app nunca sobe cliente |
| `Languages.*` | — | **ninguém** | nunca referenciado |

O caso do `Syntax` é o mais grave, e não é um item de M4: o plano colocava
"syntax core (chroma mapping + markdown list continuation)" no **M1**. Falta desde
o primeiro marco.

O `i18n` já esteve nesta lista — portado, completo, sem consumidor — até a
`menubar` passar a lê-lo. É o mesmo defeito, ainda em quatro lugares.

### 1.2 Componentes escritos à mão em paralelo com a biblioteca

O plano nomeava `bubbles/v2` v2.2.1. Duas das três bibliotecas foram adicionadas e
a do meio não — e **nada falhou**: nem build, nem teste, nem gate.

| Componente | Deveria ser | Estado |
|---|---|---|
| `overlay` (palette, which-key, ajuda) | `list` | ✅ substituído |
| `findbar` (campos) | `textinput` | ⏳ à mão |
| `editorview` (scroll) | `viewport` | ⏳ à mão |
| `tree` (painel) | `tree` | ⏳ à mão |
| which-key / ajuda | `key` / `help` | ⏳ à mão |

### 1.3 Três implementações de largura

A classe de bug mais frequente deste projeto — **cinco ocorrências** — é "um valor
medido numa unidade e afirmado noutra". Hoje convivem três medidas:

1. `layout.Width` — minha, `runewidth` com **ambíguo = largo** (conservadora)
2. `bubbles` — `clipperhouse/displaywidth`
3. `x/ansi` — `StringWidth`

Três medidas que podem discordar é o terreno exato do sexto bug. **Consolidar numa
só é pré-requisito** de qualquer trabalho novo de layout.

### 1.4 O que a matriz de golden *não* pode pinar

O campo de filtro do `bubbles/list` **pisca**: a saída depende do relógio, e um
golden passaria numa máquina rápida e falharia numa lenta. Os casos de
sobreposição foram removidos da matriz com o motivo escrito. O mesmo vale para
qualquer componente animado (`spinner`, `progress`, `timer`) — a regra é:
**golden só para o que é função pura do estado.**

---

## 2. Saneamento

Ordem de execução. Cada item é verificável e nenhum depende do seguinte, exceto
onde anotado.

### S1 — Extrair o realce de sintaxe (desbloqueia M4) · **concluído**

Adotar `chroma/v2` e ligar os 20 papéis de `config.Syntax`. É o item de maior
impacto visual e o que torna a configuração honesta.

**Aceite:** um arquivo `.go` aberto mostra tokens coloridos; os 20 papéis têm
consumidor, ou o papel é removido da configuração.

**O que foi entregue:** `internal/syntax` (os 20 papéis, o mapeamento por família
de linguagem e o leitor de `[[languages]]`), `Theme.Syntax(papel)`, a pintura em
`editorview` e o realce assíncrono cacheado por `(documento, versão, caminho)`.

**Desvios, ditos na hora:** o item levou duas coisas que o plano não nomeava — a
conversão byte→célula em `layout.Cells` e a mudança de `editorview.Lines` de
`[]string` para `[]Line`. As duas são consequência de "os 20 papéis têm
consumidor" e nenhuma delas é adiável sem refazer o mesmo trabalho.

**Aproxima para S2:** `layout.Cells` é a única conversão entre offset de byte e
célula que existe hoje. As outras duas medidas (`bubbles`, `x/ansi`) continuam
convivendo, e S2 é o que fecha isso.

### S2 — Consolidar a medida de largura · **concluído**

Escolher **uma** implementação e apontar as outras para ela. A recomendação era
manter a minha como autoridade de medida da aplicação — a regra conservadora é uma
decisão, não um acidente.

**O que foi entregue:** `layout` passou a **delegar** a `clipperhouse/displaywidth`
— a engine que os componentes `bubbles` usam — em vez de manter uma segunda
implementação. Não há mais medidas para comparar: há uma, e a ponte é o ponto
onde ela é configurada. `go-runewidth` saiu da dependência direta.

**Desvio declarado, e ele é o achado:** a recomendação era manter a regra
conservadora (ambíguo = largo). **Ela estava errada, e a medição provou.** A
tabela ambígua do `runewidth` inclui `à á è é ê ì í ò ó ù ú ü` e **não** inclui
`ã õ â ç` — num produto em português, "são" mediava quatro células num terminal
que desenha três. O custo não era "uma coluna": era **cada linha acentuada uma
coluna mais larga que a vizinha**, e esta medição discordando da que os
componentes usam.

| | antes | depois |
|---|---|---|
| `são` | 4 células | 3 |
| `(sem título)` | 1 coluna a mais que o terminal | exato |
| `→ · ± … »` | 2 | 1 |
| `日本語` | 6 | 6 |
| sequência ANSI | removida antes de medir | zero por configuração |

**A segunda metade do S2 era mais funda que a engine:** `Slice`, `Truncate` e
`Cells` mediam **por rune**, e a menor unidade de exibição é o **cluster de
grafema** — acento combinado, emoji com ZWJ e bandeira são um caractere na tela e
vários runes no texto. Cortar no meio de um cluster produz um caractere que o
terminal não sabe desenhar. Os três passaram a percorrer clusters, e foi no
primeiro teste dessa classe que uma comparação de **byte com célula** apareceu
dentro de `Slice` — a mesma troca de unidade, agora dentro da função que existe
para preveni-la.

**Aceite cumprido:** `TestTheThreeMeasuresAgree` compara `layout.Width` com
`displaywidth` e `x/ansi` sobre texto ambíguo, largo, com acento combinado, com
emoji e com ANSI, e exige acordo — não há onde discordar. E os dez golden frames
foram regenerados com o diff lido: uma linha por arquivo, sempre a linha do
`(sem título)` e a linha da mensagem truncada, cada uma **um caractere a menos de
enchimento** — que é o bug corrigido, aparecendo no snapshot.

### S3 — Substituir os quatro componentes restantes · **concluído: 1 aplicado, 3 medidos**

`findbar` → `textinput`; `editorview` → `viewport`; `tree` → `tree`;
which-key/ajuda → `key` + `help`. Um commit por componente, como o do `overlay`.

**Aceite, por componente:** o código manual some, os testes do que deixou de ser
nosso são **apagados** (não adaptados), e o que é nosso ganha teste novo.

#### `findbar` → `textinput` · **feito**

- ✅ O código manual sumiu: `field()`, `tail()`, `appendFind()` e `backspaceFind()`
  não existem mais. Inserção, apagar por rune, placeholder e cauda são do
  componente.
- ✅ Os **dois testes** que mediam cauda e placeholder foram **apagados**. O que
  ficou de nosso ganhou teste: a barra desenha o campo que recebeu, na largura
  exata.
- ✅ Um único caminho escreve a consulta (`setFindQuery`), porque dois
  escritores seriam duas verdades — e um teste que escreve no modelo sem escrever
  no campo mediria uma barra que o usuário nunca vê. O teste do backspace agora
  **digita** em vez de atribuir, e afirma os dois lados.
- **Evidência mais forte do aceite:** os **quatro goldens da barra de busca
  passaram sem regerar**. O quadro saiu byte a byte igual, o que prova que a troca
  não mudou comportamento — e não só que os testes foram adaptados.
- **Duas decisões, ambas forçadas pelo componente:**
  - **Cursor virtual desligado** porque ele pisca, e um golden não pode depender
    do relógio. O custo é dito: **a barra não tem cursor visível**, e consertar é
    posicionar o cursor real do runtime sobre o campo (M4).
  - **Estilos do componente zerados**, não os do tema: o perfil sem cor promete não
    emitir escape, e o componente emitia. `TestNoColourFramesCarryNoEscapeSequences`
    é quem mede.

#### Os três que faltam · **medidos, e nenhum se aplica**

A lista dizia que três componentes ainda estavam escritos à mão. A medição diz
outra coisa, e o item que mais se aproxima de "feito" é o que o próprio plano já
tinha marcado como feito.

##### which-key / ajuda → `key` + `help` · **já estava feito, e o plano se contradiz**

A primeira linha da tabela S3 diz que `overlay (palette, which-key, ajuda)` foi
substituído pelo `list` — **✅ substituído**. A quinta linha lista which-key e
ajuda de novo como "à mão". As duas linhas não podem ser verdadeiras, e a segunda
é a errada: o que sobrou é `bindingItems`, uma projeção de 18 linhas que lê o
keymap e vira `Item{Label, Detail}` para a lista. Não é um componente, é uma
leitura de dados — e a biblioteca que a faria (`help`) desenha duas colunas
estáticas, o que é **pior** para 98 bindings filtráveis do que a lista com scroll.

##### `editorview` → `viewport` · **o aceite não se aplica: não há o que remover**

- O que existe: `component.Window`, **53 linhas** de função pura, com a regra "a
  seleção está sempre visível e a lista não salta enquanto a seleção anda dentro
  do range visível", e com dois usos provados (a árvore e o editor).
- O que **não** existe: qualquer estado de scroll. A composição nunca preencheu um
  campo `Scroll` — ele existia na `View` da árvore, sempre zero, e **foi removido**
  nesta rodada, com a medição anotada no código.
- `viewport` não substituiria código: **adicionaria um recurso** (rolagem própria)
  que o produto não tem, em troca de trocar uma regra testada por outra. Isso é
  item de M4 ("rolagem do editor"), não saneamento.

##### `tree` → `tree` · **o aceite não se aplica: a biblioteca repetiria o modelo**

`bubbles/tree.New(t *Node, width, height)` **possui** a árvore: o nó, o cursor, a
janela e os próprios caracteres de estado (`▼`, `▶`, `→`). Adotá-lo aqui produziria
três defeitos medidos, não um ganho:

1. **Duplicaria o modelo.** `internal/fs` é o que o harness de paridade compara
   com o oráculo Rust, chave `tree` do dump. A biblioteca passaria a ser um segundo
   lugar com a mesma estrutura — a mesma classe de `Find.Query`/campo.
2. **Trazia de volta a classe de bug de largura.** `▼` e `→` são caracteres de
   largura ambígua, que é exatamente o que `component.Row` mede e documenta: a
   largura do `▶` já foi uma das cinco ocorrências desta projeto.
3. **Trazia uma segunda janela.** A `tree` da biblioteca carrega o próprio
   `viewport` — o que o S2 acabou de consolidar em uma medição só.

**S3 fecha com 1 de 4 aplicado e 3 medidos.** O que resta de verdade é o item
manual que ninguém escreveu: a barra de busca sem cursor visível (dívida dita no
S3) e a rolagem do editor (recurso ausente, não código duplicado).

### S4 — Regra de aptidão: configuração sem consumidor · **parcialmente feito, e o resto medido**

Acrescentar a `internal/architecture` uma regra que falhe o build quando um campo
de configuração não tem leitor fora do `config/`. Um mapa explícito de exceções
(com motivo) mantém a regra útil em vez de barulhenta.

**Aceite:** reintroduzir `Languages` sem consumidor **falha** o build. Contraprova
obrigatória, como nas outras sete.

**A regra das teclas foi feita primeiro, e é a que dói.** A mesma ideia, aplicada
ao que o leitor vê: **81 ações têm tecla, 37 são tratadas, 4 são interceptadas
pela TUI e 40 são teclas mortas** — 49% do que o keymap anuncia não faz nada, e
`docs/ui-ux/keymap.md` é *gerado* dessas mesmas bindings.

- `internal/app/command_fitness_test.go` — a regra: toda ação com tecla tem
  tratamento, ou está na dívida nomeada com motivo. A dívida é também levada pelo
  caminho inverso (uma entrada que já tem tratamento quebra o teste), e há uma
  lista separada para a dívida que só a palette alcança.
- `internal/tui/keys_fitness_test.go` — a outra metade: **toda binding anunciada é
  uma tecla que o terminal manda**. Verifica pelo caminho do produto (sintetiza a
  tecla, tira a grafia que o runtime entregaria, pergunta ao keymap).

**Uma correção que este trabalho quase fez, e o ledger teria registerado errado:**
trocar o canônico de `pagedown` para `pgdown` — a grafia que o Bubble Tea emite.
Isso **quebrou a paridade do dump** e, ao investigar, mostrou que `ApplyKey` já faz
`keymap.Parse`, cujos apelidos traduzem `pgdown` → `pagedown`. **Não havia binding
morta por grafia**: a tradução já morava no keymap, e a segunda tradução só criou
uma segunda verdade. A correção foi revertida, e ficou um teste que mede a
propriedade real (toda binding é alcançável pelo caminho do produto) em vez de uma
que media a própria duplicação.

**O que ainda falta do S4:** nada — a regra do enunciado existe agora, como **R8**
em `internal/architecture`. Ela varre as tabelas de `internal/config` e conta as
leituras (seletores `.Campo`) em cada arquivo **que importa o config**; um campo sem
leitor e sem dívida falha o build. A dívida nomeada tem 16 entradas, e a regra
falha quando uma delas ganha leitor (a dívida que já não é dívida) ou quando um
campo novo aparece sem leitor.

**A regra encontrou uma dezesseisésima chave morta que o grep não via:**
`[ui].gutter_width` — a largura da gutter é derivada do número de linhas pela
própria `editorview`, e o campo documenta um valor que nada lê. Entrou na dívida.

**O que a regra não prova, escrito na própria regra:** a contagem é lexical. Um
campo cujo nome colide com o de outro tipo **no mesmo arquivo que importa o config**
passa. É o mesmo trade-off de `Load`, que parseia imports sem verificar tipos, e
está escrito em `config_fitness.go` para que ninguém acredite que ela prova mais do
que prova.

---

## 3. Dependências

### 3.1 Adotar, sem decisão pendente

| Biblioteca | Para | Nota |
|---|---|---|
| `chroma/v2` v2.27.0 | realce de sintaxe | API de token stream. **Custo medido no S1: +4,4 MB no binário** (7,7 → 12,0 MB) — `lexers` embute os ~250 lexers em XML e o `embed.FS` é privado, então um registro curado não é construível de fora. Se o tamanho virar problema no M5, a saída é `go:generate` do registro ou trocar de motor |
| `doublestar/v4` v4.10.2 | globs de busca | substitui o `globAllows` caseiro |
| `go-git/v5/plumbing/format/gitignore` | fecha a divergência **B25** | só o subpacote; o módulo é grande |
| `glamour/v2` | preview Markdown | escolha do plano |
| `golang.org/x/image` v0.46.0 | dimensões de imagem | `image.DecodeConfig` cobre PNG/JPEG/GIF/WebP |
| `fsnotify/fsnotify` v1.10.1 + `bep/debounce` v1.2.1 | reload em disco | `fsnotify` não é recursivo: watch por diretório |
| `teatest/v2` | testes de interação | `x/exp/golden` **já está no árbol**, transitivo |

### 3.2 Exige decisão

**`charmbracelet/x/vt`** — emulador VT real (ledger **B5**). **Não tem versão
marcada**, só pseudo-versão. Decisão: fixar por commit e pinar, aceitando que uma
atualização é manual. Sem isso, o terminal embutido continua sendo o scrollback
com CSI mínimo, que não faz alt-screen nem scrollback real.

**`tree-sitter/go-tree-sitter` + grammars** — **exige CGO**, e o plano fixou
`CGO_ENABLED=0` em todos os alvos. O ledger (B17) já resolveu: *removido, com
substituição por `-tags treesitter`*. Concretamente:

- **padrão:** `chroma` faz o realce, sem CGO
- **opcional:** `-tags treesitter` para realce preciso, documentado no README de build

É a única decisão de stack desta lista, e afeta o release — não o `go.mod`.

### 3.3 Não adotar

`huh/v2` e `wish/v2` (o ch14 do Prumo os nomeia para a TUI **dele**; o Oride não
tem formulários nem servidor SSH), `creativeprojects/go-selfupdate` (roadmap v0.3),
e o `go-git` inteiro — só o subpacote de gitignore.

---

## 4. Waves

### M4 — Paridade de features (em andamento)

**Feito:** find no buffer (modelo + barra + realce + digitação ao vivo), tema com
perfis e degradação, seleção pintada, sessão persistida, sobreposições pelo
`bubbles/list`.

**Falta, na ordem:**

1. **S1–S3** deste documento (realce, largura, componentes) — pré-requisito, não
   paralelo: substituir componentes mexe em tudo que vem depois
2. Painel SCM + diff
3. Splits
4. Multi-cursor com gesto
5. Preview Markdown (`glamour` + `x/image`)
6. Pickers de tema e locale
7. Motor modal Vim (o keymap é **plano**, sem camadas — isto é trabalho real)
8. Task runner, `:health`, jump list, undo tree, buffer picker, surround
9. Disk watch (`fsnotify`)
10. Bracketed paste (`PasteMsg` nativo do Bubble Tea v2)

**Gate M4:** 100% dos casos `parity/` verdes · ledger sem exceções abertas ·
`--stat`/`--demo`/`--help`/`--version` equivalentes · `docs/migration/parity-report.md`
publicado · **mais o teste de fumaça (S5)**, que o gate original não previa.

### M5 — Cross-platform e release

ConPTY e detecção de shell no Windows, PATHEXT, separadores de caminho,
portabilidade de `/dev/null` e `sh -c` — o `oride-osutil` do Rust já tem o
precedente; o Go precisa do equivalente.

Release: matriz Go de 6–8 alvos **sem split glibc/musl**, ldflags com versão de
fonte única, goreleaser, checksums, e a **regeneração de todo o packaging** — hoje
12 arquivos hardcodam `0.2.0` e o nome `crates/` (PKGBUILD, `.spec`, control,
`build-deb.sh`, template Void, `flake.nix`, `default.nix`, `install.sh`,
`install.ps1`, `release.yml`). Self-update (`--update`/`:update`) entra aqui.

**Gate M5:** matriz CI verde · `CGO_ENABLED=0` em todos os alvos, verificado por
script · instaladores testados em container · `SHA256SUMS` conferido ·
`buildGoModule` no Nix funcionando.

### M6 — Fechamento Prumo-nativo

Docs canônicas reescritas (PT como fonte, EN como **projeção** com
`docs/i18n/translations.json` e frescor por digest — hoje `docs/en/` está abridged:
`design.md` 410→85 linhas). Pack `docs/ui-ux/` completo, contratos em
`docs/contracts/`, `docs/media/records.json` com os golden frames, goals fechados,
`crates/` → tag `v0.2.0-rust-final` + `archive/rust-crates/` (**arquivado, não
deletado** — mesmo precedente do ADR 013 do Prumo).

**Camada de tokens semânticos** (`surface`, `panel`, `border-subtle`, `text`,
`text-muted`, `accent`, `success`, `warning`, `error`, `focus`) — **o Oride não tem
essa camada hoje**: temas são `[ui]`/`[syntax]` com nomes de cor diretos. É trabalho
real do M6, e é o que habilita o tema `no-color` obrigatório: todo token não
primitivo resolve para `none`, e uma paleta que **alega** suprimir cor pintando cor
é **erro**, não aviso.

**Gate M6:** `prumo-agent docs audit`, `docs readiness --goal`, `ui verify` e
`docs release` verdes — o último **recusa o release** com evidência stale, que é o
mecanismo que impede a dissonância de voltar.

---

## 5. As cinco regras que este projeto pagou para aprender

Cada uma nasceu de um defeito concreto. Elas valem para qualquer código daqui, e
três delas são verificáveis por teste.

| O defeito | A regra |
|---|---|
| Um marcador de cursor reservava três células enquanto a função de medida do próprio pacote contava duas | **Se você expõe `Width()`, não pode embarcar constante que assume largura.** Derive da medição |
| Uma cor não resolvia e todo teste passava, porque os testes usavam hexadecimal e o produto usava nomes | **Teste a saída renderizada, não o valor parseado.** Nome não resolvido é no-op silencioso |
| Uma condição de corrida real, porque um teste escrevia no ambiente do processo | **Capacidade se injeta, nunca se lê do ambiente** |
| Um retorno antecipado pulava a pintura inteira quando a gutter estava desligada | **Caminho que parece correto por nunca ser exercitado** |
| Dois overlays distintos, e a regra de captura checada em um só — digitar ia para o documento atrás da barra | **Regra checada num lugar é regra não checada** |

---

## 6. O que mudou no processo

**O gate mede o que ele mede, e nada além.** Nove checks verdes coexistiam com uma
TUI inutilizável. Frame, teste e paridade não perguntam se uma tecla faz algo — e
"marco completo" era verdade sobre a medição e mentira sobre o produto.

Três consequências, todas já incorporadas acima:

1. **Teste de fumaça por wave** (S5) — caminhos, não frames
2. **Regra de configuração sem consumidor** (S4) — o tipo de defeito que passa por
   build, teste e revisão
3. **Golden só para o que é função pura do estado** (§1.4) — um golden que depende
   do relógio é um teste que falha por motivo alheio ao produto

E uma que não é verificável por máquina: **declarar desvio do plano quando ele
acontece**. O `bubbles` foi omitido em silêncio porque nada falhou. Uma linha no
relatório de marco teria custado nada.
