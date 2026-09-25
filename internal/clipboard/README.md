# internal/clipboard

O buffer de cópia da sessão, e a sequência que o entrega ao terminal.

**Camadas, na ordem em que são tentadas, e a ordem é o desenho:**
1. **Buffer interno** — sempre presente, sempre autoritativo *nesta* sessão.
2. **OSC 52** — o lado da escrita é gratuito e síncrono: um terminal que aceita
   coloca o texto no clipboard do **sistema**, então a cópia sobrevive à saída e
   funciona sobre SSH, onde não existe clipboard de X11.
3. **Nada mais.** O `atotto/clipboard` faria `fork` de `xclip`/`wl-copy` a cada
   cópia, e o plano o lista como **não-candidata**: uma TUI que cria um processo
   por tecla é uma TUI que engasga.

**A escrita devolve a sequência, não faz o efeito.** O pacote é puro e o terminal
é assunto da raiz de composição — a mesma divisão que mantém a sequência testável
sem terminal, e impede que uma escape entre no dump de conformidade, que é
comparado byte a byte.

**`Paste` lê a sessão, e diz isso.** Perguntar ao terminal o que há no clipboard
exige uma resposta assíncrona que o runtime ainda não tem, e um editor que finge
ler o clipboard do sistema e cola o buffer da última cópia é pior do que um que
declara de onde lê.

**O buffer é estado do processo**, e por isso os testes da cópia esvaziam o buffer
explicitamente: um teste que presume buffer vazio falha quando alguém acrescenta
um teste antes dele. O oráculo tem o mesmo problema e a mesma resposta
(`CLIPBOARD_TEST_LOCK`).
