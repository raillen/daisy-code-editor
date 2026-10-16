# Demos (VHS) e goldens opt-in

Evidência visual sem travar o CI: as demos são GIFs reproduzíveis via
`bash scripts/record-demos.sh` (nunca no CI bloqueante) e os goldens são
frames de caracteres versionados que só rodam sob flag.

## Demos

```
bash scripts/record-demos.sh   # build + vhs demos/*.tape → demos/out/*.gif
```

- **Requisitos:** `vhs`, `ffmpeg`, `git` e `bun` no PATH.
- **O que sai:** `demos/out/agents.gif` (painel de agentes: menu Exibir →
  prompt no provedor echo → segunda conversa) e `demos/out/scm.gif` (repo
  montado na hora → modal SCM → `s` adiciona ao índice).
- `demos/out/` é gitignored: o versionado é a receita (tapes, fixtures,
  `demos/run.sh`, record script), não o artefato.
- O script isola `HOME`/`XDG_*` (locale/tema determinísticos — sem isso o
  config do usuário vaza para a gravação), limpa os fixtures antes e depois
  (o app persiste `session.toml` ao sair; o fixture-scm ganha um `.git/` na
  hora) e valida as tapes antes de gravar.

### Restrições do terminal VHS (verificadas na prática)

- **Acordes `Ctrl+Shift` não atravessam:** o terminal legado entrega byte
  `0x01` sem o shift (`Ctrl+Shift+A` vira `select_all`). As demos abrem
  painel/modal pelo **menu** (`Alt+X`/`Alt+T` + `Enter`), que despacha as
  mesmas ações. Comportamento real do app, só outro caminho de entrada.
- **`Alt+letra` funciona** (o VHS despacha o modificador via CDP e o parser
  combina ESC+byte colados).
- **`Type` é literal:** `\x1b` e `\n` não são interpretados — chega
  contrabarra, x, 1, b no terminal.
- **Sem `Wait`:** este VHS só tem `Sleep`; boot usa folga generosa (6s).
- **`Hide` como primeiro comando quebra a gravação** ("no frames").
- **`Type@15ms`** acelera linhas longas de setup.
- `Output` é relativo ao **cwd** de quem chama o `vhs` (o record roda da raiz;
  as tapes apontam para `demos/out/`).

## Goldens

`tests/golden.test.ts` + `tests/golden/*.txt` (frames 100x30, nomes de
workspace normalizados para `<root>`/`<ws>`).

```
bun test                                    # pula os 4 goldens (custo zero)
DAISY_GOLDEN=1 bun test tests/golden.test.ts        # compara, falha em 1 célula
DAISY_GOLDEN_UPDATE=1 bun test tests/golden.test.tsx # regrava — revise o diff!
```

Regras: cenário precisa esperar o estado estável (entries do git, turno
idle) antes do frame; matches exatos (`toBe`), sem `toContain`; ao mudar
render propositalmente, regrave e inclua o `.txt` no mesmo commit.
