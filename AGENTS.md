# AGENTS.md

These rules apply to all agent-made changes in this repository.

## Project Summary

DS2API converts DeepSeek Web chat capability into OpenAI-, Claude-, and Gemini-compatible APIs.

- Backend: pure Go implementation, centered on `cmd/ds2api`, `api/`, and `internal/`.
- Frontend: React + Vite admin console in `webui/`, built into `static/admin` for runtime hosting.
- Deployments: local source run, Docker, Vercel Serverless, and Linux/systemd.
- Main docs:
  - `README.MD`: product overview and quick start.
  - `docs/ARCHITECTURE.md`: directory structure and module boundaries.
  - `API.md`: external API contract.
  - `docs/prompt-compatibility.md`: source of truth for the API-to-DeepSeek prompt compatibility pipeline.

## Main Features

- OpenAI-compatible surface:
  - `GET /v1/models`
  - `GET /v1/models/{model_id}`
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
  - `GET /v1/responses/{response_id}`
  - `POST /v1/files`
  - `POST /v1/embeddings`
- Claude-compatible surface:
  - `GET /anthropic/v1/models`
  - `POST /anthropic/v1/messages`
  - `POST /anthropic/v1/messages/count_tokens`
  - shortcut paths under `/v1/messages` and `/messages`
- Gemini-compatible surface:
  - `POST /v1beta/models/{model}:generateContent`
  - `POST /v1beta/models/{model}:streamGenerateContent`
  - same handlers also mounted under `/v1/models/{model}:*`
- Admin API and WebUI:
  - config import/export
  - runtime settings hot update
  - account and proxy management
  - queue status and account testing
  - Vercel sync
  - chat history inspection and cleanup
  - local dev/raw-sample capture
- Runtime features:
  - shared auth resolver and account pool
  - long-history split into uploaded transcript files
  - canonical XML tool calling plus stream-time anti-leak handling
  - DeepSeek PoW implemented in Go
  - `/healthz` and `/readyz` probes

## Key Entrypoints

- `cmd/ds2api/main.go`
  - local binary entry
  - loads `.env`, refreshes logger, ensures WebUI build, starts HTTP server
- `app/handler.go`
  - app-level handler factory used by serverless runtime
- `api/index.go`
  - Vercel Go entrypoint
- `internal/server/router.go`
  - root router, middleware, health probes, protocol route mounting, admin mount, WebUI mount
- `api/chat-stream.js`
  - Vercel Node streaming entry for `/v1/chat/completions`
- `internal/js/chat-stream/*`
  - Node-side prepare/stream/release bridge and tool-sieve logic
- `start.mjs`
  - local dev helper for backend, frontend, build, and stop/status commands
- `cmd/ds2api-tests/main.go`
  - CLI entry for end-to-end testsuite execution

## Code Structure

### Top Level

- `api/`: Vercel serverless entrypoints, including the Node streaming bridge.
- `app/`: app handler assembly for serverless use.
- `cmd/`: executable entrypoints for the main server and testsuite CLI.
- `docs/`: architecture, deploy, testing, contributing, compatibility docs.
- `internal/`: core implementation.
- `pow/`: PoW implementation and related benchmarks/helpers.
- `scripts/`: lint/build/release helper scripts.
- `tests/`: fixtures, node tests, raw SSE samples, and test scripts.
- `webui/`: React admin source code.

### Core Internal Modules

- `internal/server`: root router and middleware wiring.
- `internal/httpapi/openai`: OpenAI HTTP surface, split into `chat`, `responses`, `files`, `embeddings`, `history`, `shared`.
- `internal/httpapi/claude`: Claude request normalization and response adaptation.
- `internal/httpapi/gemini`: Gemini request normalization and response adaptation.
- `internal/httpapi/admin`: admin root handler plus subpackages for `auth`, `accounts`, `configmgmt`, `settings`, `proxies`, `rawsamples`, `vercel`, `history`, `devcapture`, `version`.
- `internal/promptcompat`: the compatibility kernel that converts structured API inputs into DeepSeek-style prompt plus file references.
- `internal/prompt`: prompt assembly and role-tag formatting.
- `internal/toolcall` and `internal/toolstream`: canonical XML tool-call parsing, repair, filtering, and streaming deltas.
- `internal/deepseek/client`, `internal/deepseek/protocol`, `internal/deepseek/transport`: upstream login, session, completion, file, protocol, and transport behavior.
- `internal/account`: account pool, queue, and concurrency limits.
- `internal/auth`: API key, bearer, admin, and request auth resolution.
- `internal/chathistory`: persisted server-side chat history store and retention logic.
- `internal/config`: config loading, validation, store accessors, runtime settings.
- `internal/stream` and `internal/sse`: shared streaming parse/consume logic.
- `internal/webui`: runtime hosting for `static/admin`.
- `internal/testsuite`: reusable end-to-end testsuite engine.

