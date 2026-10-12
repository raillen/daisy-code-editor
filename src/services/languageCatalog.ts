// Static language catalogs for keywords, built-ins, standard libraries and documentation.

export type LanguageCatalogItem = {
  kind: "macro" | "type" | "function" | "keyword" | "constant" | "module" | "snippet"
  detail: string
  doc: string
  snippet?: string
}

export const languageCatalogs: Record<string, Record<string, LanguageCatalogItem>> = {
  rust: {
    "println!": {
      kind: "macro",
      detail: "println!(format, ...)",
      doc: "Imprime texto formatado na saída padrão (stdout) com quebra de linha.",
    },
    "eprintln!": {
      kind: "macro",
      detail: "eprintln!(format, ...)",
      doc: "Imprime texto formatado na saída de erro padrão (stderr) com quebra de linha.",
    },
    "format!": {
      kind: "macro",
      detail: "format!(format, ...) -> String",
      doc: "Cria uma String formatada usando interpolação de argumentos.",
    },
    "vec!": {
      kind: "macro",
      detail: "vec![elements...]",
      doc: "Cria um novo vetor Vec<T> com os elementos especificados.",
    },
    "panic!": {
      kind: "macro",
      detail: "panic!(msg, ...)",
      doc: "Provoca um pânico imediato na thread atual com mensagem de erro.",
    },
    "Vec": {
      kind: "type",
      detail: "struct Vec<T>",
      doc: "Vetor dinâmico contíguo alocado no heap.",
    },
    "String": {
      kind: "type",
      detail: "struct String",
      doc: "String dinâmica UTF-8 mutável alocada no heap.",
    },
    "Option": {
      kind: "type",
      detail: "enum Option<T> { Some(T), None }",
      doc: "Tipo opcional que representa um valor presente (Some) ou ausente (None).",
    },
    "Result": {
      kind: "type",
      detail: "enum Result<T, E> { Ok(T), Err(E) }",
      doc: "Tipo de resultado que representa sucesso (Ok) ou erro (Err).",
    },
    "Some": {
      kind: "constant",
      detail: "Option::Some(T)",
      doc: "Variante de Option que contém um valor.",
    },
    "None": {
      kind: "constant",
      detail: "Option::None",
      doc: "Variante de Option que representa a ausência de valor.",
    },
    "Ok": {
      kind: "constant",
      detail: "Result::Ok(T)",
      doc: "Variante de Result que indica uma operação bem-sucedida.",
    },
    "Err": {
      kind: "constant",
      detail: "Result::Err(E)",
      doc: "Variante de Result que contém o erro retornado.",
    },
    "Box": {
      kind: "type",
      detail: "struct Box<T>",
      doc: "Ponteiro inteligente de alocação única no heap.",
    },
    "Rc": {
      kind: "type",
      detail: "struct Rc<T>",
      doc: "Ponteiro com contagem de referências para thread única.",
    },
    "Arc": {
      kind: "type",
      detail: "struct Arc<T>",
      doc: "Ponteiro seguro entre threads com contagem de referências atômica.",
    },
    "HashMap": {
      kind: "type",
      detail: "struct HashMap<K, V>",
      doc: "Tabela hash que armazena pares chave-valor.",
    },
    "u8": {
      kind: "type",
      detail: "primitive u8",
      doc: "Inteiro de 8 bits sem sinal (0 a 255).",
    },
    "u16": {
      kind: "type",
      detail: "primitive u16",
      doc: "Inteiro de 16 bits sem sinal.",
    },
    "u32": {
      kind: "type",
      detail: "primitive u32",
      doc: "Inteiro de 32 bits sem sinal.",
    },
    "u64": {
      kind: "type",
      detail: "primitive u64",
      doc: "Inteiro de 64 bits sem sinal.",
    },
    "usize": {
      kind: "type",
      detail: "primitive usize",
      doc: "Inteiro sem sinal com tamanho igual ao ponteiro da arquitetura.",
    },
    "i8": {
      kind: "type",
      detail: "primitive i8",
      doc: "Inteiro de 8 bits com sinal (-128 a 127).",
    },
    "i16": {
      kind: "type",
      detail: "primitive i16",
      doc: "Inteiro de 16 bits com sinal.",
    },
    "i32": {
      kind: "type",
      detail: "primitive i32",
      doc: "Inteiro de 32 bits com sinal (tipo padrão de inteiro em Rust).",
    },
    "i64": {
      kind: "type",
      detail: "primitive i64",
      doc: "Inteiro de 64 bits com sinal.",
    },
    "isize": {
      kind: "type",
      detail: "primitive isize",
      doc: "Inteiro com sinal com tamanho igual ao ponteiro da arquitetura.",
    },
    "f32": {
      kind: "type",
      detail: "primitive f32",
      doc: "Ponto flutuante de precisão simples (32 bits, IEEE 754).",
    },
    "f64": {
      kind: "type",
      detail: "primitive f64",
      doc: "Ponto flutuante de precisão dupla (64 bits, IEEE 754).",
    },
    "bool": {
      kind: "type",
      detail: "primitive bool",
      doc: "Tipo booleano com valores true ou false.",
    },
    "char": {
      kind: "type",
      detail: "primitive char",
      doc: "Caractere escalar Unicode de 4 bytes.",
    },
    "str": {
      kind: "type",
      detail: "primitive str",
      doc: "Fatia de string UTF-8 de comprimento fixo (usualmente &str).",
    },
    "fn": {
      kind: "keyword",
      detail: "fn name(args) -> ReturnType",
      doc: "Palavra-chave para declaração de função.",
    },
    "let": {
      kind: "keyword",
      detail: "let [mut] name = expr;",
      doc: "Declaração de variável com inferência de tipo.",
    },
    "mut": {
      kind: "keyword",
      detail: "mut identifier",
      doc: "Modificador que torna a vinculação de variável mutável.",
    },
    "match": {
      kind: "keyword",
      detail: "match expr { pattern => result, ... }",
      doc: "Controle de fluxo baseado em casamento de padrões exaustivo.",
    },
    "struct": {
      kind: "keyword",
      detail: "struct Name { fields... }",
      doc: "Declaração de estrutura de dados com campos nomeados ou tupla.",
    },
    "enum": {
      kind: "keyword",
      detail: "enum Name { Variants... }",
      doc: "Declaração de tipo enumerado (tipos algébricos / soma).",
    },
    "impl": {
      kind: "keyword",
      detail: "impl [Trait for] Type { ... }",
      doc: "Bloco de implementação de métodos ou traits para um tipo.",
    },
    "trait": {
      kind: "keyword",
      detail: "trait Name { ... }",
      doc: "Define uma interface de comportamento compartilhado entre tipos.",
    },
    "async": {
      kind: "keyword",
      detail: "async fn / async { ... }",
      doc: "Declaração de função ou bloco assíncrono retornando um Future.",
    },
    "await": {
      kind: "keyword",
      detail: "future.await",
      doc: "Suspende a execução assíncrona até a conclusão do Future.",
    },
    "pub": {
      kind: "keyword",
      detail: "pub identifier",
      doc: "Modificador de visibilidade pública.",
    },
    "use": {
      kind: "keyword",
      detail: "use path::Item;",
      doc: "Traz itens de outros módulos para o escopo local.",
    },
    "mod": {
      kind: "keyword",
      detail: "mod name;",
      doc: "Declaração de módulo no sistema de arquivos ou inline.",
    },
    "return": {
      kind: "keyword",
      detail: "return [expr];",
      doc: "Retorna explicitamente o valor de uma função.",
    },
    "pfn": {
      kind: "snippet",
      detail: "pub fn name() { ... }",
      doc: "Snippet: Declaração de função pública.",
      snippet: "pub fn $0() {\n    \n}",
    },
    "test": {
      kind: "snippet",
      detail: "#[test] fn test_...()",
      doc: "Snippet: Função de teste unitário Rust.",
      snippet: "#[test]\nfn test_$0() {\n    \n}",
    },
    "pln": {
      kind: "snippet",
      detail: "println!(\"...\")",
      doc: "Snippet: Chamada da macro println! com quebra de linha.",
      snippet: "println!(\"$0\");",
    },
  },
  typescript: {
    "console.log": {
      kind: "function",
      detail: "console.log(...data: any[]): void",
      doc: "Imprime dados na saída padrão (stdout) do console.",
    },
    "console.error": {
      kind: "function",
      detail: "console.error(...data: any[]): void",
      doc: "Imprime mensagens de erro na saída de erro (stderr).",
    },
    "Promise": {
      kind: "type",
      detail: "class Promise<T>",
      doc: "Objeto que representa a conclusão ou falha de uma operação assíncrona.",
    },
    "Array": {
      kind: "type",
      detail: "class Array<T>",
      doc: "Estrutura de lista indexada de elementos.",
    },
    "Map": {
      kind: "type",
      detail: "class Map<K, V>",
      doc: "Coleção de pares chave-valor que lembra a ordem original de inserção.",
    },
    "Set": {
      kind: "type",
      detail: "class Set<T>",
      doc: "Coleção de valores exclusivos.",
    },
    "Record": {
      kind: "type",
      detail: "type Record<K, T>",
      doc: "Tipo utilitário para construir um tipo de objeto com chaves K e valores T.",
    },
    "JSON": {
      kind: "module",
      detail: "JSON.stringify / JSON.parse",
      doc: "Objeto contendo funções para análise e serialização de JSON.",
    },
    "function": {
      kind: "keyword",
      detail: "function name(args): ReturnType",
      doc: "Declaração de função.",
    },
    "const": {
      kind: "keyword",
      detail: "const name = expr;",
      doc: "Declaração de constante de escopo de bloco imutável em reatribuição.",
    },
    "let": {
      kind: "keyword",
      detail: "let name = expr;",
      doc: "Declaração de variável mutável de escopo de bloco.",
    },
    "async": {
      kind: "keyword",
      detail: "async function / async () =>",
      doc: "Declaração de função assíncrona que retorna uma Promise.",
    },
    "await": {
      kind: "keyword",
      detail: "await promise",
      doc: "Pausa a execução assíncrona até que a Promise seja resolvida.",
    },
    "interface": {
      kind: "keyword",
      detail: "interface Name { ... }",
      doc: "Declaração de contrato de tipo ou forma de objeto.",
    },
    "type": {
      kind: "keyword",
      detail: "type Name = ...",
      doc: "Alias de tipo TypeScript para uniões, interseções ou primitivos.",
    },
    "class": {
      kind: "keyword",
      detail: "class Name { ... }",
      doc: "Declaração de classe para instanciação orientada a objetos.",
    },
    "import": {
      kind: "keyword",
      detail: "import { ... } from '...';",
      doc: "Importa módulos ou declarações de arquivos externos.",
    },
    "export": {
      kind: "keyword",
      detail: "export [default] item;",
      doc: "Exporta itens para consumo por outros módulos.",
    },
    "return": {
      kind: "keyword",
      detail: "return [expr];",
      doc: "Retorna da função atual.",
    },
    "if": {
      kind: "keyword",
      detail: "if (condition) { ... }",
      doc: "Estrutura condicional que executa o bloco se a expressão for verdadeira.",
    },
    "else": {
      kind: "keyword",
      detail: "else / else if",
      doc: "Cláusula condicional executada quando a condição de if é falsa.",
    },
    "for": {
      kind: "keyword",
      detail: "for (let i = 0; i < n; i++) / for (const item of arr)",
      doc: "Estrutura de repetição para iterar sobre valores ou sequências.",
    },
    "while": {
      kind: "keyword",
      detail: "while (condition) { ... }",
      doc: "Estrutura de repetição que executa enquanto a condição for verdadeira.",
    },
    "switch": {
      kind: "keyword",
      detail: "switch (expr) { case val: ... }",
      doc: "Estrutura condicional de múltipla escolha.",
    },
    "case": {
      kind: "keyword",
      detail: "case value:",
      doc: "Cláusula de correspondência dentro de um switch.",
    },
    "try": {
      kind: "keyword",
      detail: "try { ... } catch (err) { ... }",
      doc: "Bloco para captura e tratamento de exceções.",
      snippet: "try {\n  $0\n} catch (error) {\n  \n}",
    },
    "catch": {
      kind: "keyword",
      detail: "catch (error) { ... }",
      doc: "Bloco executado quando ocorre um erro dentro do try correspondente.",
    },
    "throw": {
      kind: "keyword",
      detail: "throw new Error(msg);",
      doc: "Lança uma exceção explicitamente.",
    },
    "new": {
      kind: "keyword",
      detail: "new Constructor(args)",
      doc: "Instancia um novo objeto invocando o construtor da classe.",
    },
    "this": {
      kind: "keyword",
      detail: "this",
      doc: "Referência à instância ou contexto de execução corrente.",
    },
    "typeof": {
      kind: "keyword",
      detail: "typeof operand",
      doc: "Retorna uma string indicando o tipo do operando.",
    },
    "instanceof": {
      kind: "keyword",
      detail: "obj instanceof Class",
      doc: "Testa se o construtor aparece na cadeia de protótipos do objeto.",
    },
    "null": {
      kind: "constant",
      detail: "null",
      doc: "Valor primitivo que representa a ausência intencional de objeto.",
    },
    "undefined": {
      kind: "constant",
      detail: "undefined",
      doc: "Valor primitivo de variável declarada mas ainda não inicializada.",
    },
    "true": {
      kind: "constant",
      detail: "true",
      doc: "Literal booleano verdadeiro.",
    },
    "false": {
      kind: "constant",
      detail: "false",
      doc: "Literal booleano falso.",
    },
    "string": {
      kind: "type",
      detail: "primitive string",
      doc: "Tipo primitivo de texto UTF-16 em TypeScript.",
    },
    "number": {
      kind: "type",
      detail: "primitive number",
      doc: "Tipo primitivo numérico (IEEE 754 float de 64 bits).",
    },
    "boolean": {
      kind: "type",
      detail: "primitive boolean",
      doc: "Tipo primitivo booleano que aceita true ou false.",
    },
    "void": {
      kind: "type",
      detail: "type void",
      doc: "Tipo que representa a ausência de valor de retorno em funções.",
    },
    "any": {
      kind: "type",
      detail: "type any",
      doc: "Tipo escape que desativa as checagens de tipo do compilador.",
    },
    "unknown": {
      kind: "type",
      detail: "type unknown",
      doc: "Tipo seguro para valores arbitrários, requer narrowing antes do uso.",
    },
    "clg": {
      kind: "snippet",
      detail: "console.log(...)",
      doc: "Snippet: Imprime valor no console.",
      snippet: "console.log($0)",
    },
    "fn": {
      kind: "snippet",
      detail: "function name() { ... }",
      doc: "Snippet: Declaração de função com bloco.",
      snippet: "function $0() {\n  \n}",
    },
    "af": {
      kind: "snippet",
      detail: "(...) => { ... }",
      doc: "Snippet: Arrow function com corpo de bloco.",
      snippet: "($0) => {\n  \n}",
    },
  },
  python: {
    "print": {
      kind: "function",
      detail: "print(*objects, sep=' ', end='\\n', file=None, flush=False)",
      doc: "Imprime objetos no fluxo de saída padrão.",
    },
    "len": {
      kind: "function",
      detail: "len(s) -> int",
      doc: "Retorna o número de itens de uma sequência ou coleção.",
    },
    "range": {
      kind: "type",
      detail: "range(stop) / range(start, stop[, step])",
      doc: "Gera uma sequência imutável de números inteiros.",
    },
    "enumerate": {
      kind: "function",
      detail: "enumerate(iterable, start=0)",
      doc: "Retorna um iterador que produz tuplas contendo índice e elemento.",
    },
    "zip": {
      kind: "function",
      detail: "zip(*iterables)",
      doc: "Agrupa elementos de múltiplos iteráveis em tuplas.",
    },
    "isinstance": {
      kind: "function",
      detail: "isinstance(object, classinfo) -> bool",
      doc: "Verifica se um objeto é uma instância da classe fornecida.",
    },
    "dict": {
      kind: "type",
      detail: "class dict(**kwargs)",
      doc: "Tabela associativa de chave e valor (dicionário).",
    },
    "list": {
      kind: "type",
      detail: "class list([iterable])",
      doc: "Lista dinâmica sequencial e mutável.",
    },
    "str": {
      kind: "type",
      detail: "class str(object='')",
      doc: "Sequência imutável de caracteres Unicode.",
    },
    "int": {
      kind: "type",
      detail: "class int(x=0)",
      doc: "Número inteiro de precisão arbitrária.",
    },
    "def": {
      kind: "keyword",
      detail: "def name(args):",
      doc: "Declaração de função ou método.",
      snippet: "def $0():\n    pass",
    },
    "class": {
      kind: "keyword",
      detail: "class Name(Base):",
      doc: "Declaração de classe.",
      snippet: "class $0:\n    pass",
    },
    "import": {
      kind: "keyword",
      detail: "import module / from module import item",
      doc: "Importa módulos ou pacotes.",
    },
    "return": {
      kind: "keyword",
      detail: "return [value]",
      doc: "Retorna um valor de uma função.",
    },
    "yield": {
      kind: "keyword",
      detail: "yield [value]",
      doc: "Pausa a execução da função geradora e produz um valor.",
    },
    "async": {
      kind: "keyword",
      detail: "async def name():",
      doc: "Declara uma corrotina assíncrona.",
    },
    "await": {
      kind: "keyword",
      detail: "await coroutine",
      doc: "Pausa a execução da corrotina até que ela seja resolvida.",
    },
    "ifm": {
      kind: "snippet",
      detail: "if __name__ == '__main__':",
      doc: "Snippet: Ponto de entrada do script Python.",
      snippet: "if __name__ == \"__main__\":\n    $0",
    },
  },
  go: {
    "fmt.Println": {
      kind: "function",
      detail: "fmt.Println(a ...any) (n int, err error)",
      doc: "Imprime operandos formatados com espaços e adiciona uma quebra de linha.",
    },
    "make": {
      kind: "function",
      detail: "make(t Type, size ...IntegerType) Type",
      doc: "Aloca e inicializa slices, maps ou channels.",
    },
    "new": {
      kind: "function",
      detail: "new(Type) *Type",
      doc: "Aloca memória zerada para um tipo e retorna seu ponteiro.",
    },
    "append": {
      kind: "function",
      detail: "append(slice []Type, elems ...Type) []Type",
      doc: "Adiciona elementos a um slice e retorna o slice resultante.",
    },
    "len": {
      kind: "function",
      detail: "len(v Type) int",
      doc: "Retorna o comprimento de um array, slice, string, map ou channel.",
    },
    "func": {
      kind: "keyword",
      detail: "func name(args) (returns)",
      doc: "Declaração de função ou método.",
    },
    "package": {
      kind: "keyword",
      detail: "package name",
      doc: "Declara o pacote ao qual o arquivo pertence.",
    },
    "import": {
      kind: "keyword",
      detail: "import \"path\"",
      doc: "Importa pacotes externos ou da biblioteca padrão.",
    },
    "type": {
      kind: "keyword",
      detail: "type Name struct/interface",
      doc: "Declaração de novo tipo.",
    },
    "struct": {
      kind: "keyword",
      detail: "struct { fields... }",
      doc: "Tipo composto de campos tipados.",
    },
    "interface": {
      kind: "keyword",
      detail: "interface { methods... }",
      doc: "Conjunto de assinaturas de métodos que define uma interface.",
    },
    "return": {
      kind: "keyword",
      detail: "return [values]",
      doc: "Retorna de uma função.",
    },
    "defer": {
      kind: "keyword",
      detail: "defer func_call()",
      doc: "Adia a execução da chamada de função até o retorno da função circundante.",
    },
    "go": {
      kind: "keyword",
      detail: "go func_call()",
      doc: "Inicia a execução da chamada de função em uma nova goroutine.",
    },
    "fn": {
      kind: "snippet",
      detail: "func name() { ... }",
      doc: "Snippet: Declaração de função Go.",
      snippet: "func $0() {\n\t\n}",
    },
    "ife": {
      kind: "snippet",
      detail: "if err != nil { return ... }",
      doc: "Snippet: Verificação idiomática de erro em Go.",
      snippet: "if err != nil {\n\treturn $0\n}",
    },
    "pln": {
      kind: "snippet",
      detail: "fmt.Println(...)",
      doc: "Snippet: Print na saída padrão.",
      snippet: "fmt.Println($0)",
    },
  },
  c: {
    "printf": {
      kind: "function",
      detail: "int printf(const char *format, ...)",
      doc: "Imprime dados formatados na saída padrão stdout.",
    },
    "malloc": {
      kind: "function",
      detail: "void *malloc(size_t size)",
      doc: "Aloca um bloco contíguo de memória no heap.",
    },
    "free": {
      kind: "function",
      detail: "void free(void *ptr)",
      doc: "Libera a memória alocada previamente por malloc/calloc/realloc.",
    },
    "sizeof": {
      kind: "keyword",
      detail: "sizeof(type | expr)",
      doc: "Retorna o tamanho em bytes de um tipo ou expressão em tempo de compilação.",
    },
    "struct": {
      kind: "keyword",
      detail: "struct Name { ... };",
      doc: "Declaração de estrutura agregada de dados.",
    },
    "typedef": {
      kind: "keyword",
      detail: "typedef Existing NewName;",
      doc: "Cria um alias de tipo.",
    },
    "return": {
      kind: "keyword",
      detail: "return [expr];",
      doc: "Retorna da função atual.",
    },
  },
  cpp: {
    "std::cout": {
      kind: "constant",
      detail: "std::ostream std::cout",
      doc: "Fluxo de saída padrão do C++.",
    },
    "std::vector": {
      kind: "type",
      detail: "template<class T> class std::vector",
      doc: "Array dinâmico contíguo da STL do C++.",
    },
    "std::string": {
      kind: "type",
      detail: "class std::string",
      doc: "Classe para manipulação de strings da biblioteca padrão do C++.",
    },
    "class": {
      kind: "keyword",
      detail: "class Name { ... };",
      doc: "Declaração de classe em C++.",
    },
    "template": {
      kind: "keyword",
      detail: "template<typename T>",
      doc: "Declaração de template genérico.",
    },
  },
  bash: {
    "echo": {
      kind: "function",
      detail: "echo [-n] [-e] [arg ...]",
      doc: "Imprime argumentos separados por espaço na saída padrão.",
    },
    "printf": {
      kind: "function",
      detail: "printf format [arguments ...]",
      doc: "Formata e imprime argumentos conforme a string de formato.",
    },
    "read": {
      kind: "function",
      detail: "read [-r] [-p prompt] var",
      doc: "Lê uma linha da entrada padrão e atribui a variáveis.",
    },
    "exit": {
      kind: "function",
      detail: "exit [n]",
      doc: "Encerra o processo de shell com o código de saída especificado.",
    },
    "if": {
      kind: "keyword",
      detail: "if condition; then ...; fi",
      doc: "Estrutura condicional no shell Bash.",
    },
    "for": {
      kind: "keyword",
      detail: "for var in list; do ...; done",
      doc: "Laço de repetição sobre argumentos.",
    },
  },
  ori: {
    "fn": {
      kind: "keyword",
      detail: "fn name(args) -> ReturnType",
      doc: "Declaração de função na linguagem Ori.",
    },
    "let": {
      kind: "keyword",
      detail: "let [mut] name = expr;",
      doc: "Declaração de variável na linguagem Ori.",
    },
    "mut": {
      kind: "keyword",
      detail: "mut identifier",
      doc: "Modificador de mutabilidade para variáveis em Ori.",
    },
    "type": {
      kind: "keyword",
      detail: "type Name = ...",
      doc: "Definição de tipo na linguagem Ori.",
    },
    "match": {
      kind: "keyword",
      detail: "match expr { pattern => expr, ... }",
      doc: "Casamento de padrões exaustivo em Ori.",
    },
    "print": {
      kind: "function",
      detail: "print(value)",
      doc: "Imprime valor na saída padrão.",
    },
  },
}

// Aliases para linguagens equivalentes
languageCatalogs["javascript"] = languageCatalogs["typescript"]!
languageCatalogs["js"] = languageCatalogs["typescript"]!
languageCatalogs["ts"] = languageCatalogs["typescript"]!
languageCatalogs["jsx"] = languageCatalogs["typescript"]!
languageCatalogs["tsx"] = languageCatalogs["typescript"]!

export function getLanguageCatalog(language: string): Record<string, LanguageCatalogItem> | undefined {
  const norm = language.toLowerCase()
  return languageCatalogs[norm]
}

export function lookupCatalogItem(language: string, word: string): LanguageCatalogItem | undefined {
  const catalog = getLanguageCatalog(language)
  if (!catalog) return undefined
  return catalog[word]
}
