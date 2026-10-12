# Services Layer (`src/services`)

A camada de **Services** provê adaptadores de infraestrutura e subsistemas especializados para o Daisy Code Editor. Ela encapsula a interação com o sistema operacional, processos filhos, sistema de arquivos e serviços externos.

## Módulos

- **`catalogs.ts`**: Dicionários de internacionalização (i18n) em `pt-BR` e `en-US`, temas integrados e lookup de traduções.
- **`config.ts`**: Parser de arquivos TOML de configuração (nível global do usuário e workspace), mesclagem de defaults e watch/reload.
- **`files.ts`**: Leitura e escrita atômica em disco (tmpfile + rename protegendo permissões e symlinks), escaneamento recursivo e proteção contra path traversal.
- **`git.ts`**: Integração com Git CLI (`git status --porcelain`, `diff`, `blame`, stage/commit). Fail-closed em repositórios inválidos ou sem binário.
- **`icons.ts`**: Mapeamento de Nerd Font icons e glifos por extensão e nome de arquivo, com fallback gracioso.
- **`languageCatalog.ts`**: Inteligência offline de linguagens (palavras-chave, funções built-in e documentação contextual para TS, Rust, Go, Python, Ori).
- **`languages.ts`**: Tokenizador de sintaxe para syntax highlighting e delimitadores de comentários (`//`, `#`, `<!-- -->`).
- **`lsp.ts`**: Cliente Language Server Protocol via stdio sobre binários no PATH (`rust-analyzer`, `typescript-language-server`, etc.). Fail-closed sem travar a edição caso o servidor caia ou não exista.
- **`plugins.ts`**: Descoberta e execução isolada de plugins via argv puro (sem invocação de shell interpretado).
- **`preferences.ts`**: Armazenamento e persistência de preferências de usuário e suporte a regras de indentação via `.editorconfig`.
- **`symbols.ts`**: Extração heurística de declarações de símbolos (classes, funções, interfaces) e indexação de palavras de buffer para auto-complete offline.
- **`terminal.ts`**: Gerenciamento de sessões PTY interativas com emulação ANSI/VT.

## Invariantes Arquiteturais

1. **Direção de Dependências**: Módulos em `src/services` dependem exclusivamente de `src/core` e APIs da plataforma (Node/Bun/processos). Nunca importam `src/app` nem `src/ui`.
2. **Fail-Closed**: Falhas de binários no PATH (LSP, Git, PTY) não propagam exceções destrutivas; são traduzidas em estados inativos ou avisos controlados.
3. **Segurança de I/O**: Escritas em disco usam atomicidade para prevenir corrupção em crashes, e navegação de arquivos rejeita traversals fora do root permitido.
