# internal/tui

A camada de apresentação, em Bubble Tea v2.

**Papel:** transformar o estado de `internal/app` em tela, e transformar entrada em
mensagens. Nada mais. Nenhuma regra de edição, de layout de texto ou de protocolo
mora aqui — se uma decisão precisa existir quando não há terminal, ela pertence ao
modelo.

**O composition root é o `model.go`.** É o **único** arquivo deste pacote que pode
importar `internal/app`; as superfícies recebem o que precisam como dado. A regra
não é convenção: `R3-superficies-nao-importam-app` em `internal/architecture`
falha o build, e é o que faz uma superfície ser descartável — apagar a pasta e a
linha que a compõe, sem alcançar o modelo.

**Disciplina do runtime** (skill `bubbletea-v2`):
- `View()` é puro e barato: sem leitura de disco, sem subprocesso, sem rede. Ele
  roda a cada frame, e o que for lento pertence a um `Cmd`.
- `Cmd`/`Msg` é a **única** fronteira assíncrona. Uma goroutine alcança o programa
  por `Program.Send`, nunca mutando estado compartilhado.
- `Update` nunca bloqueia.
- Capacidade de terminal é **declarada no view** — alt screen, modo de mouse,
  keyboard enhancements, cursor. Alternar isso imperativamente a cada frame é como
  um terminal fica quebrado depois de um encerramento abrupto.

## Subpacotes

| Pacote | Papel | Importa `app`? |
|---|---|---|
| `layout` | geometria pura: onde cada superfície cai, e como o texto cabe nela | não |
| `theme` | a única coisa que sabe que cor existe:resolve um papel em estilo | não |
| `component` | primitivas compartilhadas, quando a duplicação aparecer medida | não |
| `focus` | o grafo de foco como dado, sem saber de surfaces | não |
| `overlay` | as sobreposições, pelo componente `list` | não |
| `editorview` `tree` `tabs` `statusbar` `menubar` `findbar` | uma superfície cada | não |

Nenhum importa `app` — a regra `R3` falha o build se um deles passar a importar.

**O realce é assíncrono, e a cache é por versão.** Os papéis de sintaxe chegam
numa `Cmd` e são guardados por `(documento, versão, caminho)`: uma tecla não pode
pagar por lexar o arquivo inteiro, e uma resposta que chega depois da próxima
digitação descreve um documento que não existe mais — pintar seria botar as cores
do texto velho no novo. A renomeação entra na chave porque trocar a extensão
muda a linguagem sem mudar um byte da versão.
