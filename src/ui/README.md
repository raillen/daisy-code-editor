# Presentation Layer (`src/ui`)

A camada **UI** implementa a interface de usuário baseada em terminal (TUI) utilizando **OpenTUI** integrado ao modelo reativo fino do **SolidJS**.

## Componentes

- **`workbench.tsx`**: Componente raiz que orquestra a janela do editor:
  - Barra de menus superior (`Menubar`).
  - Barra de abas com suporte a scroll e fechamento via botão do meio (`TabBar`).
  - Barra lateral com árvore de diretórios e painel de controle de versão (`SCM`).
  - Painéis de edição com suporte a divisões verticais e horizontais (`EditorPane`).
  - Painel inferior para Terminal PTY integrado e visualização em tempo real de Markdown.
  - Barra de status reativa com badges interativos (`StatusBar`).
  - Camadas sobrepostas para modais, busca, paleta de comandos e ajuda (`OverlayView`).
  - Janela flutuante de autocompletação de código (`CompletionDropdown`).
- **`editor.tsx`**: Componente de alta performance para renderização do editor de texto, números de linha, marcadores de Git no gutter, múltiplos cursores, seleções ativas, scrollbar visual e quebra suave de linha (`visualRows`).
- **`theme.ts`**: Resolução de tokens semânticos de cores e estilos para OpenTUI com suporte a terminal sem cor (`no-color`).
- **`menus.ts`**: Ponto de reexportação das definições de menu para a UI.

## Invariantes Arquiteturais

1. **Apresentação Pura**: Os componentes de UI não contêm lógica de negócio ou persistência; reagem diretamente ao estado do `Application` e emitem ações via `app.dispatch(action)`.
2. **Alta Performance de Renderização**: Algoritmos de layout de linha utilizam caminhos rápidos para texto ASCII, evitando alocações desnecessárias por frame.
