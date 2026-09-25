# docs/contracts

A camada de controle da documentação, lida pelo Prumo.

**Não é um índice de documentos.** É o lugar onde o Prumo aprende o que este
repositório deve, o que ele deve de verdade, e onde cada afirmação canônica
mora. Três arquivos, três perguntas:

| Arquivo | Pergunta que ele responde | Comando |
|---|---|---|
| `bindings.json` | Que documento responde cada contrato de documentação? | `prumo docs audit` |
| `../AUTHORITY_MAP.json` | Este documento é canônico, projeção ou registro? | `prumo docs authority` |
| `../lifecycle.json` | Que versão vale, o que está deprecado, que gate libera release? | `prumo docs version` · `prumo docs release` |

**Ausência de binding significa obrigação, não dispensa.** O campo
`applicability: not-applicable` exige um motivo escrito, porque um contrato que
some em silêncio é um contrato que ninguém decidiu dispensar. Quatro estão
dispensados aqui — `screen-reader`, `target-size`, `zoom-reflow`,
`drag-alternatives` — e cada motivo aponta para o que o produto faz no lugar, não
para o que ele não faz.

**O que o `audit` mede, e o que ele não mede.** O estado de cobertura é
**lexical**: ele procura as frases de `required_knowledge` do contrato dentro do
texto das fontes. O vocabulário dos contratos é inglês e a fonte canônica é
pt-BR, então todo contrato vinculado aparece como `partial` mesmo quando o
conhecimento está lá — `architecture.system` tem "dependency direction" escrito
como "direção das dependências". **`partial` significa "a frase não está", não
"o conhecimento não está".** O caminho para `verified` é semântico
(`prumo docs verify --strict`, com `requirement → claim → evidence`), e é
trabalho do M6.

**Um link quebrado encontrado aqui é drift real, não ruído.** Na primeira
execução eram oito, e todos eram verdade: dois índices apontando para
`ATLAS.md`, que virou `PRUMO.md` no rebranding do Prumo 0.5; a projeção
`docs/en/ui-ux/` apontando para arquivos que nunca foram escritos; e um caminho
relativo quebrado num guia. Um gate que só mede o que foi pedido deixa passar
exatamente isso.

**Um achado que fica é um achado que está sendo dito em voz alta.** Hoje
`prumo docs verify` reporta um `version-policy-drift`: a política de versão
compara a versão do produto com a versão do CLI do Prumo, e o Oride tem eixo de
versão próprio (0.2.0) contra o eixo do framework (0.6.0). Escrever `0.6.0`
deixaria o gate verde e a afirmação falsa; o achado fica, registrado como **P15**
no ledger de paridade. É a mesma regra do projeto: uma moldura que mede o que
ela mede precisa que o relato diga o que ela não mede.
