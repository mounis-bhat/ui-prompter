package handlers

import (
	gocontext "context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqweek/dialog"

	"ui-prompter/internal/context"
	"ui-prompter/internal/db"
	"ui-prompter/internal/db/queries"
	"ui-prompter/internal/figma"
	"ui-prompter/internal/vision"
	"ui-prompter/ui"
)

type App struct {
	db       *db.Database
	homeTmpl *template.Template
}

func NewApp(database *db.Database) *App {
	return &App{
		db:       database,
		homeTmpl: template.Must(template.ParseFS(ui.Files, "templates/home.html")),
	}
}

func (a *App) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", a.homeHandler)
	mux.HandleFunc("POST /api/figma", a.figmaHandler)
	mux.HandleFunc("POST /api/image", a.imageHandler)
	mux.HandleFunc("POST /api/config", a.saveConfigHandler)
	mux.HandleFunc("POST /api/save", a.saveIntentHandler)
	mux.HandleFunc("GET /api/pick-dir", a.pickDirHandler)

	// Serve static files
	mux.Handle("GET /static/", http.FileServer(http.FS(ui.Files)))
}

// figmaAssetMeta identifies a downloadable raster asset by its Figma node
// ID. Fresh render URLs are fetched at save time because Figma's S3 URLs
// expire and must not be cached.
type figmaAssetMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// figmaFileAssets groups asset and design-screenshot nodes per Figma file,
// since responsive variants may live in different files.
type figmaFileAssets struct {
	FileKey string           `json:"fileKey"`
	Assets  []figmaAssetMeta `json:"assets,omitempty"`
	Designs []figmaAssetMeta `json:"designs,omitempty"`
}

type figmaAssetsPayload struct {
	Files []figmaFileAssets `json:"files,omitempty"`

	// Legacy single-file fields, kept so older cached payloads still work.
	FileKey      string           `json:"fileKey,omitempty"`
	DesignNodeID string           `json:"designNodeId,omitempty"`
	Assets       []figmaAssetMeta `json:"assets,omitempty"`
}

const maxFigmaURLs = 5

type HomeData struct {
	FigmaKey         string
	OpenAIKey        string
	AnthropicKey     string
	GeminiKey        string
	DefaultModel     string
	ModelDisplayName string
	TargetDir        string
	VisionReady      bool
	SuccessMessage   string
	Result           string
	Error            string
	ImageHash        string
	ImageExt         string
	FigmaAssets      string
}

func (a *App) getHomeData(r *http.Request) HomeData {
	ctx := r.Context()
	getConfig := func(key string) string {
		val, _ := a.db.Queries.GetConfig(ctx, key)
		return val
	}

	defaultModel := getConfig("default_model")
	if defaultModel == "" {
		defaultModel = "gemini"
	}

	var visionReady bool
	var modelDisplay string
	switch defaultModel {
	case "openai":
		visionReady = getConfig("openai_key") != ""
		modelDisplay = "OpenAI (Best Available)"
	case "anthropic":
		visionReady = getConfig("anthropic_key") != ""
		modelDisplay = "Anthropic (Best Available)"
	case "gemini":
		visionReady = getConfig("gemini_key") != ""
		modelDisplay = "Gemini (Best Available)"
	}

	data := HomeData{
		FigmaKey:         getConfig("figma_key"),
		OpenAIKey:        getConfig("openai_key"),
		AnthropicKey:     getConfig("anthropic_key"),
		GeminiKey:        getConfig("gemini_key"),
		DefaultModel:     defaultModel,
		ModelDisplayName: modelDisplay,
		TargetDir:        getConfig("target_dir"),
		VisionReady:      visionReady,
	}
	return data
}

func (a *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	data := a.getHomeData(r)

	if r.URL.Query().Get("success") == "1" {
		data.SuccessMessage = "Settings saved successfully!"
	}

	if err := a.homeTmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

// renderData writes HomeData either as JSON (for fetch requests) or as the
// rendered home template (no-JS fallback).
func (a *App) renderData(w http.ResponseWriter, r *http.Request, data HomeData) {
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}
	a.homeTmpl.Execute(w, data)
}

