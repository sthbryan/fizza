## Problem

<!-- What is broken or missing, and how it shows up. Paste the failing command,
     the wrong output, the error. If it is not a bug, say what the change enables. -->

## Fix

<!-- What this does, and why this approach over the obvious alternative.
     One bullet per atomic commit if there is more than one. -->

## Verification

<!-- How you know it works. Before/after output, or the commands you ran.
     Say plainly what you did NOT verify. -->

- [ ] `make test`
- [ ] `make vet` and `gofmt -l .` clean
- [ ] `make mcp-test` (if MCP or the model changed)
- [ ] `cd web && bun run verify` (if the frontend changed)
- [ ] Checked by hand against a built binary (if behaviour changed)

## Notes

<!-- Follow-ups deliberately left out, known limits, anything a reviewer would
     otherwise have to ask about. Delete the section if there is nothing. -->
