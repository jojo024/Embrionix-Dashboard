# Contributing

## Development Setup

```bash
git clone https://github.com/embrionix/dashboard.git
cd dashboard

# Backend
go mod download
go run ./cmd/server/

# Frontend (separate terminal)
cd web && npm install && npm run dev
```

## Workflow

For every change, no matter how small:

```
1. Open an issue (bug report or feature request)
2. Create a branch:  git checkout -b feature/<slug>
3. Implement + test
4. Update docs if behaviour changed
5. Push branch + open PR referencing the issue
6. Merge after review
```

Branch naming: `feature/<slug>`, `fix/<slug>`, `docs/<slug>`, `chore/<slug>`.

## Rules

### Embrionix API

- **Never assume** an API endpoint exists without verifying it against `documentations/api_e+.html`.
- If a desired feature is **not supported** by the device API, record it in `ISSUES.md` and propose an alternative.
- Test API calls against a real EM6 device or a documented mock before merging.

### Go

- `go vet ./...` and `go build ./...` must pass cleanly.
- Follow standard Go conventions (gofmt, no exported symbol without a doc comment if it's public API).
- No new external dependencies without discussion.

### TypeScript / React

- `npx tsc --noEmit` must pass.
- `npm run build` must produce a clean dist.
- Components should be small and single-purpose.
- All new API calls go through `src/api/client.ts`.
- New server-state hooks go in `src/hooks/`.

### Documentation

- Update `API.md` for any new endpoint.
- Update `CHANGELOG.md` for any user-visible change.
- Update `ROADMAP.md` when a phase item is completed.
- If a feature is deferred or blocked, record it in `ISSUES.md`.

## Commit Messages

Use imperative mood, present tense:

```
Add SFP DDM temperature chart to device detail
Fix polling panic when device has no management IP
Update ROADMAP to mark Phase 1 complete
```

## Code Review Checklist

- [ ] Does the change touch device communication? Verified against API docs?
- [ ] Is the happy path tested?
- [ ] Is the error path handled gracefully (no panics, meaningful error messages)?
- [ ] Are docs updated?
- [ ] Does `go build ./...` and `npm run build` pass?
