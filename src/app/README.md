# Application Layer (`src/app`)

A camada **Application** atua como o orquestrador central do Daisy Code Editor, coordenando o estado da aplicação, despacho de comandos orientados a dados e roteamento de entradas do usuário.

## Módulos

- **`application.ts`**: Container de estado reativo e fachada da aplicação. Mantém documentos abertos, abas ativas, layouts de split, visibilidade de painéis (árvore, terminal, Git), overlays ativos, integração com LSP e histórico de navegação (jump stack).
- **`commands.ts`**: Tabela canônica de ações (`Record<ActionId, Command>`). Implementa o padrão Command-Query Separation (CQS) e executa operações atômicas delegando para `core` e `services`.
- **`input.ts`**: Processamento e roteamento de eventos de teclado, detecção de acordes (`Ctrl+...`, `Alt+...`), navegação na barra de menus e despacho para ações.
- **`menus.ts`**: Estrutura descritiva dos menus da barra de ferramentas (`File`, `Edit`, `View`, `Settings`, `Go`, `Git`, `Help`) associando entradas a seus respectivos `ActionId`s.
- **`vim.ts`**: Máquina de estados modal para emulação de comandos Vim (Normal, Insert, Visual), movimentos de texto (`h`, `j`, `k`, `l`, `w`, `b`, `gg`, `G`) e operações (`d`, `y`, `p`).

## Invariantes Arquiteturais

1. **Ações como Dados**: Toda ação do usuário ou atalho dispara um `ActionId` registrado formalmente, desacoplando a camada de UI da lógica de execução.
2. **Isolamento de UI**: Nenhum módulo de `src/app` depende ou importa da camada `src/ui`. A UI apenas consome a API e o estado expostos por `Application`.
3. **Desacoplamento de Tarefas Pesadas**: Operações custosas (ex: fuzzy search em grandes diretórios) são delegadas a workers dedicados (`src/workers/fuzzy.ts`).
