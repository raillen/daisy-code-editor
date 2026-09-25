# Padrões de Codificação e Engenharia

A implementação alvo é **Go** (`cmd/`, `internal/`). A implementação **Rust**
(`crates/`) está congelada como oráculo diferencial e é buildada pelo CI ao lado
dela: as ferramentas das duas linhas importam, e uma regra que só vale para a
implementação que não é produto é uma regra que ninguém verifica.

1. **Formatação e estilo**
   - Go: `gofmt -l cmd internal conformance` sem saída, e `go vet ./...` limpo.
   - Go: `staticcheck ./...` limpo no CI.
   - Rust (oráculo): `cargo fmt --all -- --check` e
     `cargo clippy --all-targets -- -D warnings`.
   - Linhas mantidas em até 100 caracteres quando razoável.

2. **Tipagem e erros**
   - Assinaturas públicas tipadas; `error` é do domínio, não `string`.
   - Erros carregam contexto da operação e o caminho de volta (`fmt.Errorf` com
     `%w` quando há causa).
   - `CGO_ENABLED=0` em todos os alvos: o binário que se distribui não pode
     depender de toolchain de C da máquina que o constrói.

3. **Documentação no código**
   - Comentários explicam o *porquê*, nunca o *o quê*.
   - Toda função e tipo público tem comentário de doc, e o comentário registra a
     razão de uma decisão — não a descrição dela.
   - Um teste nomeia o defeito que ele fecha. "Testa o parser" não diz nada;
     "testa o acento em caixa" diz qual bug ele impede de voltar.

4. **Contratos executáveis**
   - Regras de arquitetura e de documentação são **testes**, não prosa: R1–R7 em
     `internal/architecture`, a tabela de teclas em `internal/docdrift`. Cada
     regra tem uma contraprova que prova que ela sabe falhar.
