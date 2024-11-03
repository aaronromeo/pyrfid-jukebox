package templategen

import (
	"bytes"
	"fmt"
	"html/template"
)

func GenerateTemplate(inputFile string, data map[string]string) (string, error) {
	// Parse the template file
	tmpl, err := template.ParseFS(templatesFS, inputFile)
	if err != nil {
		// the executable directory
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	// Use a buffer to capture the output instead of writing to a file
	var out bytes.Buffer
	err = tmpl.Execute(&out, data)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
