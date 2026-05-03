# Adding or modifying jtk commands

Two normative specs live in this directory. Read whichever is relevant **before** writing code, not after a reviewer flags it.

- **[GUARDRAILS.md](GUARDRAILS.md)** — the command surface contract: verb language, flag short-alias map, pagination defaults, positional-vs-flag rule, mutation safety, boolean flag conventions, naming hygiene.

- **[OUTPUT_SPEC.md](OUTPUT_SPEC.md)** — the output contract: list/get/mutation shapes, `--id` / `--extended` / `--fulltext` semantics, date formatting, error rules. Includes a per-command example catalog.

These docs are the single source of truth. CLAUDE.md files elsewhere in this repo (root and `tools/jtk/`) act as indexes that point here — do not duplicate the rules; update these files instead.