// sanitizeFileName converts an arbitrary Figma node name into a safe,
// filesystem-friendly slug.
func sanitizeFileName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteRune('_')
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		s = "variant"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func (a *App) figmaHandler(w http.ResponseWriter, r *http.Request) {
	data := a.getHomeData(r)

	if err := r.ParseForm(); err != nil {
		data.Error = "Unable to parse form"
		a.renderData(w, r, data)
		return
	}

	// Collect submitted URLs. Multiple URLs are treated as responsive
	// variants of the same page/component.
	var figmaURLs []string
	for _, u := range r.Form["figma_url"] {
		u = strings.TrimSpace(u)
		if u != "" {
			figmaURLs = append(figmaURLs, u)
		}
	}

	if len(figmaURLs) == 0 {
		data.Error = "At least one Figma URL is required"
		a.renderData(w, r, data)
		return
	}
	if len(figmaURLs) > maxFigmaURLs {
		data.Error = fmt.Sprintf("A maximum of %d Figma URLs is supported per blueprint.", maxFigmaURLs)
		a.renderData(w, r, data)
		return
	}

	ctx := r.Context()
	figmaKey := data.FigmaKey
	if figmaKey == "" {
		data.Error = "Figma API Key (PAT) is missing. Please configure it in Settings."
		a.renderData(w, r, data)
		return
	}

	type nodeRef struct {
		fileKey string
		nodeID  string
	}
	var refs []nodeRef
	seenRef := make(map[string]bool)
	for i, u := range figmaURLs {
		fileKey, nodeID, err := figma.ExtractFileKeyAndNodeID(u)
		if err != nil {
			data.Error = fmt.Sprintf("URL %d: %s", i+1, err.Error())
			a.renderData(w, r, data)
			return
		}
		key := fileKey + ":" + nodeID
		if seenRef[key] {
			continue // ignore duplicate links
		}
		seenRef[key] = true
		refs = append(refs, nodeRef{fileKey: fileKey, nodeID: nodeID})
	}

	// Key the cache on the stable file key + node ID pairs so volatile
	// query params (e.g. the "t" token) don't cause cache misses.
	var keyParts []string
	for _, ref := range refs {
		keyParts = append(keyParts, ref.fileKey+":"+ref.nodeID)
	}
	hashBytes := sha256.Sum256([]byte("figma:" + strings.Join(keyParts, "|")))
	hashStr := hex.EncodeToString(hashBytes[:])

	if cachedResp, err := a.db.Queries.GetCache(ctx, hashStr); err == nil && cachedResp != "" {
		data.Result = cachedResp
		if cachedAssets, err := a.db.Queries.GetCache(ctx, hashStr+"_assets"); err == nil && cachedAssets != "" {
			data.FigmaAssets = cachedAssets
		}
		a.renderData(w, r, data)
		return
	}

	client := figma.NewClient(figmaKey)
	nodes := make([]*figma.Node, 0, len(refs))
	for i, ref := range refs {
		node, err := client.GetNode(ref.fileKey, ref.nodeID)
		if err != nil {
			data.Error = fmt.Sprintf("Failed to fetch Figma node for URL %d: %s", i+1, err.Error())
			a.renderData(w, r, data)
			return
		}
		nodes = append(nodes, node)
	}

	// Collect downloadable asset metadata (node IDs only, never URLs).
	// Figma render URLs expire quickly, so fresh URLs are fetched at save
	// time instead. Only raster images (png/jpg) are included; SVGs and
	// vectors are intentionally excluded to avoid hammering the Figma API.
	payload := figmaAssetsPayload{}
	fileIdx := make(map[string]int)
	fileGroup := func(fileKey string) *figmaFileAssets {
		if idx, ok := fileIdx[fileKey]; ok {
			return &payload.Files[idx]
		}
		payload.Files = append(payload.Files, figmaFileAssets{FileKey: fileKey})
		fileIdx[fileKey] = len(payload.Files) - 1
		return &payload.Files[len(payload.Files)-1]
	}

	seenAsset := make(map[string]bool)
	for i, ref := range refs {
		group := fileGroup(ref.fileKey)

		for _, ast := range figma.ExtractAssets(nodes[i]) {
			if ast.Format != "png" && ast.Format != "jpg" && ast.Format != "jpeg" {
				continue
			}
			key := ref.fileKey + ":" + ast.ID
			if seenAsset[key] {
				continue
			}
			seenAsset[key] = true
			group.Assets = append(group.Assets, figmaAssetMeta{
				ID:   ast.ID,
				Name: ast.Name + "." + ast.Format,
			})
		}

		designName := "design.png"
		if len(refs) > 1 {
			designName = fmt.Sprintf("design_%d_%s.png", i+1, sanitizeFileName(nodes[i].Name))
		}
		group.Designs = append(group.Designs, figmaAssetMeta{
			ID:   ref.nodeID,
			Name: designName,
		})
	}

	if b, err := json.Marshal(payload); err == nil {
		data.FigmaAssets = string(b)
	}

	// Build the markdown dump. Multiple frames are labeled as variants of
	// the same page so the LLM merges them into one responsive blueprint.
	var markdown string
	systemPrompt := figma.SystemPrompt
	if len(refs) == 1 {
		markdown = figma.ParseNodeToMarkdown(nodes[0], 0)
	} else {
		systemPrompt += figma.ResponsiveSystemAddendum
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("The following %d frames are responsive variants of the SAME page/component at different breakpoints.\n\n", len(refs)))
		for i, node := range nodes {
			width := ""
			if node.BoundingBox != nil && node.BoundingBox.Width > 0 {
				width = fmt.Sprintf(", width: %.0fpx", node.BoundingBox.Width)
			}
			sb.WriteString(fmt.Sprintf("===== VARIANT %d: %q%s =====\n\n", i+1, node.Name, width))
			sb.WriteString(figma.ParseNodeToMarkdown(node, 0))
			sb.WriteString("\n")
		}
		markdown = sb.String()
	}

	defaultModel := data.DefaultModel
	apiKey := ""
	switch defaultModel {
	case "openai":
		apiKey = data.OpenAIKey
	case "anthropic":
		apiKey = data.AnthropicKey
	case "gemini":
		apiKey = data.GeminiKey
	}

	if apiKey == "" {
		data.Error = fmt.Sprintf("API Key for %s is missing. Please configure it in Settings to polish the Figma output.", defaultModel)
		a.renderData(w, r, data)
		return
	}

	provider, err := vision.NewProvider(defaultModel, apiKey)
	if err != nil {
		data.Error = "Error initializing LLM provider: " + err.Error()
		a.renderData(w, r, data)
		return
	}

	finalPrompt, err := provider.GenerateText(ctx, systemPrompt, markdown)
	if err != nil {
		data.Error = "Error generating prompt via LLM: " + err.Error()
		a.renderData(w, r, data)
		return
	}

	_ = a.db.Queries.SetCache(ctx, queries.SetCacheParams{
		Hash:     hashStr,
		Response: finalPrompt,
	})

	if data.FigmaAssets != "" {
		_ = a.db.Queries.SetCache(ctx, queries.SetCacheParams{
			Hash:     hashStr + "_assets",
			Response: data.FigmaAssets,
		})
	}

	_, _ = a.db.Queries.AddHistory(ctx, queries.AddHistoryParams{
		SourceType: "figma",
		SourceUri:  strings.Join(figmaURLs, " | "),
		Prompt:     finalPrompt,
	})

	data.Result = finalPrompt
	a.renderData(w, r, data)
}

