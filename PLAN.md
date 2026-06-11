# UI Prompter - Project Plan & Intent

## The Core Problem
Generating UI code using AI (Cursor, Copilot, Claude) directly from screenshots is expensive (high token usage), slow, and often results in hallucinations regarding layout structure, hex colors, and typography. The AI often fails to leverage the existing project's design system or reusable components.

## The Solution
**UI Prompter** is a local, context-aware web tool that distills heavy visual designs (images or Figma files) into a lightweight, token-efficient, and highly structural text prompt (a "Design Blueprint"). It then enriches this blueprint with the target project's local context so the coding LLM knows exactly what to build and what existing components/styles to use.

---

## Architectural Pillars

### 1. The Input Engines
*   **Figma Engine:** (For 100% Accuracy) Takes a Figma Frame URL. The Go backend uses the Figma API to fetch the raw node tree and parses it down to a structural Markdown blueprint. Auto-layout translates directly to CSS flexbox.
*   **Vision Engine:** (For Screenshots/Images) Takes an uploaded image. Uses a Vision LLM API (e.g., Gemini) with a highly strict extraction prompt to output the structural Markdown blueprint.

### 2. The Local Context Engine
*   **Tech Stack Discovery:** Scans the target project directory (e.g., `package.json`) to identify frameworks (Next.js, React, Vue, Tailwind).
*   **Design System Extraction:** Parses `tailwind.config.js` or global CSS files to map generic hex colors to actual project tokens (e.g., mapping `#4f46e5` to `bg-brand-primary`).
*   **Component Discovery:** Scans `src/components` to find existing components and their props via basic AST/Regex (e.g., recognizing `<Button variant="outline">`) so the AI doesn't rebuild them from scratch.

### 3. The Compiler & Output
*   Merges the *Design Blueprint* and the *Local Context*.
*   Outputs a ready-to-use prompt.
*   **Future Feature:** Can write directly to an `.ai-intent.md` (or `.cursorrules`) file in the target project directory for zero-friction integration with AI IDEs.

### 4. Storage & Persistence
*   **Database:** SQLite in WAL mode (using `modernc.org/sqlite` + SQLC + Goose).
*   **Use Cases:** 
    *   **Caching:** Store responses for hashed images or Figma URLs to save API costs and time.
    *   **Configuration:** Store API keys (Figma PAT, Gemini Key) locally.
    *   **History:** Keep a log of generated prompts to prevent accidental data loss.

---

## Current State of Repository
*   **Boilerplate Cloned:** Copied the starter boilerplate (Go 1.22 net/http routing, SQLite setup, HTML templates) from the previous `ssr` project into this new directory (`ui-prompter`).
*   **Module Renamed:** `go.mod` is updated to `module ui-prompter`.
*   **Database Cleared:** The old `notes.db` was removed.

## Immediate Next Steps (For the Next Session)
1.  **Refactor Imports:** Do a global find/replace in the `.go` files to change `ssr/internal/...` imports to `ui-prompter/internal/...`.
2.  **Update Database Schema:** 
    *   Delete the old `notes.sql` queries and migration files.
    *   Create new migrations for `config`, `history`, and `cache` tables.
    *   Regenerate the type-safe DB code using SQLC.
3.  **Frontend Overhaul:** Replace the notes UI in `home.html` with a clean, dual-tab layout ("Figma URL" and "Upload Image").
4.  **Backend Routes:** Create handlers to receive the Figma URL and image uploads.
