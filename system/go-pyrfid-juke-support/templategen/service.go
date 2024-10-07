package templategen

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

type FileTemplate struct {
	Name            string
	TemplateFile    string
	DestinationFile string
	EnvVars         []string
}

type Service struct {
	// cmdExecutor CommandExecutor
	Templates []FileTemplate
	logger    *slog.Logger
	OutputDir string
}

func NewTemplateGenService(logger *slog.Logger, outputDir string) *Service {
	templates := []FileTemplate{
		{
			Name:            "config-cmus-autosave",
			TemplateFile:    "config-cmus-autosave.txt",
			DestinationFile: "/home/pi/.config/cmus/autosave",
			EnvVars:         []string{"PJ_BLUETOOTH_DEVICE"},
		},
		{
			Name:            "asoundrc",
			TemplateFile:    "asoundrc.txt",
			DestinationFile: "/home/pi/.asoundrc",
			EnvVars:         []string{"PJ_BLUETOOTH_DEVICE"},
		},
	}

	return &Service{
		Templates: templates,
		logger:    logger,
		OutputDir: outputDir,
	}
}

func (ft *Service) Run() error {
	if ft.logger == nil {
		return fmt.Errorf("logger has not been configured")
	}

	runner, err := os.Create(filepath.Join(ft.OutputDir, "runner.sh"))
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
		generatedTemplate, err = os.Create(filepath.Join(ft.OutputDir, t.TemplateFile))
		if err != nil {
			ft.logger.Error("Error creating output file", "error", err)
			return err
		}
		defer generatedTemplate.Close()

		outputs[t.DestinationFile] = output
		_, err = generatedTemplate.WriteString(output)
		if err != nil {
			ft.logger.Error("Error creating output file", "error", err)
			return err
		}

		_, destinationDirErr := os.Stat(filepath.Dir(t.DestinationFile))
		_, destinationFileErr := os.Stat(t.DestinationFile)

		if destinationFileErr == nil {
			var destinationBytes []byte
			destinationBytes, err = os.ReadFile(t.DestinationFile)
			if err != nil {
				ft.logger.Error("Error reading destination file", "error", err, "file", t.DestinationFile)
				return err
			}

			generatedBytes := make([]byte, 100)
			_, err = generatedTemplate.Read(generatedBytes)
			if err != nil {
				ft.logger.Error("Error reading generated file", "error", err, "file", generatedTemplate.Name())
				return err
			}

			if !bytes.Equal(destinationBytes, generatedBytes) {
				mvCmd := fmt.Sprintf("mv %s %s\n", t.TemplateFile, t.DestinationFile)
				_, err = runner.WriteString(mvCmd)
				if err != nil {
					ft.logger.Error("Error writing to runner file", "error", err, "cmd", mvCmd)
					return err
				}
			}
		}

		if os.IsNotExist(destinationFileErr) && os.IsNotExist(destinationDirErr) {
			destinationDir := filepath.Dir(t.DestinationFile)
			mkdirCmd := fmt.Sprintf("mkdir -p %s\n", destinationDir)
			_, err = runner.WriteString(mkdirCmd)
			if err != nil {
				ft.logger.Error("Error writing to runner file", "error", err, "cmd", mkdirCmd)
				return err
			}
		} else if !os.IsNotExist(destinationFileErr) {
			ft.logger.Error("Error stating file", "error", destinationFileErr)
			return destinationFileErr
		}
	}

	return nil
}
