# Background Workers (`src/workers`)

O módulo **Workers** provê execução concorrente em threads isoladas para operações intensivas de CPU, garantindo que o loop de eventos e a taxa de quadros da interface TUI nunca sofram atrasos perceptíveis.

## Módulos

- **`fuzzy.ts`**: Worker dedicado à filtragem e ordenação fuzzy de coleções volumosas (por exemplo, listas com mais de 20.000 caminhos de arquivos em grandes monorepos).

## Invariantes Arquiteturais

1. **Protocolo Tipado**: A comunicação entre a thread principal e o worker utiliza contratos estritamente tipados (`FuzzyWorkerRequest` e `FuzzyWorkerResponse`).
2. **Fallback Síncrono Gracioso**: Caso o ambiente restrinja workers de threads (como em determinados contextos restritos ou testes unitários simplificados), o `Application` realiza fallback transparente para o algoritmo de scoring síncrono sem que o usuário perceba falhas.
