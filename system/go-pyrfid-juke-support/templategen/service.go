package templategen

import (
	"fmt"
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
	Templates []FileTemplate
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
		Templates: templates,
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
		log.Printf("Error creating runner file: %v", err)
		return err
	}
	defer runner.Close()

	_, err = runner.WriteString("#!/bin/bash\n\n")
	if err != nil {
		log.Printf("Error writing to runner file: %v", err)
		return err
	}

	outputs := map[string]string{}
	for _, t := range ft.Templates {
		var output string

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
		output, err = GenerateTemplate(absTemplateFilename, substitutions)
		if err != nil {
			return err
		}

		var generatedTemplate *os.File
		generatedTemplate, err = os.Create(filepath.Join(outputPath, t.TemplateFile))
		if err != nil {
			ft.logger.Error("Error creating output file", "error", err)
			return err
		}
		defer generatedTemplate.Close()

		outputs[t.OutputFile] = output
		_, err = generatedTemplate.WriteString(output)
		if err != nil {
			ft.logger.Error("Error creating output file", "error", err)
			return err
		}

		mvCmd := fmt.Sprintf("mv %s %s\n", t.TemplateFile, t.OutputFile)
		_, err = runner.WriteString(mvCmd)
		if err != nil {
			ft.logger.Error("Error writing to runner file", "error", err)
			return err
		}
	}

	return nil
}
