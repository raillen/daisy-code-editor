#!/usr/bin/env bash
# Fake do binário `opencode` para os testes do provedor nativo: responde como
# `opencode run --format json` (uma linha JSON por evento, ver
# packages/cli/src/run/noninteractive.ts do CLI v2.0.29) sem rede, sem modelo
# e sem custo. Chamado como `opencode` com o PATH dos testes apontando aqui.
#
# - FAKE_OPENCODE_ARGS: arquivo onde cada invocação anexa seus argv (uma linha).
# - FAKE_OPENCODE_MODE: ok (padrão) | split | noisy | error | fail | slow | quiet
# - FAKE_OPENCODE_SESSION: sessionID emitido (padrão ses_test123).
# - O prompt é o último argv; nos testes, prompts são texto simples (sem
#   aspas), porque o fake interpola sem escapar JSON — igual ao echo provider.
set -euo pipefail

{
  printf 'run-argv:'
  printf ' <%s>' "$@"
  printf '\n'
  printf 'run-cwd: %s\n' "$PWD"
} >> "${FAKE_OPENCODE_ARGS:-/dev/null}"

mode_file="${FAKE_OPENCODE_MODE_FILE:-}"
if [[ -n "${mode_file}" && -f "${mode_file}" ]]; then
  mode="$(cat "${mode_file}")"
else
  mode="${FAKE_OPENCODE_MODE:-ok}"
fi
session="${FAKE_OPENCODE_SESSION:-ses_test123}"
prompt="${@: -1}"

emit() { printf '%s\n' "$1"; }

case "${mode}" in
  error)
    emit "{\"type\":\"error\",\"timestamp\":1,\"sessionID\":\"${session}\",\"error\":{\"message\":\"boom from fake\"}}"
    exit 1
    ;;
  fail)
    echo "fake-opencode: exploded" >&2
    exit 3
    ;;
  slow)
    exec sleep 10
    ;;
  quiet)
    exit 0
    ;;
esac

if [[ "${mode}" == "noisy" ]]; then
  emit "! permission requested: edit (src/a.ts); auto-rejecting"
  emit "not json at all"
fi

emit "{\"type\":\"step_start\",\"timestamp\":1,\"sessionID\":\"${session}\",\"part\":{}}"
if [[ "${mode}" == "split" ]]; then
  emit "{\"type\":\"text\",\"timestamp\":2,\"sessionID\":\"${session}\",\"part\":{\"id\":\"prt_1\",\"text\":\"Fake: \"}}"
fi
emit "{\"type\":\"text\",\"timestamp\":3,\"sessionID\":\"${session}\",\"part\":{\"id\":\"prt_1\",\"text\":\"Fake: ${prompt}\"}}"
emit "{\"type\":\"step_finish\",\"timestamp\":4,\"sessionID\":\"${session}\",\"part\":{}}"
