Abide to the root AGENTS.md instructions.
For main validation, always execute:
- `make test` for full test suite execution, avoid using go test directly.
- `make format` for autoformatting, avoid manually formatting specific files, do not bother in case it formats files outside of api-backend.
- `make lint-backend` for backend linting, avoid running linters manually.