### Frontend Structure

- `webui/src/app`: app bootstrapping, auth, config fetch, route composition.
- `webui/src/components`: shared UI pieces.
- `webui/src/features`: feature pages such as account management, API tester, settings, and Vercel sync.
- `webui/src/layout`: dashboard shell and layout wiring.
- `webui/src/locales`: bilingual text resources.
- `webui/vite.config.js`: dev proxy and build output to `../static/admin`.

## Main Flows

### 1. Standard API Request Flow

`cmd/ds2api/main.go` or `api/index.go`
-> `internal/server/router.go`
-> protocol handler in `internal/httpapi/openai`, `internal/httpapi/claude`, or `internal/httpapi/gemini`
-> request normalization in `internal/promptcompat`
-> prompt assembly in `internal/prompt`
-> optional inline-file preprocessing and history split
-> auth/account selection through `internal/auth` and `internal/account`
-> upstream call via `internal/deepseek/client`
-> stream or non-stream rendering back into protocol-specific output

The adapter-layer contract should stay simple: request normalization -> DeepSeek invocation -> protocol-shaped rendering.

### 2. Prompt Compatibility Flow

This is the most important logic in the repository.

- Structured client inputs are not forwarded upstream as-is.
- They are converted into:
  - one prompt string
  - one `ref_file_ids` array
  - a few control flags such as thinking/search
- Tools are injected into prompt text, not passed as native upstream tool schema.
- Historical tool calls are preserved as canonical XML in prompt-visible history.
- Long histories can be moved into uploaded transcript files such as `HISTORY.txt`.

When changing this area, always inspect:

- `internal/promptcompat/*`
- `internal/prompt/*`
- `internal/httpapi/openai/history/*`
- `internal/httpapi/openai/files/*`
- `internal/toolcall/*`
- `internal/toolstream/*`
- `docs/prompt-compatibility.md`

### 3. Vercel Streaming Flow

Vercel uses a hybrid path only for OpenAI chat streaming.

- Route rewrite: `vercel.json`
- Node entry: `api/chat-stream.js`
- Node implementation: `internal/js/chat-stream/*`
- Go-side prepare/release hooks: OpenAI chat handler in `internal/httpapi/openai/chat/*`

Flow:

1. `/v1/chat/completions` on Vercel rewrites to `api/chat-stream.js`.
2. Node asks Go for `__stream_prepare=1` to resolve auth, session, PoW, and account lease.
3. Node streams directly from DeepSeek upstream and converts SSE into OpenAI chunks.
4. Node applies tool anti-leak and finish-state rules aligned with Go.
5. Node calls Go `__stream_release=1` to release the account lease.

Do not change this flow in only one runtime. Go and Node stream semantics must remain aligned.

### 4. Admin and WebUI Flow

- Admin route mount: `internal/httpapi/admin/handler.go`
- Admin auth:
  - public login/verify routes first
  - protected routes under `RequireAdmin`
- WebUI runtime mount: `internal/webui/handler.go`
- Frontend route shell: `webui/src/app/AppRoutes.jsx`

Keep the split clear:

- `/admin/config*`: static configuration state
- `/admin/settings*`: runtime behavior and hot updates

If you add or change admin capability, check both backend admin routes and the matching `webui/src/features/*` page.

## Tech Stack

- Go `1.26`
- `github.com/go-chi/chi/v5` for HTTP routing and middleware
- `github.com/refraction-networking/utls` for upstream transport compatibility
- `github.com/router-for-me/CLIProxyAPI/v6` for proxy integration
- `github.com/google/uuid`
- React `18`
- React Router `7`
- Vite `8`
- Tailwind CSS `3`
- Docker multi-stage build
- Vercel Go + Node hybrid runtime for streaming deployment

