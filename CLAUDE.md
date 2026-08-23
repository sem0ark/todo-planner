Application vision and requirements are defined @docs/ along with API specification, DB structure, etc. In case of any explicit changes, adjust the documentation.

**Avoid documenting all your actions after completing a task in separate report files**. All necessary and appropriate changes should be documented in appropriate files @docs/ only.

In case of working on the backend:
- Always imlpement features along with the structures defined in @api-backend/ARCHITECTURE.md
- Use `go` CLI

In case of working on any of the frontend projects:
- Always align the style with the style specification @docs/
- Use `pnpm` CLI for web/React projects
- Use `make` CLI for the Swift macOS desktop widget (@front-desktop-widget/)

In case of working on the desktop widget (@front-desktop-widget/):
- Use Swift with SwiftUI for macOS 13+
- Follow the spatial layout and state machine defined in the README
- Use `make` commands for build/run operations
- Keep the codebase minimal and beginner-friendly
