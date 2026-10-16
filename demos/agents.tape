# Demo do painel de agentes: abre pelo menu Exibir (o acorde Ctrl+Shift+A não
# atravessa o terminal legado do VHS como chord — byte 0x01 chega sem o shift
# — então a demo usa o caminho do menu, que despacha a mesma ação), manda um
# prompt para o provedor echo, abre uma segunda conversa e volta ao editor.
Output demos/out/agents.gif
Set Shell "bash"
Set Width 1200
Set Height 700
Type "./demos/run.sh fixture-agents" Enter
Sleep 6s
Alt+X
Sleep 1s
Down
Sleep 500ms
Enter
Sleep 1s
Type "explain greet"
Sleep 500ms
Enter
Sleep 4s
Ctrl+N
Sleep 500ms
Type "second chat"
Enter
Sleep 3s
Escape
Sleep 500ms
Ctrl+Q
Sleep 1s
