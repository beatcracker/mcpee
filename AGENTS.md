# AGENTS.md

## Tools

- Use project-local tooling only. Configure shared tools through `mise`
  Do not install global dependencies.
- **Run `mise run check` after every completed fix or feature.**
  It is the repo's read-only format, lint, syntax, and test gate.
  Run it once the change is done, not after every intermediate edit.
- If `mise run check` reports a fixable formatting or lint issue covered by
  repo tooling, run `mise run fix` first. Do not hand-patch fixer-managed
  output unless the fixer does not resolve it or the remaining change needs
  human judgment.

### mise rules

- Use `mise use` to add tools. Do not edit the configuration directly.
- Always enable lockfile mode.
- Prefer `latest` with a lockfile. Pin versions only when required.
- Keep tasks DRY with `mise` templates, variables, and environment passing.
- Do not use the `asdf` backend. Shared tools must use explicit non-`asdf` backends such as `core`, `aqua`, or `npm`.
- After changing shared tools, run `mise lock` and verify `mise.lock` contains no `asdf:` backend entries.

## Commit style

[Conventional Commits](https://www.conventionalcommits.org/).
Scope encouraged — use the area touched.

## Writing style

- **Bold is for emphasis, not decoration.** Use `**bold**` only when a word or phrase must punch through — a hard constraint, a surprising gotcha, a rule that was repeatedly violated. If everything is bold, nothing is. Default to plain text.
- **Sentence case for all headings.** Write "What we tried", not "What We Tried".
- Keep prose direct. Short sentences, no filler, no preamble.
- Document general approaches, not specific scripts. Unless a workflow is non-obvious, highly specific, or genuinely one-off, describe the principle — not a concrete command sequence. Specific snippets are fine when the approach can't be generalized.
