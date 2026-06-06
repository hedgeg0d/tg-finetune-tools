#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# tgprep end-to-end runner: clean -> build (dataset) -> memory (RAG)
# Resumable: each stage is skipped once finished; interrupted LLM stages
# resume from their on-disk cache (system_prompts.json / memory_facts.json).
# Re-run anytime. Set FORCE=1 to redo everything. Override paths via env.
# ---------------------------------------------------------------------------

IN="${IN:-$(ls -1 ChatExport_*/result.json 2>/dev/null | head -1)}"
CONFIG="${CONFIG:-config.json}"
CLEAN="${CLEAN:-clean.jsonl}"
DATASET="${DATASET:-dataset.jsonl}"
MEMORY="${MEMORY:-memory.jsonl}"
OLLAMA="${OLLAMA:-http://localhost:11434}"
RUN_MEMORY="${RUN_MEMORY:-1}"
STATE_DIR="${STATE_DIR:-.tgprep_state}"

cd "$(dirname "$0")"
mkdir -p "$STATE_DIR"
[ "${FORCE:-0}" = "1" ] && rm -f "$STATE_DIR"/*.done

c_reset=$'\e[0m'; c_grn=$'\e[32m'; c_yel=$'\e[33m'; c_red=$'\e[31m'; c_cyn=$'\e[36m'
log()  { printf '%s\n' "$*" >&2; }
die()  { printf '%serror:%s %s\n' "$c_red" "$c_reset" "$*" >&2; exit 1; }

# Stages to run (memory optional)
STAGES=(clean build)
[ "$RUN_MEMORY" = "1" ] && STAGES+=(memory)
TOTAL=${#STAGES[@]}

done_count() { ls "$STATE_DIR"/*.done 2>/dev/null | wc -l | tr -d ' '; }

bar() {
  local done=$1 total=$2 width=24
  local fill=$(( done * width / total )) i
  printf '%s[' "$c_cyn"
  for ((i=0;i<width;i++)); do [ $i -lt $fill ] && printf '#' || printf '.'; done
  printf ']%s %d/%d stages\n' "$c_reset" "$done" "$total"
}

stage_header() {
  log ""
  log "${c_yel}==> [$1/$TOTAL] $2${c_reset}"
  bar "$(done_count)" "$TOTAL"
}

# ---------------------------------------------------------------------------
# Pre-flight
# ---------------------------------------------------------------------------
[ -f ./tgprep ] || { log "building tgprep..."; go build -o tgprep ./cmd/tgprep; }
[ -f "$IN" ] || die "input not found: $IN"
[ -f "$CONFIG" ] || die "config not found: $CONFIG"

curl -sf --max-time 5 "$OLLAMA/api/tags" >/dev/null 2>&1 || die "ollama not reachable at $OLLAMA (start: ollama serve)"

if [ "$RUN_MEMORY" = "1" ]; then
  need_embed=$(python3 -c "import json;print(json.load(open('$CONFIG')).get('memory',{}).get('do_embeddings',False))" 2>/dev/null || echo False)
  EMBED_MODEL="${EMBED_MODEL:-$(python3 -c "import json;print(json.load(open('$CONFIG')).get('memory',{}).get('embeddings',{}).get('model',''))" 2>/dev/null || true)}"
  if [ "$need_embed" = "True" ] && [ -n "$EMBED_MODEL" ]; then
    if ! curl -sf --max-time 5 "$OLLAMA/api/tags" | grep -q "\"${EMBED_MODEL%%:*}"; then
      log "${c_yel}embedding model '$EMBED_MODEL' missing — pulling...${c_reset}"
      ollama pull "$EMBED_MODEL" || die "failed to pull $EMBED_MODEL (or set memory.do_embeddings=false)"
    fi
  fi
fi

# ---------------------------------------------------------------------------
# Stages
# ---------------------------------------------------------------------------
run_clean() {
  ./tgprep clean --in "$IN" --out "$CLEAN" --config "$CONFIG"
}
run_build() {
  ./tgprep build --in "$CLEAN" --out "$DATASET" --config "$CONFIG"
}
run_memory() {
  ./tgprep memory --in "$CLEAN" --out "$MEMORY" --config "$CONFIG"
}

i=0
for s in "${STAGES[@]}"; do
  i=$((i+1))
  marker="$STATE_DIR/$s.done"
  if [ -f "$marker" ]; then
    log "${c_grn}==> [$i/$TOTAL] $s — already done (skip)${c_reset}"
    continue
  fi
  stage_header "$i" "$s"
  case "$s" in
    clean)  run_clean  ;;
    build)  run_build  ;;
    memory) run_memory ;;
  esac
  touch "$marker"
done

log ""
bar "$(done_count)" "$TOTAL"
log "${c_grn}all done.${c_reset}"
log "  dataset: ${DATASET%.jsonl}.train.jsonl / ${DATASET%.jsonl}.val.jsonl"
[ "$RUN_MEMORY" = "1" ] && log "  memory:  $MEMORY"