## Configuration and Runtime Notes

- Primary config template: `config.example.json`
- Main config source in practice: `config.json` or `DS2API_CONFIG_JSON`
- Important config domains:
  - `keys` and `api_keys`
  - `accounts`
  - `model_aliases`
  - `compat`
  - `responses`
  - `history_split`
  - `embeddings`
  - `admin`
  - `runtime`
  - `auto_delete`

If you change config shape, check:

- `internal/config/*`
- `config.example.json`
- `README.MD`
- `API.md`
- relevant admin config endpoints and WebUI forms

## Where To Look First

- API routing issue: `internal/server/router.go`
- OpenAI contract issue: `internal/httpapi/openai/*`
- Claude contract issue: `internal/httpapi/claude/*`
- Gemini contract issue: `internal/httpapi/gemini/*`
- Prompt/history/tool issue: `internal/promptcompat/*`, `internal/prompt/*`, `internal/toolcall/*`, `internal/toolstream/*`
- Upstream DeepSeek behavior: `internal/deepseek/*`
- Account queue or auth issue: `internal/account/*`, `internal/auth/*`
- Admin API issue: `internal/httpapi/admin/*`
- WebUI issue: `webui/src/*` and `internal/webui/*`
- Vercel-only stream issue: `api/chat-stream.js`, `internal/js/chat-stream/*`, `vercel.json`
- Chat history issue: `internal/chathistory/*`
- End-to-end/live test behavior: `internal/testsuite/*`, `cmd/ds2api-tests/main.go`, `tests/scripts/run-live.sh`

## PR Gate

- Before opening or updating a PR, run the same local gates as `.github/workflows/quality-gates.yml`.
- Required commands:
  - `./scripts/lint.sh`
  - `./tests/scripts/check-refactor-line-gate.sh`
  - `./tests/scripts/run-unit-all.sh`
  - `npm run build --prefix webui`

## Go Lint Rules

- Run `gofmt -w` on every changed Go file before commit or push.
- Do not ignore error returns from I/O-style cleanup calls such as `Close`, `Flush`, `Sync`, or similar methods.
- If a cleanup error cannot be returned, log it explicitly.

## Change Scope

- Keep changes additive and tightly scoped to the requested feature or bugfix.
- Do not mix unrelated refactors into feature PRs unless they are required to make the change pass gates.
- Preserve the adapter layering:
  - protocol surface
  - prompt compatibility core
  - shared runtime
  - upstream client
- Avoid duplicating compatibility logic across OpenAI, Claude, and Gemini paths when a shared module already exists.

## Documentation Sync

- When business logic or user-visible behavior changes, update the corresponding documentation in the same change.
- `docs/prompt-compatibility.md` is the source-of-truth document for the “API -> pure-text web-chat context” compatibility flow.
- If a change affects message normalization, tool prompt injection, prompt-visible tool history, file/reference handling, history split, or completion payload assembly, update `docs/prompt-compatibility.md` in the same change.
- If a change affects route layout, module boundaries, or major flow descriptions, update `docs/ARCHITECTURE.md`.
- If a change affects external request or response behavior, update `API.md` and `API.en.md`.
- If a change affects deployment behavior, Vercel/Docker entrypoints, or required environment variables, update `docs/DEPLOY.md`.
- If a change affects developer workflow or verification steps, update `docs/TESTING.md` and/or `docs/CONTRIBUTING.md`.

## Validation Guidance

- For backend-only changes, start with targeted Go tests around the touched package, then run the required gates.
- For stream or tool-call changes, verify both Go and Node sides when applicable.
- For WebUI changes, run `npm run build --prefix webui`; use `./scripts/build-webui.sh` if you need the runtime artifact locally.
- For high-risk protocol or upstream-behavior changes, consider `./tests/scripts/run-live.sh`.

## Useful Commands

- Start backend locally: `go run ./cmd/ds2api`
- Start guided local dev flow: `node start.mjs dev`
- Build backend binary: `node start.mjs build`
- Build WebUI: `node start.mjs webui` or `./scripts/build-webui.sh`
- Run all unit tests: `./tests/scripts/run-unit-all.sh`
- Run live testsuite: `./tests/scripts/run-live.sh`
