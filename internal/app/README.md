# internal/app

O modelo. Estado e despacho, sem apresentação.

**Papel:** ser a **única** implementação do comportamento do editor. A TUI
renderiza daqui; o harness de conformidade compara daqui. Não existem dois
caminhos, então uma correção aparece nos dois.

**Contrato com quem consome**
- `New(...)` monta o estado; `ApplyKey`/`ApplyText`/`Apply`/`ApplySearch` mudam.
- `DumpState()` é o estado observável serializado — `SchemaVersion` **congelado
  em 1**, porque é comparado byte a byte contra o oráculo Rust.
- Nenhuma decisão de cor, geometria ou widget mora aqui.

**Não importa `internal/tui` nem `conformance`** (R2). O core desconhece a camada
visual — `clean-code-contract` §2.

## Despacho

`Apply` é um **lookup**, não um `switch`. A tabela é composta de arquivos por
domínio — `command_file.go`, `command_edit.go`, `command_movement.go`,
`command_selection.go`, `command_view.go`, `command_search.go`,
`command_clipboard.go` e `command_panel.go` — de modo que uma mudança em busca não
colide com uma mudança em movimento, e acrescentar ação é uma entrada em vez de
um braço novo numa função que cresce.

**Uma ação anunciada sem tratamento é uma mentira**, e a tabela de teclas é
*gerada* para `docs/ui-ux/keymap.md`. Por isso `command_fitness_test.go` cobra
que toda ação com tecla tenha tratamento, ou esteja na dívida nomeada com motivo —
nos dois sentidos. A auditoria encontrou 40 dessas; hoje são 28, e cada grupo
pendente diz o que falta (o painel de terminal, o SCM, os splits, o LSP).

```go
type Handler struct {
    NeedsDocument bool
    Apply         func(*App, *editor.Document) error
}
```

`NeedsDocument` é **declarado**, não inferido, porque a palette precisa da mesma
resposta: uma ação que exige documento é uma ação a desabilitar quando não há
documento aberto. Com a informação num lugar só, despacho e palette não podem
discordar. O documento é resolvido uma vez no `Apply`, não em cada handler.

Os 16 movimentos vêm de uma tabela de 8 direções × 2 variantes (`plain`/`extend`),
porque são a mesma operação com o flag de seleção — a verdade do domínio, não uma
coincidência estrutural.

**O que a TUI exige e ainda não existe aqui:** overlay e split como estado real,
viewport/scroll, tamanho de janela e mouse. Ver o plano de M3.
