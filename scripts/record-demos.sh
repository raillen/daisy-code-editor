#!/usr/bin/env bash
# Grava as demos do Daisy com VHS (fora do CI bloqueante: é processo manual).
#
# - Faz o build, isola HOME/git-config (determinismo) e limpa os fixtures
#   antes e depois (o app persiste session.toml ao sair; o fixture-scm ganha
#   um .git/ na hora).
# - Saída em demos/out/ (gitignored). Rode com um terminal decente por perto:
#   o VHS precisa de `vhs`, `ffmpeg` e `git` no PATH.
set -euo pipefail
ROOT="$(cd "$(dirname "${0}")/.." && pwd)"
cd "${ROOT}"

for bin in vhs ffmpeg bun git; do
  command -v "${bin}" >/dev/null || { echo "record-demos.sh: '${bin}' não está no PATH" >&2; exit 1; }
done

vhs validate demos/agents.tape demos/scm.tape
bun run build
bash -n demos/run.sh

clean_fixtures() {
  rm -rf demos/fixture-scm/.git
  rm -f demos/fixture-agents/.daisy/session.toml demos/fixture-scm/.daisy/session.toml
  git checkout -- demos/ 2>/dev/null || true
}
clean_fixtures
mkdir -p demos/out

export HOME="$(mktemp -d "${TMPDIR:-/tmp}/daisy-demos-XXXXXX")"
export XDG_CONFIG_HOME="${HOME}/.config" XDG_CACHE_HOME="${HOME}/.cache"
export DAISY_REPO="${ROOT}"
trap 'rm -rf "${HOME}"' EXIT

for tape in demos/agents.tape demos/scm.tape; do
  vhs "${tape}"
  pkill -f "dist/cli.js" 2>/dev/null || true
done

clean_fixtures
ls -la demos/out/
