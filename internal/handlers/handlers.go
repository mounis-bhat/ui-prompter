package handlers

import (
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
	ActiveTab        string
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
		ActiveTab:        "figma", // default
	}
	return data
}

func (a *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	data := a.getHomeData(r)

	if r.URL.Query().Get("success") == "1" {
		data.SuccessMessage = "Settings saved successfully!"
		data.ActiveTab = "settings"
	}

	if err := a.homeTmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func (a *App) figmaHandler(w http.ResponseWriter, r *http.Request) {
	data := a.getHomeData(r)
	data.ActiveTab = "figma"

	if err := r.ParseForm(); err != nil {
		data.Error = "Unable to parse form"
		a.homeTmpl.Execute(w, data)
		return
	}

	figmaURL := r.FormValue("figma_url")
	if figmaURL == "" {
		data.Error = "Figma URL is required"
		a.homeTmpl.Execute(w, data)
		return
	}

	ctx := r.Context()
	figmaKey := data.FigmaKey
	if figmaKey == "" {
		data.Error = "Figma API Key (PAT) is missing. Please configure it in Settings."
		a.homeTmpl.Execute(w, data)
		return
	}

	fileKey, nodeID, err := figma.ExtractFileKeyAndNodeID(figmaURL)
	if err != nil {
		data.Error = err.Error()
		a.homeTmpl.Execute(w, data)
		return
	}

	hashBytes := sha256.Sum256([]byte("figma:" + figmaURL))
	hashStr := hex.EncodeToString(hashBytes[:])

	if cachedResp, err := a.db.Queries.GetCache(ctx, hashStr); err == nil && cachedResp != "" {
		data.Result = cachedResp
		a.homeTmpl.Execute(w, data)
		return
	}

	client := figma.NewClient(figmaKey)
	node, err := client.GetNode(fileKey, nodeID)
	if err != nil {
		data.Error = err.Error()
		a.homeTmpl.Execute(w, data)
		return
	}

	markdown := figma.ParseNodeToMarkdown(node, 0)

	defaultModel := a.getHomeData(r).DefaultModel
	apiKey := ""
	switch defaultModel {
	case "openai":
		apiKey = a.getHomeData(r).OpenAIKey
	case "anthropic":
		apiKey = a.getHomeData(r).AnthropicKey
	case "gemini":
		apiKey = a.getHomeData(r).GeminiKey
	}

	if apiKey == "" {
		data.Error = fmt.Sprintf("API Key for %s is missing. Please configure it in Settings to polish the Figma output.", defaultModel)
		a.homeTmpl.Execute(w, data)
		return
	}

	provider, err := vision.NewProvider(defaultModel, apiKey)
	if err != nil {
		data.Error = "Error initializing LLM provider: " + err.Error()
		a.homeTmpl.Execute(w, data)
		return
	}

	finalPrompt, err := provider.GenerateText(ctx, figma.SystemPrompt, markdown)
	if err != nil {
		data.Error = "Error generating prompt via LLM: " + err.Error()
		a.homeTmpl.Execute(w, data)
		return
	}

	_ = a.db.Queries.SetCache(ctx, queries.SetCacheParams{
		Hash:     hashStr,
		Response: finalPrompt,
	})

	_, _ = a.db.Queries.AddHistory(ctx, queries.AddHistoryParams{
		SourceType: "figma",
		SourceUri:  figmaURL,
		Prompt:     finalPrompt,
	})

	// Extract Assets
	assets := figma.ExtractAssets(node)
	var svgIDs []string
	var pngIDs []string
	for _, ast := range assets {
		if ast.Format == "svg" {
			svgIDs = append(svgIDs, ast.ID)
		} else {
			pngIDs = append(pngIDs, ast.ID)
		}
	}

	pngIDs = append(pngIDs, nodeID)

	svgURLs, _ := client.GetImages(fileKey, svgIDs, "svg")
	pngURLs, _ := client.GetImages(fileKey, pngIDs, "png")

	type DownloadableAsset struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	var downloadAssets []DownloadableAsset
	for _, ast := range assets {
		var u string
		if ast.Format == "svg" {
			u = svgURLs[ast.ID]
		} else {
			u = pngURLs[ast.ID]
		}
		if u != "" {
			downloadAssets = append(downloadAssets, DownloadableAsset{
				Name: ast.Name + "." + ast.Format,
				URL:  u,
			})
		}
	}

	if designURL, ok := pngURLs[nodeID]; ok && designURL != "" {
		downloadAssets = append(downloadAssets, DownloadableAsset{
			Name: "design.png",
			URL:  designURL,
		})
	}

	if len(downloadAssets) > 0 {
		b, _ := json.Marshal(downloadAssets)
		data.FigmaAssets = string(b)
	}

	data.Result = finalPrompt
	a.homeTmpl.Execute(w, data)
}

func (a *App) imageHandler(w http.ResponseWriter, r *http.Request) {
	data := a.getHomeData(r)
	data.ActiveTab = "image"

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		data.Error = "Unable to parse form"
		a.homeTmpl.Execute(w, data)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		data.Error = "Unable to read image"
		a.homeTmpl.Execute(w, data)
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		data.Error = "Unable to read image bytes"
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
		a.homeTmpl.Execute(w, data)
		return
	}

	provider, err := vision.NewProvider(defaultModel, apiKey)
	if err != nil {
		data.Error = err.Error()
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
	a.homeTmpl.Execute(w, data)
}

func (a *App) saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
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
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}
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

	err := os.WriteFile(filepath.Join(targetDir, ".ai-intent.md"), []byte(content), 0644)
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
				_ = os.WriteFile(filepath.Join(targetDir, "original_image"+ext), imgData, 0644)
			}
		}
	}

	figmaAssetsStr := r.FormValue("figma_assets")
	if figmaAssetsStr != "" {
		var assets []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		if err := json.Unmarshal([]byte(figmaAssetsStr), &assets); err == nil {
			assetsDir := filepath.Join(targetDir, "assets")
			os.MkdirAll(assetsDir, 0755)

			for _, a := range assets {
				resp, err := http.Get(a.URL)
				if err == nil {
					outPath := filepath.Join(assetsDir, a.Name)
					if a.Name == "design.png" {
						outPath = filepath.Join(targetDir, a.Name)
					}
					outFile, err := os.Create(outPath)
					if err == nil {
						io.Copy(outFile, resp.Body)
						outFile.Close()
					}
					resp.Body.Close()
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Saved successfully"))
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
