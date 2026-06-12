# UI Prompter

A context-aware, dual-engine UI Prompt Generator designed to dramatically improve the workflow of building interfaces with AI coding assistants.

## Overview

Sending screenshots of designs directly to coding LLMs (like Cursor, Copilot, or Claude) is token-heavy, slow, and often results in inaccurate colors, spacings, and hallucinations. **UI Prompter** solves this by acting as a middle-layer that distills heavy visual designs into lightweight, highly accurate "Design Blueprint" markdowns.

Once compiled, it saves an instructional intent file (`intent.md`) and the required visual assets straight into your project folder. You then hand that folder off to your AI coding assistant for a flawless build.

### The Core Architecture

1. **Dual-Input Engines**
   - **Figma Engine:** Paste one or more Figma frame URLs. The tool fetches each node tree using the Figma API and parses it into flexbox-equivalent layouts with 100% accurate colors, typography, and structure.
     - **Responsive Variants:** Add up to 5 URLs (e.g., desktop, tablet, and mobile frames of the same page). The frames are merged into a *single unified responsive blueprint*, with breakpoints inferred from the actual frame widths and per-element annotations like "stacks vertically below 768px".
   - **Vision Engine:** Upload a design screenshot or mockup. The tool uses a Vision LLM (Gemini, OpenAI, or Anthropic — auto-selecting the best available model on your key) to extract layout structures, color palettes, and design tokens into a standardized blueprint. It also scans your target project (frameworks, Tailwind tokens, existing components) and injects that context into the prompt.

2. **The Output Compiler**
   - Compiles the blueprint and writes a highly targeted instruction prompt into your designated project folder (e.g., `ui-prompter-plan-xxxxxx/intent.md`).
   - Downloads the raster image assets (`png`/`jpg` only — SVGs/vectors are intentionally skipped to stay clear of Figma's rate limits) into `ui-prompter-plan-xxxxxx/assets/`.
   - Saves a full-design screenshot per variant (`design.png`, or `design_1_desktop.png`, `design_2_mobile.png`, ...) as a *visual reference only* — the generated intent explicitly instructs the coding agent not to copy these into the project.

## Tech Stack

- **Backend:** Go (`net/http`)
- **Database:** SQLite (`modernc.org/sqlite`) + SQLC + Goose. Caches generated blueprints keyed by stable Figma file/node IDs so repeat runs are instant and free.
- **Frontend SPA:** Pure HTML/JS/CSS embedded directly into the Go binary via `embed.FS`.

## Features

- **True SPA Experience:** Built using a custom frontend router with hash-based navigation.
- **Native View Transitions:** Leverages the modern `document.startViewTransition()` API for buttery smooth page swaps.
- **Modern Dark UI:** Indigo/violet accent system, glassy sidebar, refined cards and inputs.
- **Toast Notifications:** Non-blocking pill toasts for loading states (with rotating status phrases), successes, and errors — no full-screen overlays.
- **Honest Asset Pipeline:** Figma render URLs expire quickly, so the app never caches them. It stores node IDs and fetches fresh URLs only at save time (one `images` API call per Figma file), verifying every download's HTTP status and surfacing per-asset warnings instead of failing silently.
- **State Management:** Fully reactive JSON-based API integration that updates UI status bars and input locks on the fly without ever reloading the browser window.
- **Project-Aware Prompts:** Scans the target directory for frameworks, Tailwind color tokens, and existing components so blueprints reference your real design system.

## Getting Started

1. Ensure you have Go 1.22+ installed.
2. Clone the repository and navigate to the project directory.
3. Run `go run ./cmd/server` to start the application.
4. Navigate to `http://localhost:8080`.
5. Enter your Figma API key and your preferred Vision LLM API key in the **Settings** tab, and set your target project directory.
6. Start generating UI Blueprints!

> **Note:** UI Prompter is a local, single-user tool. Keep it bound to localhost — API keys are stored in the local SQLite database and surfaced to the settings UI.
