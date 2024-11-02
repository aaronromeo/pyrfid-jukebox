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

	if _, err := runner.WriteString("#!/bin/bash\n\n"); err != nil {
		log.Printf("Error writing to runner file: %v", err)
		return err
	}

	outputs := map[string]string{}
	for _, template := range ft.Templates {
		if err := ft.processTemplateSubs(template, outputs, runner); err != nil {
			return err
		}
	}

	return nil
}

func (ft *Service) processTemplateSubs(template FileTemplate, outputs map[string]string, runner *os.File) error {
	ft.logger.Info("Generating template", "Name", template.Name)

	generateTemplateFilename, err := ft.createNewTemplatedFile(template, outputs)
	if err != nil {
		return err
	}

	fileExists, dirExists, err := ft.checkDestination(template)
	if err != nil {
		return err
	}

	cmds := []string{}

	if !dirExists {
		destinationDir := filepath.Dir(template.DestinationFile)
		cmds = append(cmds, fmt.Sprintf("mkdir -p %s\n", destinationDir))
	}

	contentEqual := false
	if fileExists {
		contentEqual, err = ft.contentEqual(template.DestinationFile, generateTemplateFilename)
		if err != nil {
			return err
		}
	}

	if !contentEqual {
		cmds = append(cmds, []string{
			fmt.Sprintf("mv %s %s\n", template.TemplateFile, template.DestinationFile),
			fmt.Sprintf("chown pi %s\n", template.DestinationFile),
			fmt.Sprintf("sudo supervisorctl %s reload\n", template.ServiceRestart),
			fmt.Sprintf("sudo supervisorctl %s restart\n", template.ServiceRestart),
		}...)
	}

	for _, cmd := range cmds {
		if _, err = runner.WriteString(cmd); err != nil {
			ft.logger.Error("Error writing to runner file", "error", err, "cmd", cmd)
			return err
		}
	}
	return nil
}

func (ft *Service) checkDestination(template FileTemplate) (bool, bool, error) {
	_, destinationDirErr := os.Stat(filepath.Dir(template.DestinationFile))
	_, destinationFileErr := os.Stat(template.DestinationFile)

	if (destinationFileErr != nil && !os.IsNotExist(destinationFileErr)) ||
		(destinationDirErr != nil && !os.IsNotExist(destinationDirErr)) {
		ft.logger.Error("Error stating file", "error", destinationFileErr)
		return false, false, destinationFileErr
	}

	fileExists := destinationFileErr == nil
	dirExists := destinationDirErr == nil

	return fileExists, dirExists, nil
}

func (ft *Service) createNewTemplatedFile(template FileTemplate, outputs map[string]string) (string, error) {
	substitutions, err := ft.getSubstitutions(template)
	if err != nil {
		return "", err
	}
	absTemplateFilename := filepath.Join("templates", template.TemplateFile)
	output, err := GenerateTemplate(absTemplateFilename, substitutions)
	if err != nil {
		return "", err
	}

	generatedTemplate, err := os.Create(filepath.Join(ft.OutputDir, template.TemplateFile))
	if err != nil {
		ft.logger.Error("Error creating output file", "error", err)
		return "", err
	}
	defer generatedTemplate.Close()

	outputs[template.DestinationFile] = output
	if _, err := generatedTemplate.WriteString(output); err != nil {
		ft.logger.Error("Error creating output file", "error", err)
		return "", err
	}
	return generatedTemplate.Name(), nil
}

func (ft *Service) contentEqual(destinationFile string, generatedFile string) (bool, error) {
	destinationBytes, err := os.ReadFile(destinationFile)
	if err != nil {
		ft.logger.Error("Error reading destination file", "error", err, "file", destinationFile)
		return false, err
	}

	generatedBytes, err := os.ReadFile(generatedFile)
	if err != nil {
		ft.logger.Error("Error reading generated file", "error", err, "file", generatedFile)
		return false, err
	}

	return bytes.Equal(destinationBytes, generatedBytes), nil
}

func (ft *Service) getSubstitutions(template FileTemplate) (map[string]string, error) {
	substitutions := map[string]string{}
	for _, e := range template.EnvVars {
		if v, found := os.LookupEnv(e); found {
			substitutions[e] = v
		} else {
			ft.logger.Error("Missing env var", "env var", e, "Name", template.Name)
			return nil, fmt.Errorf("env var %s not found for substitution", e)
		}
	}
	return substitutions, nil
}