func (a *App) imageHandler(w http.ResponseWriter, r *http.Request) {
	data := a.getHomeData(r)

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		data.Error = "Unable to parse form"
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
		a.homeTmpl.Execute(w, data)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		data.Error = "Unable to read image"
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
		a.homeTmpl.Execute(w, data)
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		data.Error = "Unable to read image bytes"
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
		a.homeTmpl.Execute(w, data)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/png"
	}

	hashBytes := sha256.Sum256(imgBytes)
	hashStr := hex.EncodeToString(hashBytes[:])

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		if mimeType == "image/jpeg" {
			ext = ".jpg"
		} else {
			ext = ".png"
		}
	}
	tempImgPath := filepath.Join(os.TempDir(), "ui-prompter-"+hashStr+ext)
	_ = os.WriteFile(tempImgPath, imgBytes, 0644)

	ctx := r.Context()
	if cachedResp, err := a.db.Queries.GetCache(ctx, hashStr); err == nil && cachedResp != "" {
		data.Result = cachedResp
		data.ImageHash = hashStr
		data.ImageExt = ext
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
		a.homeTmpl.Execute(w, data)
		return
	}

	defaultModel := data.DefaultModel
	if defaultModel == "" {
		defaultModel = "gemini"
	}

	var apiKey string
	switch defaultModel {
	case "openai":
		apiKey = data.OpenAIKey
	case "anthropic":
		apiKey = data.AnthropicKey
	case "gemini":
		apiKey = data.GeminiKey
	}

	if apiKey == "" {
		data.Error = fmt.Sprintf("API key for %s is missing. Please configure it in Settings.", defaultModel)
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
		a.homeTmpl.Execute(w, data)
		return
	}

	provider, err := vision.NewProvider(defaultModel, apiKey)
	if err != nil {
		data.Error = err.Error()
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
		a.homeTmpl.Execute(w, data)
		return
	}

	base64Image := base64.StdEncoding.EncodeToString(imgBytes)

	pc, _ := context.ScanContext(data.TargetDir)
	contextPrompt := ""
	if pc != nil {
		contextPrompt = pc.FormatForLLM()
	}

	respText, err := provider.AnalyzeImage(ctx, base64Image, mimeType, contextPrompt)
	if err != nil {
		data.Error = err.Error()
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
			return
		}
		a.homeTmpl.Execute(w, data)
		return
	}

	_ = a.db.Queries.SetCache(ctx, queries.SetCacheParams{
		Hash:     hashStr,
		Response: respText,
	})

	_, _ = a.db.Queries.AddHistory(ctx, queries.AddHistoryParams{
		SourceType: "image",
		SourceUri:  "upload",
		Prompt:     respText,
	})

	data.Result = respText
	data.ImageHash = hashStr
	data.ImageExt = ext
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}
	a.homeTmpl.Execute(w, data)
}

