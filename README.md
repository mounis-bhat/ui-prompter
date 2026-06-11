# UI Prompter

A context-aware, dual-engine UI Prompt Generator designed to dramatically improve the workflow of building interfaces with AI coding assistants.

## Overview

Sending screenshots of designs directly to coding LLMs (like Cursor, Copilot, or Claude) is token-heavy, slow, and often results in inaccurate colors, spacings, and hallucinations. **UI Prompter** solves this by acting as a middle-layer that distills heavy visual designs into lightweight, highly accurate "Design Blueprint" markdowns. 

Once compiled, it saves an instructional intent file (`.ai-intent.md`) and the required visual assets straight into your project folder. You then hand that folder off to your AI coding assistant for a flawless build.

### The Core Architecture

1. **Dual-Input Engines**
   - **Figma Engine:** Paste a Figma URL. The tool fetches the node tree using the Figma API and parses it into flexbox-equivalent layouts with 100% accurate colors, typography, and structure. It natively handles throttled downloads of `png` and `jpg` assets to bypass Figma's rate limits while ignoring SVGs you already have natively.
   - **Vision Engine:** Upload a design screenshot or mockup. The tool uses a Vision LLM (like Gemini, OpenAI, or Anthropic) to extract layout structures, color palettes, and design tokens into a standardized blueprint.

2. **The Output Compiler**
   - Compiles the blueprint and writes a highly targeted instruction prompt into your designated project folder (e.g., `ui-prompter-plan/intent.md`).
   - Automatically downloads, routes, and saves all necessary image assets to `ui-prompter-plan/assets` or injects the reference image directly.

## Tech Stack

- **Backend:** Go (`net/http`)
- **Database:** SQLite (`modernc.org/sqlite`) + SQLC + Goose. Caches heavy API responses, generated blueprints, and Figma asset URLs so you don't hit rate limits on reloads.
- **Frontend SPA:** Pure HTML/JS/CSS embedded directly into the Go binary.

## Features

- **True SPA Experience:** Built using a custom frontend router with Hash-based navigation.
- **Native View Transitions:** Leverages the modern `document.startViewTransition()` API for buttery smooth page swaps, text-scoped crossfades, and animations.
- **Stunning UI/UX:** A true dark-mode aesthetic with glassmorphism, responsive dashboard layouts, custom styled checkboxes, and dynamic rotating loading phrases.
- **State Management:** Fully reactive JSON-based API integration that updates UI status bars and input locks on the fly without ever reloading the browser window.
- **Smart Throttling:** Built-in sleep cycles and SVG-skipping logic to navigate strict external API rate limits securely.

## Getting Started

1. Ensure you have Go 1.22+ installed.
2. Clone the repository and navigate to the project directory.
3. Run `go run ./cmd/server` to start the application.
4. Navigate to `http://localhost:8080`.
5. Enter your Figma API key and your preferred Vision LLM API key in the **Settings** tab.
6. Start generating UI Blueprints!
