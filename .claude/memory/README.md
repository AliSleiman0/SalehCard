# Carried Claude Code memory

These are the Claude Code **project memory** files from the machine this repo was
developed on, committed so they travel when switching machines. They are a snapshot
from when they were written — for the authoritative current state of the code, prefer
`../../CLAUDE.md`, `../../HANDOFF.md`, and `../../MANUAL-TEST.md`.

- `MEMORY.md` — the index Claude loads each session (one line per memory).
- `salehcard-dev-env.md` — how to run the stack (ports, pnpm location, admin dev-bypass).
- `salehcard-design-port.md` — storefront design-system architecture and what's real vs mock.
- `salehcard-admin-port.md` — admin console architecture and what's wired vs stubbed.

## Restore on a new machine

Claude Code reads project memory from
`~/.claude/projects/<encoded-project-path>/memory/`, where `<encoded-project-path>` is
the project's absolute path with `/` replaced by `-` (e.g. `/home/you/salehcard` →
`-home-you-salehcard`). To restore:

```bash
# from the repo root, after cloning on the new machine:
DEST=~/.claude/projects/$(pwd | sed 's#/#-#g')/memory
mkdir -p "$DEST"
cp .claude/memory/*.md "$DEST"/
```

Then start Claude Code in this repo — it will pick up `MEMORY.md` and the linked files.
Some details (ports already in use, `~/.local/bin` pnpm path) were specific to the old
machine; verify them against the new environment.
