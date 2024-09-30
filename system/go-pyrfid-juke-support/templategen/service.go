package templategen

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
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
			TemplateFile: "config-cmus-autosave.txt",
			OutputFile:   "/home/pi/.config/cmus/autosave",
			EnvVars:      []string{"PJ_BLUETOOTH_DEVICE"},
		},
		{
			Name:         "asoundrc",
			TemplateFile: "asoundrc.txt",
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

	outputPath, err := filepath.Abs("./../../../outputs")
	if err != nil {
		return err
	}

	runner, err := os.Create(filepath.Join(outputPath, "runner.sh"))
	if err != nil {
		log.Fatalf("Error creating runner file: %v", err)
		return err
	}
	defer runner.Close()

	_, err = runner.WriteString("#!/bin/bash\n\n")
	if err != nil {
		log.Fatalf("Error writing to runner file: %v", err)
		return err
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
		absTemplateFilename := filepath.Join("templates", t.TemplateFile)
		output, err := generateTemplate(absTemplateFilename, substitutions)
		if err != nil {
			return err
		}

		generatedTemplate, err := os.Create(filepath.Join(outputPath, t.TemplateFile))
		if err != nil {
			log.Fatalf("Error creating output file: %v", err)
			return err
		}
		defer generatedTemplate.Close()

		outputs[t.OutputFile] = output
		_, err = generatedTemplate.WriteString(output)
		if err != nil {
			log.Fatalf("Error creating output file: %v", err)
			return err
		}

		mvCmd := fmt.Sprintf("mv %s %s\n", t.TemplateFile, t.OutputFile)
		_, err = runner.WriteString(mvCmd)
		if err != nil {
			log.Fatalf("Error writing to runner file: %v", err)
			return err
		}
	}

	runner.Sync()
	return nil
}

func generateTemplate(inputFile string, data map[string]string) (string, error) {
	dir, err := filepath.Abs("./")
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
