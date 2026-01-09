# Databricks Apps CLI Design Doc

## 1. Goals & Non-Goals

### 1.1 Goals

**Primary goal:** Enable users to **create, develop, and deploy Databricks apps quickly and consistently**, by providing a **batteries-included, unified dev/build/deploy loop** that abstracts Databricks integration complexity.

**Target audience:**

* Databricks customers:

  * Data engineers
  * Platform engineers
  * Product engineers building end-user-facing applications

**Magical workflows:**

1. App creation — minimal setup, opinionated defaults, ready-to-run projects
2. Development — fast feedback loops, local or hybrid dev, minimal Databricks friction
3. Time to production — predictable, repeatable deploys with low cognitive overhead

**Design philosophy:**

* Batteries-included, prescriptive
* Strong happy-path guidance, optional escape hatches

### 1.2 Non-goals

* General infrastructure provisioning
* Replacing Terraform or other IaC tools
* Replacing existing Databricks Bundle workflows
* Monorepo support (one app per repository)

**Scope limitation:** apps-only; CLI does not manage jobs, notebooks, pipelines, or other Databricks assets directly.

---

## 2. Relationship to Existing Databricks CLI

* Integrated as a **new subcommand** within the existing Databricks CLI:

```bash
# Current (experimental)
databricks experimental appkit <command>

# Future (GA)
databricks appkit <command>
```

* Written in **Go**, sharing authentication, workspace config, and profiles with the main CLI.
* **Independent from Databricks Bundles**: AppKit has its own deployment mechanism. `databricks.yml` is optional.
* Users may mix workflows in the same repository if desired:

```bash
databricks bundle deploy      # Optional - if using bundles
databricks appkit deploy      # AppKit's native deployment
```

* Backward compatibility: `appkit` commands must **not break existing CLI commands**. Within `appkit`, breaking changes are permissible during early iterations.

---

## 3. App Mental Model & Structure

### 3.1 Concept

* A Databricks app is a **single deployable unit** consisting of:

  * A **browser-based frontend** (React for `appkit`)
  * A **backend service** (Node.js now, Python in future)
* May depend on Databricks-managed resources (compute, APIs, Unity Catalog) but does **not provision these resources itself**.
* Deployable to **multiple workspaces**; one codebase may target multiple environments.
* **One app per repository** — monorepo patterns are not supported.

### 3.2 Project Structure & Detection

* `appkit` apps follow a **fixed, opinionated project layout**, inspired by frameworks like Next.js or Vite.
* Some flexibility exists to support **other app types** (e.g., Streamlit), though feature support may be limited.
* Apps include an `app.yaml` file for runtime configuration.
* CLI auto-detects the app type based on:

  * Presence of `app.yaml`
  * Directory structure heuristics (`client/`, `server/`)

### 3.3 Frontend / Backend Assumptions

* Frontend: browser-based, React (for appkit), supports hot reload
* Backend: Node.js (TypeScript) now; Python support planned
* `dev` and `dev-remote` loops orchestrate local and hybrid development

---

## 4. Languages & Runtimes

### 4.1 Supported Runtimes

| Runtime    | Status      | Backend         | Frontend |
| ---------- | ----------- | --------------- | -------- |
| TypeScript | Supported   | Node.js         | React    |
| Python     | Planned     | Flask/FastAPI   | React    |

* Each app has **one backend service**; multi-backend apps are out of scope.
* Frontend is always browser-based (React) regardless of backend language.

### 4.2 Runtime Detection

CLI automatically detects the project runtime using:

1. **Explicit declaration** in `app.yaml` (takes precedence):
   ```yaml
   runtime: node    # or: python
   ```

2. **File-based detection** (fallback):
   | File                  | Detected Runtime |
   | --------------------- | ---------------- |
   | `package.json`        | TypeScript/Node  |
   | `pyproject.toml`      | Python           |
   | `requirements.txt`    | Python           |

### 4.3 Command Behavior by Runtime

Commands adapt their behavior based on detected runtime:

| Command   | TypeScript (Node)          | Python                     |
| --------- | -------------------------- | -------------------------- |
| `dev`     | `npm run dev`              | TBD (e.g., `flask run`)    |
| `build`   | `npm run build` → `dist/`  | TBD (may skip build step)  |
| `deploy`  | Upload `dist/`             | TBD (upload source?)       |
| `status`  | Same behavior              | Same behavior              |
| `logs`    | Same behavior              | Same behavior              |

**Override via `app.yaml`:**

Commands can be customized in `app.yaml` for non-standard setups:

```yaml
runtime: node
scripts:
  dev: npm run dev:custom
  build: npm run build:prod
```

### 4.4 `create` and Language Selection

* **Current**: TypeScript is the default (only supported runtime)
* **Future**: Interactive prompt will ask for language choice when Python is supported

```bash
databricks appkit create
# → Select language: TypeScript / Python
```

### 4.5 Non-AppKit Apps

Non-appkit apps (e.g., Streamlit, Gradio) receive **partial support**:
* CLI scaffolds them and auto-detects app type
* Only `deploy`, `status`, `logs` are fully functional
* `dev`, `build` may have limited or no support

---

## 5. CLI Command Design

### 5.1 Command Set

| Command          | Description                                                                 |
| ---------------- | --------------------------------------------------------------------------- |
| `create`         | Scaffold a new appkit app; supports interactive wizard or GitHub template. |
| `list-templates` | Show available built-in and cached templates.                               |
| `dev`            | Local dev loop; frontend + backend local, connected to Databricks.          |
| `dev-remote`     | Frontend local, backend runs on Databricks, connected via websocket.        |
| `build`          | Compile/build app into deployable artifacts (outputs to `dist/`).           |
| `deploy`         | Deploy app to workspace; implies `build`. Creates or updates the app.       |
| `status`         | Show deployment state, app URL, and last deploy time.                       |
| `logs`           | Stream app logs from workspace.                                             |
| `destroy`        | Remove deployed app; shows what will be deleted, requires confirmation.     |
| `metrics`        | Show usage metrics. (TBD)                                                   |

