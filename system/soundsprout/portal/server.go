package portal

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Server struct {
	logger *slog.Logger
	tmpl   *template.Template
}

// NewServer parses the embedded templates and returns a Server.
func NewServer(logger *slog.Logger) (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{logger: logger, tmpl: tmpl}, nil
}

// Routes returns the portal's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	return mux
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil && s.logger != nil {
		s.logger.Error("render", "template", name, "error", err)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "index.html", map[string]any{"Title": "SoundSprout"})
}
