# Web Portal Part 1 — Framework / Hello World Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Add a `soundsprout-server serve` subcommand that launches a supervised HTTP portal serving a Hello World page — the framework the card manager (Part 2) and Bluetooth (Part 3) build on.

**Architecture:** A new `portal` package holds the HTTP server and an embedded `index.html`. `cmd/main.go` registers the `serve` command and (while here) fixes the dotenv path to the absolute `/home/pi/.soundsprout/conf`. A new Supervisor program runs the binary as `pi` on port 8080.

**Tech Stack:** Go 1.21 (stdlib `net/http`, `html/template`, `embed`), existing `urfave/cli`, `godotenv`, `testify`; Supervisor.

## Global Constraints

- Go module path: `aaronromeo.com/soundsprout`. Go floor: 1.21 (CI) — do not use 1.22 `ServeMux` method-pattern routing.
- No shell anywhere: always `exec.Command(name, args...)` / `exec.CommandContext`. Never pass user input to a shell.
- Go tests live in external `_test` packages (e.g. `package portal_test`) — the `testpackage` linter is enabled in `system/soundsprout/.golangci.yml`.
- Canonical paths (used as defaults; later parts inject temp paths in tests):
  - music dir `/home/pi/workspace/pyrfid-jukebox/music`
  - map file `/home/pi/workspace/pyrfid-jukebox/server/rfid_map.json`
  - settings dotenv `/home/pi/.soundsprout/conf`
- No authentication (LAN-only) — later parts require same-host `Origin`/`Referer` on POSTs.
- Run Go tests with `ASDF_GOLANG_VERSION=1.25.6` locally (repo pins 1.23.1 which is not installed).
- Python: black line-length 79, flake8 max-line-length 80.

---

### Task 1: Portal package — Hello World HTTP server

**Files:**
- Create: `system/soundsprout/portal/server.go`
- Create: `system/soundsprout/portal/service.go`
- Create: `system/soundsprout/portal/templates/index.html`
- Test: `system/soundsprout/portal/server_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Server struct{...}`
  - `func NewServer(logger *slog.Logger) (*Server, error)`
  - `func (s *Server) Routes() http.Handler`
  - `func NewService(logger *slog.Logger) (*Server, error)`

- [x] **Step 1: Write the failing test**

Create `system/soundsprout/portal/server_test.go`:

```go
package portal_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aaronromeo.com/soundsprout/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexRendersHello(t *testing.T) {
	srv, err := portal.NewServer(nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hello, world")
}

func TestUnknownPathIsNotFound(t *testing.T) {
	srv, err := portal.NewServer(nil)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: FAIL — `package portal` has no Go files / `portal.NewServer` undefined.

- [x] **Step 3: Write `service.go`**

Create `system/soundsprout/portal/service.go`:

```go
package portal

import "log/slog"

// NewService wires the portal's dependencies and returns a ready Server.
func NewService(logger *slog.Logger) (*Server, error) {
	return NewServer(logger)
}
```

- [x] **Step 4: Write `server.go`**

Create `system/soundsprout/portal/server.go`:

```go
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
```

- [x] **Step 5: Write `templates/index.html`**

Create `system/soundsprout/portal/templates/index.html`:

```html
<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>{{.Title}}</title></head>
<body>
<h1>{{.Title}}</h1>
<p>Hello, world.</p>
</body>
</html>
```

- [x] **Step 6: Run tests to verify they pass**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go test ./portal/...`
Expected: PASS (2 tests).

- [x] **Step 7: Commit**

```bash
git add system/soundsprout/portal
git commit -m "feat(portal): add hello-world HTTP server"
```

---

### Task 2: `serve` subcommand and absolute dotenv path

**Files:**
- Modify: `system/soundsprout/cmd/main.go`

**Interfaces:**
- Consumes: `portal.NewService`, `(*portal.Server).Routes`.
- Produces: CLI command `serve` with an `--addr` flag (default `0.0.0.0:8080`).

- [x] **Step 1: Add `serve` and fix the dotenv path**

Replace the entire contents of `system/soundsprout/cmd/main.go` with:

```go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"aaronromeo.com/soundsprout/btconnect"
	"aaronromeo.com/soundsprout/portal"
	"aaronromeo.com/soundsprout/templategen"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := godotenv.Load("/home/pi/.soundsprout/conf")
	if err != nil {
		logger.Error("Error loading .env file")
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "btconnect",
				Aliases: []string{"b"},
				Usage:   "Maintain a connection to bluetooth device",
				Action: func(c *cli.Context) error {
					connectService := btconnect.NewBtConnectService(
						&btconnect.OSCommandExecutor{},
						logger,
					)
					err = connectService.Run()
					if err != nil {
						logger.Error(
							"btconnect failure",
							"error", err,
						)
					}
					return nil
				},
			},
			{
				Name:    "templategen",
				Aliases: []string{"t"},
				Usage:   "Generate the templates needed for this service",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output-dir",
						Aliases:  []string{"o"},
						Value:    "./outputs",
						Required: true,
					},
				},
				Action: func(ctx *cli.Context) error {
					var outputPath string
					outputPath, err = filepath.Abs(ctx.String("output-dir"))
					if err != nil {
						return err
					}

					_, err = os.Stat(outputPath)
					if err != nil && os.IsNotExist(err) {
						return fmt.Errorf("output path '%s' must exist", outputPath)
					}

					templateService := templategen.NewTemplateGenService(
						logger,
						outputPath,
					)

					err = templateService.Run()
					if err != nil {
						logger.Error(
							"templategen failure",
							"error", err,
						)
					}
					return nil
				},
			},
			{
				Name:  "serve",
				Usage: "Run the web portal",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Value: "0.0.0.0:8080",
					},
				},
				Action: func(ctx *cli.Context) error {
					portalServer, srvErr := portal.NewService(logger)
					if srvErr != nil {
						return srvErr
					}
					logger.Info("portal listening", "addr", ctx.String("addr"))
					httpServer := &http.Server{
						Addr:              ctx.String("addr"),
						Handler:           portalServer.Routes(),
						ReadHeaderTimeout: 5 * time.Second,
					}
					return httpServer.ListenAndServe()
				},
			},
		},
	}

	if err = app.Run(os.Args); err != nil {
		logger.Error(
			"failure on run",
			"args", os.Args,
			"error", err,
		)
	}
}
```