**Future:** lint, test, package validation

### 5.2 `create` UX

**Interactive default:**

```bash
databricks appkit create
```

CLI prompts in sequence:

1. **App name** — lowercase letters, numbers, hyphens (max 26 chars)
2. **Select features** — optional capabilities (e.g., Analytics with SQL charts)
3. **Feature dependencies** — prompts for required config (e.g., SQL Warehouse ID if Analytics selected)
4. **Description** — optional app description

**Template resolution priority:**

1. `--template` flag (local path or GitHub URL)
2. `DATABRICKS_APPKIT_TEMPLATE_PATH` environment variable
3. Built-in AppKit template (default)

**Template sources:**

```bash
# Built-in default (no flags needed)
databricks appkit create

# Local path
databricks appkit create --template /path/to/template

# GitHub URL (simple repo)
databricks appkit create --template https://github.com/user/my-template

# GitHub URL with subdirectory
databricks appkit create --template https://github.com/user/repo/tree/main/templates/starter

# GitHub URL with explicit branch/tag
databricks appkit create --template https://github.com/user/repo --branch v1.0.0
```

**Non-AppKit templates:**

* CLI scaffolds the project and attempts to detect app type
* Partial support: `deploy`, `status`, `logs` work; other commands may be limited

### 5.3 Command Philosophy

* Commands operate **in current directory**; no `--app`/`--path`.
* **Safe by default** (prompts before destructive actions).
* Interactive/user-focused; CI support optional.
* `deploy` implies `build`.
* **Minimal errors by default**; use `--debug` for verbose output. Errors link to docs/troubleshooting.

### 5.4 Environments & Workspace Targets

Deploy target can be specified via:

* `--profile` flag (uses `~/.databrickscfg`)
* `--workspace` flag with URL
* Configuration in `app.yaml`

```bash
databricks appkit deploy --profile prod
databricks appkit deploy --workspace https://my-workspace.cloud.databricks.com
```

### 5.5 Dev Loop

**`dev` command:**

* Frontend and backend both run **locally**
* Backend connects to Databricks APIs using CLI authentication
* Requires `.env` file for resource configuration (warehouse ID, etc.)
* Hot reload supported for both frontend and backend

**`dev-remote` command:**

* Frontend runs **locally** with hot reload
* Backend runs on **Databricks** (deployed environment)
* Connected via **websocket** for real-time communication
* Useful for testing with production-like backend behavior

**Authentication:**

* Requires prior `databricks auth login --host <workspace-url>`
* Dev loop uses same authentication as CLI
* `.env` file configures app-specific resources (not auth)

---

## 6. Configuration Files

### 6.1 `app.yaml` (Required)

Runtime configuration for the Databricks Apps platform:

```yaml
name: my-app
command:
  - node
  - dist/server/server.js
env:
  - name: DATABRICKS_WAREHOUSE_ID
    valueFrom: sql-warehouse
```

### 6.2 `databricks.yml` (Optional)

Only needed if using Databricks Bundles for deployment. AppKit has its own deployment mechanism and does not require bundles.

### 6.3 `.env` (Local development)

Local environment configuration for `dev` command:

```env
DATABRICKS_WAREHOUSE_ID=abc123def456
DATABRICKS_APP_PORT=8000
```

---

## 7. Project Layout (AppKit Scaffold)

```
my-app/
├─ app.yaml              # Required - runtime config for Databricks Apps
├─ databricks.yml        # Optional - only if using Databricks Bundles
├─ .env                  # Local dev config (gitignored)
├─ client/               # Frontend (React)
│  ├─ src/
│  ├─ vite.config.ts
│  └─ ...
├─ server/               # Backend (Node.js/TypeScript)
│  ├─ server.ts
│  └─ ...
├─ config/               # App configuration (queries, etc.)
├─ dist/                 # Build output
├─ tests/
├─ package.json
└─ README.md
```

* `app.yaml` contains runtime config for Databricks Apps platform
* `client/` and `server/` are separate for hot reload and build isolation
* Build outputs to `dist/`

---

## 8. Example CLI Workflows

**Create an app (interactive):**

```bash
databricks appkit create
```

**Create from GitHub template:**

```bash
databricks appkit create --template https://github.com/user/app-template
```

**Create with specific branch:**

```bash
databricks appkit create --template https://github.com/user/repo --branch v2.0.0
```

**List available templates:**

```bash
databricks appkit list-templates
```

**Local development:**

```bash
cd my-app
databricks appkit dev
```

**Remote development (backend on Databricks):**

```bash
databricks appkit dev-remote
```

**Build & deploy:**

```bash
databricks appkit build
databricks appkit deploy --profile staging
```

or combined:

```bash
databricks appkit deploy --profile staging
```

**Monitor app:**

```bash
databricks appkit status
databricks appkit logs
databricks appkit metrics
```

**Destroy app safely:**

```bash
databricks appkit destroy
# Shows what will be deleted, requires confirmation
```

---

## 9. Future Considerations

* **Python support** (backend)
* **Offline/mock mode** — develop without Databricks connection using mocked data
* **Additional commands**: lint, test, validate
* **CI/CD integration** — non-interactive mode for pipelines
* **Enhanced environment support** (`env` vs `profile`)
* **Community templates registry** for `create --template`
* **Metrics command** — request counts, latency, resource usage
