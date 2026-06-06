# tgprep

[![CI](https://github.com/hedgeg0d/tg-finetune-tools/actions/workflows/ci.yml/badge.svg)](https://github.com/hedgeg0d/tg-finetune-tools/actions/workflows/ci.yml)

Turn a Telegram chat export into a clean, chat-formatted dataset ready for LLM fine-tuning.

Point it at a personal chat export and it produces JSONL conversations where one
participant is mapped to `assistant` and the other to `user` — e.g. to fine-tune a
model that talks like a specific person.

## Pipeline

Processing runs in two phases that can be run separately or chained with `all`:

1. **clean** — stream the raw `result.json`, drop service/media/bot/forwarded noise,
   flatten rich text and entities to plain text, strip bare URLs, convert stickers to
   their emoji. Output: one normalized message per line.
2. **build** — sort by time, split into sessions by inactivity gap, merge message
   bursts from the same author into one turn, window long sessions into
   training-sized conversations, and emit the chosen JSONL format.

The clean phase is concurrent (worker pool) to handle large exports and reports a
breakdown of how many messages were dropped and why. PII redaction relies on the
phone/email entities Telegram tags in the export.

## Install

```sh
go build -o tgprep ./cmd/tgprep
```

## Usage

```sh
# discover participant ids and media stats to fill the config
./tgprep inspect --in result.json

# one shot: raw export -> dataset
./tgprep all --in result.json --out dataset.jsonl --config config.json

# or run the phases separately
./tgprep clean --in result.json  --out clean.jsonl   --config config.json
./tgprep build --in clean.jsonl  --out dataset.jsonl --config config.json
```

Copy `config.example.json` to `config.json` and set the role ids from `inspect`.

Every config key can also be set or overridden on the command line; flags win over
the config file:

```sh
./tgprep all --in result.json --out dataset.jsonl \
  --assistant-id user100000001 --user-id user100000002 \
  --max-turns 8 --max-chars 2000 --gap 120 --format sharegpt --workers 8
```

Run `./tgprep <command> -h` for the full flag list.

## Configuration

| key | meaning |
| --- | --- |
| `roles.assistant_id` / `roles.user_id` | Telegram `from_id` mapped to each role |
| `clean.strip_links` | drop bare URLs (anchor text is kept) |
| `clean.stickers_to_emoji` | replace stickers with their emoji |
| `clean.drop_forwarded` / `drop_via_bot` | filter forwarded and bot messages |
| `clean.redact_pii` | drop phone-number and email entities |
| `clean.min_runes` | discard messages shorter than this |
| `build.session_gap_minutes` | inactivity gap that starts a new conversation |
| `build.max_turns` / `max_chars` | window size for long sessions (`0` = unlimited) |
| `build.min_turns` | discard conversations shorter than this |
| `build.dedup` | drop conversations that are exact duplicates |
| `build.min_assistant_runes` | drop conversations where the assistant says too little |
| `build.format` | `openai` (messages) or `sharegpt` (conversations) |
| `build.system` | optional system prompt prepended to each conversation |
| `build.val_ratio` | fraction held out for validation (`0` = single file) |
| `build.seed` | shuffle seed for a reproducible train/val split |

## Output

`openai` format, one conversation per line:

```json
{"messages":[{"role":"system","content":"..."},{"role":"user","content":"..."},{"role":"assistant","content":"..."}]}
```

When `val_ratio > 0` the output is shuffled (by `seed`) and split into
`<out>.train.jsonl` and `<out>.val.jsonl` instead of a single file.

## License

MIT — see [LICENSE](LICENSE).
