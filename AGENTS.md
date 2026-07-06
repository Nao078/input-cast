# AGENTS.md

AI agents working in this repository must follow these project rules.

## Local Execution Environment

For this working environment, use Debian WSL.

- Do not `cd` directly into Linux/WSL paths from Windows.
- At the start of work, run `wsl.exe -l -v`.
- Always run project commands through `wsl.exe -d Debian bash -lc '...'`.
- Use this working directory inside Debian:

```text
/home/naoki/develop/input-cast
```

- If Debian is not available, stop the task and report that Debian is unavailable.

## Project Overview

- input-cast is a Go-based input visualization tool for GP2040-CE-style leverless controllers.
- The recommended flow is: Input Cast Client captures controller input, the Go server relays state, and OBS/browser renders `/overlay`.
- The tool is for streaming and practice visualization only.
- It must not access game processes, game memory, or rendering APIs.
- It must not implement automatic input, macros, input correction, game-state analysis, or memory reads.
- Combo display is based on input state plus YAML move/recipe data. It must not infer in-game hit success, guard state, distance, or juggle state.

## Directory Structure

- `cmd/server/`: Go HTTP/WebSocket server, config APIs, combo APIs, and static route wiring.
- `cmd/input-cast-client/`: native Input Cast Client application and its assets.
- `internal/config/`: JSON/YAML config and combo parsing logic.
- `internal/input/`: input manager and platform-specific input providers.
- `internal/gamepad/`: gamepad backend implementations.
- `internal/bridge/`: bridge client/config/state code.
- `internal/ws/`: WebSocket hub.
- `web/overlay/`: OBS/preview overlay HTML, CSS, JavaScript, and overlay tests.
- `web/gamepad/`: browser client UI.
- `web/assets/`: shared browser assets.
- `configs/`: controller/layout profile JSON files and `.active-profile`.
- `combos/`: combo YAML files and `.active-combo`.
- `packaging/linux/`: Linux package build script and desktop entry.
- `dist/`: generated binaries and Linux packages.

## Work Rules

- Keep changes small and scoped to the requested behavior.
- Read the relevant README section and nearby source before editing.
- Preserve existing config/profile behavior in `configs/*.json`, `.active-profile`, `combos/*.yaml`, and `.active-combo`.
- Preserve the public URLs documented in README: `/overlay`, `/preview`, `/gamepad`, `/ws`, and `/api/*`.
- Preserve the recommended server/client/OBS flow unless the user explicitly asks to change it.
- When changing overlay behavior, consider OBS browser source compatibility as well as normal browser preview behavior.
- When changing combo behavior, keep existing YAML schema compatibility where tests or README document it.
- When README and implementation differ, report the mismatch before changing public behavior.
- Preserve README-documented behavior unless the task explicitly asks to update the behavior and README together.
- If repository behavior is unclear, inspect the implementation or ask the user instead of inventing new behavior.
- Use tests first for verification and avoid leaving unnecessary binaries in the repository.
- Do not change source code merely to work around missing local native dependencies.
- Do not overwrite user-edited files or generated artifacts unless the task specifically requires it.

## Prohibited Changes

- Do not add game-process hooks, game-memory reads, rendering API hooks, automatic inputs, macros, or input correction.
- Do not change endpoint paths, config field meanings, combo YAML semantics, or profile persistence casually.
- Do not commit secrets, local `.env` values, device-specific paths, or private data.
- Do not perform broad refactors while fixing a narrow issue.
- Do not delete existing configs, combos, assets, or active marker files unless explicitly requested.
- Do not remove generated build outputs such as `dist/` unless the task is specifically about packaging or cleanup.

## Tests

Run commands from `/home/naoki/develop/input-cast` through Debian WSL.

```bash
wsl.exe -d Debian bash -lc 'cd /home/naoki/develop/input-cast && go test ./...'
```

For overlay JavaScript tests:

```bash
wsl.exe -d Debian bash -lc 'cd /home/naoki/develop/input-cast && node web/overlay/overlay.test.js'
```

TODO: Document any additional frontend or integration test command if a package manager configuration is added later.

## Build Artifacts

- The primary Windows distributable artifact is the Input Cast Client executable.
- Linux distributable artifacts are generated through `packaging/linux/build-package.sh`.
- When producing a handoff or release binary, write the Windows client executable under `dist/`.
- Do not overwrite existing artifacts in `dist/` unless rebuilding the same target is part of the requested task.
- Do not build or keep a server binary unless explicitly requested.
- For verification-only builds, use tests first and avoid leaving unnecessary binaries in the repository.

Windows client build:

```bash
wsl.exe -d Debian bash -lc 'cd /home/naoki/develop/input-cast && mkdir -p dist && CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui" -o dist/input-cast-client.exe ./cmd/input-cast-client'
```

If Go client builds fail due to missing native GUI/input dependencies, report the missing dependency instead of changing source code to bypass it.

For Linux package changes, README documents:

```bash
wsl.exe -d Debian bash -lc 'cd /home/naoki/develop/input-cast && packaging/linux/build-package.sh'
```

## Refactoring Notes

- Prefer local, behavior-preserving edits over new abstractions.
- Keep platform-specific input behavior separated by existing build-tag file boundaries.
- Keep server-side config/combo parsing compatible with existing tests and README examples.
- Keep overlay display logic compatible with both live WebSocket input and mocked/tested state.
- Do not rename public config fields or combo YAML keys without migration and compatibility coverage.

## Review Checklist

- Safety policy remains intact: no game process, memory, rendering API, macro, or automatic-input behavior.
- Existing OBS/preview/browser-client URLs still work.
- Existing config profiles can still be loaded, saved, listed, and switched.
- Existing combo YAML files and README examples still parse.
- Existing WebSocket message shapes remain compatible with overlay/client code.
- Input history behavior, including `history.max_entries` and `history.copy_max_entries`, remains compatible.
- Cross-platform files still build under their intended OS/build-tag constraints.
- No unnecessary server binary was generated; Windows client artifacts are placed under `dist/` only when requested.
- README remains accurate, or any required documentation updates are included in the change.
- Tests and relevant build commands were run, or any skipped command is explicitly reported with the reason.
