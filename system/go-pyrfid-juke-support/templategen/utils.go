package templategen

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
)

func GenerateTemplate(inputFile string, data map[string]string) (string, error) {
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
