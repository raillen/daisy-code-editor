#!/usr/bin/env bash
# Runner das demos VHS: entra no fixture e executa o binário construído.
#
# A tape chama `./demos/run.sh <fixture>` a partir da raiz do repo; os paths
# são resolvidos a partir do script, então a sessão do fixture (cwd) é o
# workspace restaurado — sem argumento de caminho, que é o que ativa o
# restore de `show_agents` e das larguras.
set -euo pipefail
ROOT="$(cd "$(dirname "${0}")/.." && pwd)"
FIXTURE="${1:?usage: run.sh <fixture>}"
if [[ ! -x "${ROOT}/dist/cli.js" ]]; then
  echo "demos/run.sh: dist/cli.js ausente — rode 'bun run build' antes" >&2
  exit 1
fi
cd "${ROOT}/demos/${FIXTURE}"
exec bun "${ROOT}/dist/cli.js"
