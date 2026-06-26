# AGENTS.md

Follow `docs/spec-v0.4.md` as the authoritative product and implementation specification.

Important constraints:

- Keep `disk-smi` read-only.
- Do not shell out through `sh -c` or `bash -c` for disk commands.
- Preserve the distinction between missing values and zero.
- Keep JSON keys and enum values in English regardless of display language.
- Verify rendered terminal lines with display-cell width, not byte length.
- Do not expose full serial numbers unless `--show-serial` is explicitly used.
