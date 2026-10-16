# Demo do modal SCM: monta um repo git com uma mudança (identidade local,
# sem tocar no ~/.gitconfig), abre o modal pelo menu Git — primeira ação,
# mesma que o acorde Ctrl+Shift+G despacha —, adiciona ao índice com `s` e
# fecha. O `.git/` é criado na hora e removido pelo record-demos.sh.
Output demos/out/scm.gif
Set Shell "bash"
Set Width 1200
Set Height 700
Type@15ms "cd $DAISY_REPO/demos/fixture-scm && git init -qb main . && git config user.email demo@invalid && git config user.name Demo && git add -A && git commit -qm base && printf 'version = 2\n' >> tracked.txt" Enter
Sleep 3s
Type "cd $DAISY_REPO" Enter
Sleep 500ms
Type "./demos/run.sh fixture-scm" Enter
Sleep 6s
Alt+T
Sleep 1s
Enter
Sleep 2s
Type "s"
Sleep 1s
Escape
Sleep 500ms
Ctrl+Q
Sleep 1s
