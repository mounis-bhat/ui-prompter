# UI Prompter

A context-aware, dual-engine UI Prompt Generator designed to dramatically improve the workflow of building UIs with AI coding assistants.

## Overview

Sending screenshots of designs directly to coding LLMs (like Cursor, Copilot, or Claude) is token-heavy, slow, and often results in inaccurate colors, spacings, and hallucinations. **UI Prompter** solves this by acting as a middle-layer that distills heavy visual designs into lightweight, highly accurate "Design Blueprint" markdowns.

### The Core Architecture

1. **Dual-Input Engines**
   - **Figma Engine:** Paste a Figma URL. The tool directly fetches the node tree using the Figma API and parses it into flexbox-equivalent layouts with 100% accurate colors and text.
   - **Vision Engine:** Upload a screenshot. The tool uses a Vision LLM (like Gemini) to extract the layout structure and design tokens.

2. **Local Context Engine**
   - The tool scans your local project directory (where you intend to write the code).
   - It discovers your framework (React, Next.js, etc.), parses your `tailwind.config.js` for design tokens, and indexes your existing reusable components (e.g., `<Button>`, `<Card>`) along with their props.

3. **The Compiler**
   - Merges the extracted design blueprint with your local codebase context.
   - Outputs a highly targeted instruction prompt (e.g., to a `.ai-intent.md` file) that tells your coding LLM exactly *what* to build and *which existing tools* to use.

## Tech Stack
- **Backend:** Go (net/http routing)
- **Database:** SQLite (modernc.org/sqlite) + SQLC + Goose (for caching API responses and storing local config/keys).
- **Frontend:** HTML/JS with Go templates embedded directly into the binary.

## Next Steps
- Update `sqlc.yaml` and migrations to handle caching and config rather than notes.
- Replace notes handlers with the new UI Prompt Generator interface.
