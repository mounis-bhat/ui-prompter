# UI Prompter - Project Plan & Intent

## The Core Problem
Generating UI code using AI (Cursor, Copilot, Claude) directly from screenshots is expensive (high token usage), slow, and often results in hallucinations regarding layout structure, hex colors, and typography. The AI often fails to leverage the existing project's design system or reusable components.

## The Solution
**UI Prompter** is a local, context-aware web tool that distills heavy visual designs (images or Figma files) into a lightweight, token-efficient, and highly structural text prompt (a "Design Blueprint"). It then enriches this blueprint with the target project's local context so the coding LLM knows exactly what to build and what existing components/styles to use.

---

## Architectural Pillars

### 1. The Input Engines
*   **Figma Engine:** (For 100% Accuracy) Takes a Figma Frame URL. The Go backend uses the Figma API to fetch the raw node tree and parses it down to a structural Markdown blueprint. Auto-layout translates directly to CSS flexbox.
*   **Vision Engine:** (For Screenshots/Images) Takes an uploaded image. Uses a Vision LLM API to extract the layout structure and design tokens.
    *   **Model Agnostic Support:** Users can select their preferred provider and input their own API keys. Supported models will include the latest vision-capable models:
        *   Gemini (e.g., Gemini 3.5 Pro)
        *   OpenAI (e.g., GPT-5.5)
        *   Anthropic (e.g., Claude Fable 5)

### 2. The Local Context Engine
*   **Tech Stack Discovery:** Scans the target project directory (e.g., `package.json`) to identify frameworks (Next.js, React, Vue, Tailwind).
*   **Design System Extraction:** Parses `tailwind.config.js` or global CSS files to map generic hex colors to actual project tokens (e.g., mapping `#4f46e5` to `bg-brand-primary`).
*   **Component Discovery:** Scans `src/components` to find existing components and their props via basic AST/Regex so the AI doesn't rebuild them from scratch.

### 3. The Compiler & Output
*   Merges the *Design Blueprint* and the *Local Context*.
*   Outputs a ready-to-use prompt.
*   **Future Feature:** Can write directly to an `.ai-intent.md` file in the target project directory for zero-friction integration with AI IDEs.

### 4. Storage & Persistence
*   **Database:** SQLite in WAL mode (using `modernc.org/sqlite` + SQLC + Goose).
*   **Use Cases:** 
    *   **Caching:** Store responses for hashed images or Figma URLs to save API costs.
    *   **Configuration:** Store user-selected models and API keys (Figma PAT, OpenAI/Gemini/Anthropic Keys) locally.
    *   **History:** Keep a log of generated prompts to prevent accidental data loss.

---

## Build Phases & Sequence

### Phase 1: Foundation & Configuration (Current)
*   **Goal:** Setup basic UI, API configurations, and model selection.
*   **Steps:**
    1.  ~~Clean up old boilerplate and refactor imports.~~ (Done)
    2.  ~~Update DB schema for `config`, `cache`, and `history`.~~ (Done)
    3.  ~~Create a "Settings" UI tab where users can input/save their API keys (Figma, OpenAI, Anthropic, Gemini) and select their default Vision Model.~~ (Done)
    4.  ~~Implement Go handlers to save/load these configurations to the SQLite DB.~~ (Done)

### Phase 2: The Vision Engine
*   **Goal:** Allow users to upload an image and get a structured Markdown blueprint.
*   **Steps:**
    1.  ~~Build the backend interface to communicate with the three Vision APIs (OpenAI, Anthropic, Gemini).~~ (Done)
    2.  ~~Write a highly strict extraction prompt optimized for generating structural UI blueprints.~~ (Done)
    3.  ~~Implement the image upload handler, pass the image to the selected model, and return the Markdown response.~~ (Done)
    4.  ~~Implement caching (hash the image and store the LLM response to save costs on retries).~~ (Done)
    5.  ~~Display the generated markdown blueprint in the UI.~~ (Done)

### Phase 3: The Figma Engine
*   **Goal:** Parse Figma designs directly for 100% accuracy.
*   **Steps:**
    1.  ~~Implement the Figma API client to fetch the node tree from a provided URL.~~ (Done)
    2.  ~~Build a Go parser to convert Figma Nodes (Frames, Text, Auto-layout) into CSS flexbox/structural equivalents.~~ (Done)
    3.  ~~Format the parsed output into the standardized Markdown blueprint format.~~ (Done)

### Phase 4: The Local Context Engine
*   **Goal:** Enrich blueprints with the user's local project context.
*   **Steps:**
    1.  ~~Add a UI field to input the local target directory (e.g., `/Users/dev/my-project`).~~ (Done)
    2.  ~~Implement Go logic to scan the directory for `package.json` (Tech Stack).~~ (Done)
    3.  ~~Implement parser for `tailwind.config.js` to extract color mappings.~~ (Done)
    4.  ~~*(Stretch)* Implement simple AST/regex scanning for existing components in `src/components`.~~ (Done)

### Phase 5: The Compiler & Polish
*   **Goal:** Combine outputs and provide a seamless UX.
*   **Steps:**
    1.  ~~Merge the Vision/Figma Blueprint with the Local Context.~~ (Done)
    2.  ~~Polish the UI: syntax highlighting for the output prompt, "Copy to Clipboard" buttons, and History view.~~ (Done)
### Phase 6: Advanced Prompt Compilation & Figma Polish
*   **Goal:** Create a production-ready, highly-structured LLM prompt and refine the Figma parser to output clean, semantic HTML-like structures instead of noisy vectors.
*   **Steps:**
    1.  ~~**Figma Parser Refactor:** Update `internal/figma/parser.go` to filter out noise (`VECTOR`, `ELLIPSE`, redundant `GROUP` or `RECTANGLE` nodes). Intelligently collapse wrappers so the output resembles clean semantic components (like the "User Profile Card" example) rather than raw Figma layer names.~~ (Done)
    2.  ~~**Structured Prompt Engine:** Refactor the final output generator to assemble a complete AI Prompt. This prompt must include:~~ (Done)
        *   ~~**Objective:** An info paragraph explaining the goal.~~ (Done)
        *   ~~**Execution Sequence:** A step-by-step guide for the LLM to follow (e.g., "1. Scaffold structure, 2. Apply styles, 3. Integrate components").~~ (Done)
        *   ~~**Guardrails/Restrictions:** Rules the LLM must obey (e.g., "Do not use inline styles", "Do not hallucinate colors").~~ (Done)
        *   ~~**The Blueprint:** The refined structural layout from Vision/Figma.~~ (Done)
    3.  ~~**Local Workspace Deep Context (Linux):** Enhance the context scanner to provide deep workspace awareness. Since we are starting with Linux, implement a feature that executes a local shell command (e.g., `find` or `tree`) on the target directory's design system folder to provide the LLM with the exact file structure and paths of the existing UI components.~~ (Done)
