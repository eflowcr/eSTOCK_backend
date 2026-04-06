# Development Guide — eSTOCK Backend

Welcome! This doc explains how to work on the backend day-to-day, how the CI/CD pipeline works, and how to ship a release to production. Keep it bookmarked.

---

## Branch Strategy

We use two long-lived branches:

| Branch | Purpose | Deploys automatically? |
|--------|---------|----------------------|
| `dev`  | Active development — where all new work goes | Yes, on every push |
| `main` | Production — only updated when releasing | Yes, when a version tag is pushed |

> **Rule of thumb:** Code in `dev` is what's running on customer stacks right now (dev image). Code in `main` is what gets tagged as a release and goes to production.

---

## Daily Development

### Starting a new feature or fix

Always branch off from `dev`:

```bash
git checkout dev
git pull origin dev
git checkout -b feat/my-feature   # or fix/my-bug
```

Work on your branch, then open a PR into `dev`. Once merged, the dev workflow kicks in automatically.

### Pushing directly to `dev` (small fixes)

For quick fixes it's fine to push straight to `dev`:

```bash
git checkout dev
# ... make your changes ...
git add .
git commit -m "fix: correct stock adjustment calculation"
git push origin dev
```

This triggers the **dev CI pipeline**:
1. Runs all unit tests (`go test ./... -short`)
2. Builds and pushes `ghcr.io/eflowcr/estock-backend:dev` + `:dev-<sha>`
3. Notifies VPS Manager → all customer stacks rolling-update to the new image

---

## CI Workflows

### `deploy-dev.yml` — runs on every push to `dev`

```
push to dev
    → unit tests
    → docker build
    → push :dev + :dev-<sha> to GHCR
    → POST /stacks/update → k8s rolling update
```

### `deploy-prod.yml` — runs when a version tag is pushed

```
git push origin v1.3.0
    → unit tests
    → docker build
    → push :latest + :v1.3.0 to GHCR
    → POST /stacks/update → k8s rolling update (production)
```

### `backend-sqlc.yml` — runs on every branch / PR

Validates sqlc generated code, builds, and runs tests. This is your safety net — it will catch broken SQL queries and compilation errors before merging.

---

## Shipping a Release

When `dev` is stable and you're ready to go to production:

```bash
# 1. Make sure dev is up to date
git checkout dev
git pull origin dev

# 2. Merge dev into main
git checkout main
git pull origin main
git merge dev --no-ff -m "release: v1.3.0"
git push origin main

# 3. Tag it — this triggers the prod workflow
git tag -a v1.3.0 -m "v1.3.0: brief description of what changed"
git push origin v1.3.0

# 4. Create a GitHub Release (optional but recommended)
gh release create v1.3.0 --title "v1.3.0" --generate-notes --latest
```

That's it. The tag push triggers `deploy-prod.yml` which builds `:latest` + `:v1.3.0` and updates all customer stacks.

### Version naming

We follow [semver](https://semver.org/):
- `v1.3.0` — new features, no breaking changes
- `v1.3.1` — bug fixes only
- `v2.0.0` — breaking changes (DB schema migrations that aren't backwards compatible, etc.)

---

## Rolling Back

### Option A — Manual dispatch on GitHub

Go to **Actions → Build & Deploy Backend (Prod) → Run workflow** and enter the previous tag (e.g. `v1.2.0`). This rebuilds and redeploys that version.

### Option B — CLI

```bash
gh workflow run deploy-prod.yml -f tag=v1.2.0
```

### Option C — Direct kubectl on VPS (emergency)

```bash
kubectl set image deployment/<slug>-backend \
  backend=ghcr.io/eflowcr/estock-backend:v1.2.0 \
  -n estock-<slug>
```

---

## Image Tags Reference

| Tag | What it is | Created when |
|-----|-----------|-------------|
| `:dev` | Latest dev build (mutable) | Every push to `dev` |
| `:dev-abc1234` | Specific dev build (immutable) | Every push to `dev` |
| `:latest` | Current production (mutable) | Every version tag |
| `:v1.3.0` | Specific release (immutable) | Tag `v1.3.0` pushed |

---

## Running Tests Locally

```bash
# All tests
go test ./... -count=1

# Short tests only (skips integration tests)
go test ./... -short -count=1

# Specific test suite
go test ./services/... -run TestArticles -v
```

---

## Quick Reference

```bash
# Start feature work
git checkout dev && git pull && git checkout -b feat/xyz

# Deploy to dev (after merging your branch)
git checkout dev && git push origin dev

# Release to production
git checkout main && git merge dev --no-ff -m "release: vX.Y.Z"
git push origin main
git tag -a vX.Y.Z -m "vX.Y.Z: what changed"
git push origin vX.Y.Z

# Rollback
gh workflow run deploy-prod.yml -f tag=vX.Y.Z
```
