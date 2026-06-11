package handlers

import (
	"html/template"
	"net/http"

	"ui-prompter/internal/db"
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

	// Serve static files
	mux.Handle("GET /static/", http.FileServer(http.FS(ui.Files)))
}

func (a *App) homeHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.homeTmpl.Execute(w, nil); err != nil {
		http.Error(w, "Failed to execute template", http.StatusInternalServerError)
	}
}

func (a *App) figmaHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Figma URL endpoint - Coming Soon"))
}

func (a *App) imageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Image upload endpoint - Coming Soon"))
}