- [x] **Step 2: Build and vet**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go build ./... && ASDF_GOLANG_VERSION=1.25.6 go vet ./...`
Expected: no output, exit 0.

- [x] **Step 3: Confirm the command is registered**

Run: `cd system/soundsprout && ASDF_GOLANG_VERSION=1.25.6 go run ./cmd serve --help`
Expected: prints help for `serve` including the `--addr` flag.

- [x] **Step 4: Commit**

```bash
git add system/soundsprout/cmd/main.go
git commit -m "feat(portal): add serve subcommand and fix dotenv path"
```

---

### Task 3: Supervise the portal, create logs, document it

**Files:**
- Create: `system/supervisor/conf.d/portal.conf`
- Modify: `system/scripts/prepare_logs.sh`
- Modify: `README.md`
- Modify: `AGENTS.md`

**Interfaces:**
- Consumes: the `serve` command from Task 2.
- Produces: a supervised `portal` program on port 8080 with logs under `/home/pi/logs/`.

- [x] **Step 1: Create the supervisor program**

Create `system/supervisor/conf.d/portal.conf`:

```ini
[program:portal]
autorestart=true
autostart=true
startsecs=5
command=/usr/local/bin/soundsprout-server serve --addr 0.0.0.0:8080
stderr_logfile_backups=2
stderr_logfile_maxbytes=200KB
stderr_logfile=/home/pi/logs/portal.err.log
stdout_logfile_backups=2
stdout_logfile_maxbytes=200KB
stdout_logfile=/home/pi/logs/portal.out.log
user=pi
```

Note: `setup.sh` already runs `cp system/supervisor/conf.d/* /etc/supervisor/conf.d/`, so
no `setup.sh` change is needed for the program itself.

- [x] **Step 2: Create the portal logs in `prepare_logs.sh`**

In `system/scripts/prepare_logs.sh`, insert this block immediately before the final
`chmod 666 /home/pi/logs/*` line:

```bash
touch /home/pi/logs/portal.err.log
touch /home/pi/logs/portal.out.log
chown pi:pi /home/pi/logs/portal.err.log
chown pi:pi /home/pi/logs/portal.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created portal logs." >> /home/pi/logs/portal.out.log
echo "

$(date '+%Y-%m-%d %H:%M:%S') - Created portal err." >> /home/pi/logs/portal.err.log
```

- [x] **Step 3: Verify shell syntax**

Run: `bash -n setup.sh system/scripts/prepare_logs.sh`
Expected: exit 0.

- [x] **Step 4: Document the portal in `README.md`**

Add this section at the end of `README.md`:

````markdown
## Web Portal

`soundsprout-server serve` runs a LAN-only web portal (no authentication). It is
supervised as `portal` (`system/supervisor/conf.d/portal.conf`) on port 8080 and logs to
`/home/pi/logs/portal.{out,err}.log`.

Run it locally with:

```bash
soundsprout-server serve --addr 127.0.0.1:8080
```
````

- [x] **Step 5: Document the portal in `AGENTS.md`**

In the `## Code structure` list of `AGENTS.md`, add:

```markdown
- `system/soundsprout/portal/`: Go web portal package (embedded HTML templates). Started
  with `soundsprout-server serve`.
```

- [x] **Step 6: Commit**

```bash
git add system/supervisor/conf.d/portal.conf system/scripts/prepare_logs.sh README.md AGENTS.md
git commit -m "feat(system): supervise the portal and document it"
```

---

### Task 4: Final validation for Part 1

- [x] **Step 1: Run the full Go suite**

Run: `ASDF_GOLANG_VERSION=1.25.6 make test`
Expected: all packages PASS, including `portal`.

- [x] **Step 2: Run golangci-lint (if available)**

Run: `cd system/soundsprout && golangci-lint run --config .golangci.yml ./...`
Expected: no issues. (If `golangci-lint` is not installed, skip and let CI run it; fix any
findings the CI job reports before merging.)

- [x] **Step 3: Smoke-test the running binary**

Run:

```bash
cd system/soundsprout
ASDF_GOLANG_VERSION=1.25.6 go build -o /tmp/soundsprout-server ./cmd
/tmp/soundsprout-server serve --addr 127.0.0.1:8080 &
sleep 1
curl -sS http://127.0.0.1:8080/ | grep -q "Hello, world" && echo "OK"
kill %1
```

Expected: prints `OK`.

- [x] **Step 4: Run pre-commit**

Run: `pre-commit run --all-files`
Expected: hooks clean (fix formatting if needed).

- [x] **Step 5: Commit any final fixups**

```bash
git add -A
git commit -m "chore: part 1 portal framework validation fixes"
```