func (a *App) saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		if r.Header.Get("Accept") == "application/json" {
			http.Error(w, `{"Error":"Failed to parse form"}`, http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	keys := []string{"figma_key", "openai_key", "anthropic_key", "gemini_key", "default_model", "target_dir"}

	for _, key := range keys {
		val := r.FormValue(key)
		err := a.db.Queries.SetConfig(ctx, queries.SetConfigParams{
			Key:   key,
			Value: val,
		})
		if err != nil {
			if r.Header.Get("Accept") == "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"Error": "Failed to save config: " + err.Error()})
				return
			}
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if r.Header.Get("Accept") == "application/json" {
		// Get fresh data
		data := a.getHomeData(r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"SuccessMessage":   "Settings saved successfully!",
			"FigmaReady":       data.FigmaKey != "",
			"VisionReady":      data.VisionReady,
			"ModelDisplayName": data.ModelDisplayName,
		})
		return
	}
	http.Redirect(w, r, "/?success=1", http.StatusSeeOther)
}

func (a *App) saveIntentHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Content is empty", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	targetDir, _ := a.db.Queries.GetConfig(ctx, "target_dir")
	if targetDir == "" {
		http.Error(w, "Target Directory is not set in Settings", http.StatusBadRequest)
		return
	}

	planDirName := filepath.Base(r.FormValue("plan_dir"))
	if planDirName == "" || planDirName == "." || planDirName == ".." || planDirName == string(filepath.Separator) {
		planDirName = "ui-prompter-plan"
	}
	planDirPath := filepath.Join(targetDir, planDirName)
	if err := os.MkdirAll(planDirPath, 0755); err != nil {
		http.Error(w, "Failed to create plan directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err := os.WriteFile(filepath.Join(planDirPath, "intent.md"), []byte(content), 0644)
	if err != nil {
		http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("attach_image") == "true" {
		hash := r.FormValue("image_hash")
		ext := r.FormValue("image_ext")
		if hash != "" {
			tempImgPath := filepath.Join(os.TempDir(), "ui-prompter-"+hash+ext)
			imgData, err := os.ReadFile(tempImgPath)
			if err == nil {
				_ = os.WriteFile(filepath.Join(planDirPath, "original_image"+ext), imgData, 0644)
			}
		}
	}

	var warnings []string

	figmaAssetsStr := r.FormValue("figma_assets")
	if figmaAssetsStr != "" {
		warnings = append(warnings, a.downloadFigmaAssets(ctx, figmaAssetsStr, planDirPath)...)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"Message":  "Saved successfully",
		"Warnings": warnings,
	})
}

// downloadFigmaAssets fetches fresh render URLs for the requested asset node
// IDs and downloads them into the plan directory. It returns a list of
// human-readable warnings for anything that failed; an empty list means full
// success.
func (a *App) downloadFigmaAssets(ctx gocontext.Context, figmaAssetsStr, planDirPath string) []string {
	var warnings []string

	var payload figmaAssetsPayload
	if err := json.Unmarshal([]byte(figmaAssetsStr), &payload); err != nil {
		return []string{"Asset data was invalid or outdated; assets were not downloaded. Re-generate the blueprint and try again."}
	}

	// Normalize legacy single-file payloads into the Files list.
	if len(payload.Files) == 0 && payload.FileKey != "" {
		legacy := figmaFileAssets{
			FileKey: payload.FileKey,
			Assets:  payload.Assets,
		}
		if payload.DesignNodeID != "" {
			legacy.Designs = []figmaAssetMeta{{ID: payload.DesignNodeID, Name: "design.png"}}
		}
		payload.Files = []figmaFileAssets{legacy}
	}

	if len(payload.Files) == 0 {
		return []string{"Asset data was invalid or outdated; assets were not downloaded. Re-generate the blueprint and try again."}
	}

	figmaKey, _ := a.db.Queries.GetConfig(ctx, "figma_key")
	if figmaKey == "" {
		return []string{"Figma API key is missing; assets were not downloaded."}
	}

	client := figma.NewClient(figmaKey)

	download := func(name, url, outPath string) {
		resp, err := http.Get(url)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to download %s: %v", name, err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			warnings = append(warnings, fmt.Sprintf("Failed to download %s: HTTP %d", name, resp.StatusCode))
			return
		}

		outFile, err := os.Create(outPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to save %s: %v", name, err))
			return
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, resp.Body); err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to write %s: %v", name, err))
		}
	}

	assetsDir := filepath.Join(planDirPath, "assets")
	assetsDirReady := false
	safeName := func(name, fallback string) string {
		name = filepath.Base(name)
		if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
			return fallback
		}
		return name
	}

	// One GetImages call per Figma file (variants usually share one file).
	for _, group := range payload.Files {
		var ids []string
		for _, ast := range group.Assets {
			ids = append(ids, ast.ID)
		}
		for _, d := range group.Designs {
			ids = append(ids, d.ID)
		}
		if len(ids) == 0 {
			continue
		}

		urls, err := client.GetImages(group.FileKey, ids, "png")
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to fetch asset URLs from Figma (file %s): %v", group.FileKey, err))
			continue
		}

		if len(group.Assets) > 0 && !assetsDirReady {
			if err := os.MkdirAll(assetsDir, 0755); err != nil {
				warnings = append(warnings, "Failed to create assets directory: "+err.Error())
			} else {
				assetsDirReady = true
			}
		}

		if assetsDirReady {
			for _, ast := range group.Assets {
				name := safeName(ast.Name, "asset.png")
				u, ok := urls[ast.ID]
				if !ok || u == "" {
					warnings = append(warnings, fmt.Sprintf("Figma did not return a render URL for %s; skipped.", name))
					continue
				}
				download(name, u, filepath.Join(assetsDir, name))
			}
		}

		for _, d := range group.Designs {
			name := safeName(d.Name, "design.png")
			u, ok := urls[d.ID]
			if !ok || u == "" {
				warnings = append(warnings, fmt.Sprintf("Figma did not return a render URL for %s; skipped.", name))
				continue
			}
			download(name, u, filepath.Join(planDirPath, name))
		}
	}

	return warnings
}

func (a *App) pickDirHandler(w http.ResponseWriter, r *http.Request) {
	dir, err := dialog.Directory().Title("Select Target Project Directory").Browse()
	if err != nil {
		if err.Error() == "Cancelled" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "Failed to pick directory: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(dir))
}
