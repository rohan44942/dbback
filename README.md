# dbback — Phase 1 scaffold

This repo contains the Phase 1 scaffold for the Database Backup Utility.

Phase 1 features:
- CLI (`cmd/dbback`) with `backup`, `list`, `restore` commands.
- Local storage under `backups/`.
- Simple JSON metadata under `metadata/metadata.json`.
- Placeholders for future components (agent, API, worker).

Run:
  go build -o bin/dbback ./cmd/dbback
  ./bin/dbback

