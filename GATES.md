# GATES

Ledger of gate approvals and approved versions. Every decision here is a human's: write
it by hand, or reply to the agent and confirm the exact line it proposes (see
`docs/process/gates.md`). Lines are only appended, never edited or deleted. Add one
line per decision, newest at the bottom. The last line for a gate wins.

Formats (replace the angle-bracket parts):

- `G<N>: APPROVED by <name>, <YYYY-MM-DD>. Version: <X.Y.Z or none>. Notes: <text>`
- `G<N>: REJECTED by <name>, <YYYY-MM-DD>. Notes: <what must change>`
- `R: APPROVED by <name>, <YYYY-MM-DD>. Version: <X.Y.Z>. Notes: <text>` for a release
  between gates, such as a hotfix.

`Version: none` closes a gate without a release.

## Ledger

