package portal

import (
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
)

//go:embed templates/*.html
var templatesFS embed.FS

type cardRow struct {
	ID       string
	Path     string
	Exists   bool
	MP3Count int
}

type Server struct {
	mapStore *MapStore
	library  *Library
	logger   *slog.Logger
	tmpl     *template.Template
}

// NewServer parses the embedded templates and returns a Server.
func NewServer(mapStore *MapStore, library *Library, logger *slog.Logger) (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{mapStore: mapStore, library: library, logger: logger, tmpl: tmpl}, nil
}

// Routes returns the portal's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/cards", s.handleCards)
	mux.HandleFunc("/cards/assign", s.handleAssign)
	mux.HandleFunc("/cards/remove", s.handleRemove)
	return s.csrfGuard(mux)
}

// csrfGuard rejects POSTs whose Origin/Referer host differs from the request host.
func (s *Server) csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = r.Header.Get("Referer")
			}
			if origin != "" {
				if u, err := url.Parse(origin); err != nil || u.Host != r.Host {
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil && s.logger != nil {
		s.logger.Error("render", "template", name, "error", err)
	}
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	target := r.Referer()
	if target == "" {
		target = "/cards"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "index.html", map[string]any{"Title": "SoundSprout"})
}

func (s *Server) handleCards(w http.ResponseWriter, r *http.Request) {
	data, err := s.mapStore.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	folders, err := s.library.Folders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := make([]cardRow, 0, len(data))
	unassigned := []string{}
	for id, path := range data {
		if path == "" {
			unassigned = append(unassigned, id)
			continue
		}
		info, statErr := os.Stat(path)
		exists := statErr == nil && info.IsDir()
		count := 0
		if exists {
			count, _ = countMP3(path)
		}
		rows = append(rows, cardRow{ID: id, Path: path, Exists: exists, MP3Count: count})
	}
	s.render(w, "cards.html", map[string]any{
		"Rows":       rows,
		"Unassigned": unassigned,
		"Folders":    folders,
	})
}

func (s *Server) handleAssign(w http.ResponseWriter, r *http.Request) {
	cardID := r.FormValue("card_id")
	folder := r.FormValue("folder")

	data, err := s.mapStore.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, ok := data[cardID]; !ok {
		http.Error(w, "unknown card id", http.StatusBadRequest)
		return
	}
	folderPath, err := s.library.Resolve(folder)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.mapStore.Assign(cardID, folderPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.redirect(w, r)
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	if err := s.mapStore.Remove(r.FormValue("card_id")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.redirect(w, r)
}
