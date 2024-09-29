package templategen

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
)

type FileTemplate struct {
	Name         string
	TemplateFile string
	OutputFile   string
	EnvVars      []string
}

type Service struct {
	// cmdExecutor CommandExecutor
	templates []FileTemplate
	logger    *slog.Logger
}

func NewTemplateGenService(logger *slog.Logger) *Service {
	templates := []FileTemplate{
		{
			Name:         "config-cmus-autosave",
			TemplateFile: "templates/config-cmus-autosave.txt",
			OutputFile:   "/home/pi/.config/cmus/autosave",
			EnvVars:      []string{"PJ_BLUETOOTH_DEVICE"},
		},
		{
			Name:         "asoundrc",
			TemplateFile: "templates/asoundrc.txt",
			OutputFile:   "/home/pi/.asoundrc",
			EnvVars:      []string{"PJ_BLUETOOTH_DEVICE"},
		},
	}

	return &Service{
		// cmdExecutor: cmdExecutor,
		templates: templates,
		logger:    logger,
	}
}

func (ft *Service) Run() error {
	if ft.logger == nil {
		return fmt.Errorf("logger has not been configured")
	}

	outputs := map[string]string{}
	for _, t := range ft.templates {
		ft.logger.Info("Generating template", "Name", t.Name)

		substitutions := map[string]string{}
		for _, e := range t.EnvVars {
			if v, found := os.LookupEnv(e); found {
				substitutions[e] = v
			} else {
				ft.logger.Error("Missing env var", "env var", e, "Name", t.Name)
				return fmt.Errorf("env var %s not found for substitution", e)
			}
		}
		output, err := generateTemplate(t.TemplateFile, substitutions)
		if err != nil {
			return err
		}

		outputs[t.OutputFile] = output
	}

	return nil
}

func generateTemplate(inputFile string, data map[string]string) (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}
	// Parse the template file
	tmpl, err := template.ParseFiles(inputFile)
	if err != nil {
		// the executable directory
		return "", fmt.Errorf("%w - current path %s", err, dir)
	}

	// Use a buffer to capture the output instead of writing to a file
	var out bytes.Buffer
	err = tmpl.Execute(&out, data)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
