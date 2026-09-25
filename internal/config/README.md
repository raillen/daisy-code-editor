# internal/config

Configuração efetiva, temas e `.editorconfig`.

**Papel:** resolver, em camadas, o que o editor deve usar: defaults → usuário
(`~/.config/oride/`) → projeto (`.oride/`). Depois os overlays.

**Resolve `.editorconfig` com a semântica de "mais próximo vence"** — a
referência Rust tinha o defeito inverso (ledger item B22).

**API para a view:** `Config.UI.*` (cores e largura de gutter) e
`Config.Syntax.*` (20 papéis de token) são dados puros; mapeá-los para o tipo de
estilo do Charm é trabalho da view, não daqui.

**Não importa `keymap` nem `action`** de propósito: `Config.Keys` é
`map[string]string` de ids crus, e `keymap.FromBindings` faz o parse. Isso evita
o ciclo `config ↔ keymap`.

**Gerado em parte.** `keybindings.go` (98 pares) e `themes.go` (10 temas) vêm de
`scripts/gen-tables.py`.

## A camada de tokens

`[tokens]` é a camada **semântica** do tema: cada **intenção** (`surface`, `panel`,
`text`, `text_muted`, `accent`, `success`, `warning`, `error`, `focus`,
`border_subtle`) resolve para uma cor. Um tema que escreve `#6272a4` na gutter
está dizendo *como* e não *o quê*; com a camada, ele diz `text_muted` e a paleta
decide.

**A camada é inerte por padrão, e isso é testado.** Os dez temas vêm do oráculo por
um arquivo gerado, e o quadro deles não pode mudar — `TestTheTokenLayerIsInertByDefault`
compara os tokens herdados com as cores diretas, campo a campo, e os dez goldens
passam sem regerar.

**A mesclagem é campo a campo, e por um motivo.** Um tema de arquivo que declara
`text_muted` e nada mais herda os outros nove. Com uma struct de strings normais,
"não declarado" e "cor vazia" seriam o mesmo valor — e cor vazia é o que o tema
`no-color` afirma. A diferença entre as duas coisas é a razão de `rawTokens` usar
ponteiros.

**`line_highlight` vem vazio por padrão**, e é decoração: a linha atual já é
marcada em texto na gutter. Um tema que declara o token vê a linha do caret com um
fundo sutil; a prova de que o token age está no pacote que pinta, e a prova de que
ele **não** age por omissão está aqui.

**`no-color` é um tema, não um perfil.** O perfil desliga a cor porque o terminal não
sabe fazer; o tema afirma que a paleta não tem cor nenhuma, e a afirmação é
verificável — `TestNoColourThemeReallyPaintsNothing` falha o build se um token dele
pintar. O nome é **"No Color"** (id `no-color`): a grafia americana é a do resto do
produto, e "No Colour" normalizaria para um id que ninguém digitaria.
