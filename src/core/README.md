# Core Domain (`src/core`)

O módulo **Core** encapsula a lógica essencial de domínio do Daisy Code Editor, isolada de qualquer infraestrutura de I/O, terminal, PTY ou biblioteca de interface gráfica.

## Responsabilidades

- **`document.ts`**: Modelo de buffer de texto imutável/funcional, suporte a multicursor, transações atômicas de edição (`insert`, `delete`, `replace`), árvore de undo/redo histórico, cálculo determinístico de posições/linhas e tratamento de grafemas Unicode via `Intl.Segmenter`.
- **`keymap.ts`**: Catálogo canônico de ações (`ActionId`), normalização e parsing de acordes de teclas (`chordOf`), resolução de atalhos e mapeamento de comandos como dados.
- **`search.ts`**: Algoritmos de busca literal, regex com suporte a retrocompatibilidade de captura (`$0`, `$1`), folding de acentuação (`ignoreAccents`), cálculo de edições de substituição e pontuação difusa (`fuzzyScore`).

## Invariantes Arquiteturais

1. **Zero UI / Zero I/O**: Nenhum arquivo em `src/core` importa módulos de `src/services`, `src/app` ou `src/ui`.
2. **Determinismo Total**: Todas as operações são puras, síncronas e 100% testáveis em memória sem TTY.
3. **Respeito a Grafemas**: Operações de cursor e deleção nunca quebram sequências Unicode combinadas ou emojis.
