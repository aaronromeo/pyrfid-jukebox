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
	ServiceRestart  string
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
			ServiceRestart:  "cmus_manager",
		},
		{
			Name:            "asoundrc",
			TemplateFile:    "asoundrc.txt",
			DestinationFile: "/home/pi/.asoundrc",
			EnvVars:         []string{"PJ_BLUETOOTH_DEVICE"},
			ServiceRestart:  "btconnect",
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
	for _, template := range ft.Templates {
		var output string

		ft.logger.Info("Generating template", "Name", template.Name)

		substitutions := map[string]string{}
		for _, e := range template.EnvVars {
			if v, found := os.LookupEnv(e); found {
				substitutions[e] = v
			} else {
				ft.logger.Error("Missing env var", "env var", e, "Name", template.Name)
				return fmt.Errorf("env var %s not found for substitution", e)
			}
		}
		absTemplateFilename := filepath.Join("templates", template.TemplateFile)
		output, err = GenerateTemplate(absTemplateFilename, substitutions)
		if err != nil {
			return err
		}

		var generatedTemplate *os.File
		generatedTemplate, err = os.Create(filepath.Join(ft.OutputDir, template.TemplateFile))
		if err != nil {
			ft.logger.Error("Error creating output file", "error", err)
			return err
		}
		defer generatedTemplate.Close()

		outputs[template.DestinationFile] = output
		_, err = generatedTemplate.WriteString(output)
		if err != nil {
			ft.logger.Error("Error creating output file", "error", err)
			return err
		}

		_, destinationDirErr := os.Stat(filepath.Dir(template.DestinationFile))
		_, destinationFileErr := os.Stat(template.DestinationFile)

		if destinationFileErr == nil {
			var destinationBytes []byte
			destinationBytes, err = os.ReadFile(template.DestinationFile)
			if err != nil {
				ft.logger.Error("Error reading destination file", "error", err, "file", template.DestinationFile)
				return err
			}

			generatedBytes := make([]byte, 100)
			_, err = generatedTemplate.Read(generatedBytes)
			if err != nil {
				ft.logger.Error("Error reading generated file", "error", err, "file", generatedTemplate.Name())
				return err
			}

			if !bytes.Equal(destinationBytes, generatedBytes) {
				cmds := []string{
					fmt.Sprintf("mv %s %s\n", template.TemplateFile, template.DestinationFile),
					fmt.Sprintf("chown pi %s\n", template.DestinationFile),
					fmt.Sprintf("sudo supervisorctl %s reload\n", template.ServiceRestart),
					fmt.Sprintf("sudo supervisorctl %s restart\n", template.ServiceRestart),
				}
				for _, cmd := range cmds {
					_, err = runner.WriteString(cmd)
					if err != nil {
						ft.logger.Error("Error writing to runner file", "error", err, "cmd", cmd)
						return err
					}
				}
			}
		}

		if os.IsNotExist(destinationFileErr) && os.IsNotExist(destinationDirErr) {
			destinationDir := filepath.Dir(template.DestinationFile)
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
