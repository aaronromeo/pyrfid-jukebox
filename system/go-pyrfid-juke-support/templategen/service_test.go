//nolint:gocognit
package templategen_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"aaronromeo.com/go-pyrfid-juke-support/templategen"
	"github.com/stretchr/testify/assert"
)

// MockLogger provides a simple logger for testing purposes.
type MockLogger struct {
	logs []string
}

// Enabled implements slog.Handler.
func (ml *MockLogger) Enabled(context.Context, slog.Level) bool {
	return true
}

// Handle implements slog.Handler.
func (ml *MockLogger) Handle(context.Context, slog.Record) error {
	// panic("unimplemented")
	return nil
}

// WithAttrs implements slog.Handler.
func (*MockLogger) WithAttrs(_ []slog.Attr) slog.Handler {
	panic("unimplemented")
}

// WithGroup implements slog.Handler.
func (*MockLogger) WithGroup(_ string) slog.Handler {
	panic("unimplemented")
}

func (ml *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	ml.logs = append(ml.logs, msg)
	for _, val := range keysAndValues {
		logVal, ok := val.(string)
		if ok {
			ml.logs = append(ml.logs, logVal)
		}
	}
}

func (ml *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	ml.logs = append(ml.logs, msg)
	for _, val := range keysAndValues {
		logVal, ok := val.(string)
		if ok {
			ml.logs = append(ml.logs, logVal)
		}
	}
}

func TestRun(t *testing.T) {
	// Setup mock logger
	mockLogger := &MockLogger{}

	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "Happy Path - All Env Vars Present",
			envVars: map[string]string{
				"PJ_BLUETOOTH_DEVICE": "TestBluetoothDevice",
			},
			expectError: false,
		},
		{
			name:        "Error Path - Missing Env Var",
			envVars:     map[string]string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for the test case
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Ensure to clean up environment variables after each test
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			// Create the service with the mock logger
			service := templategen.NewTemplateGenService(slog.New(mockLogger))

			// Execute the Run method
			err := service.Run()

			if tt.expectError {
				if err == nil {
					assert.Fail(t, "Expected error, but got none")
				}
			} else {
				if err != nil {
					assert.Fail(t, "Did not expect error, but got: %v", err)
				}

				var basePath string
				basePath, err = filepath.Abs("./")
				if err != nil {
					assert.Fail(t, "Did not expect error getting absolute path, but got: %v", err)
				}
				for _, templates := range service.Templates {
					assert.FileExists(t,
						filepath.Join(
							basePath,
							"..", "..", "..", "outputs",
							templates.TemplateFile,
						),
					)

					var expectedFile []byte
					expectedFile, err = os.ReadFile(filepath.Join(basePath, "baselines", templates.TemplateFile))
					if err != nil {
						assert.Fail(t, "Did not expect error getting expected file, but got: %v", err)
					}
					var actualFile []byte
					actualFile, err = os.ReadFile(filepath.Join(
						basePath,
						"..", "..", "..", "outputs",
						templates.TemplateFile,
					))
					if err != nil {
						assert.Fail(t, "Did not expect error getting expected file, but got: %v", err)
					}
					assert.Equal(t, string(expectedFile), string(actualFile))
				}
			}
		})
	}
}

func TestGenerateTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]string
		expected string
	}{
		{
			name:     "Happy Path Test",
			template: "test_template.txt",
			data: map[string]string{
				"Name":    "John Doe",
				"Age":     "30",
				"Country": "USA",
			},
			expected: "Name: John Doe, Age: 30, Country: USA",
		},
	}

	// Create a test template file
	templateContent := "Name: {{.Name}}, Age: {{.Age}}, Country: {{.Country}}"
	if err := os.WriteFile("test_template.txt", []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}
	defer os.Remove("test_template.txt") // Cleanup the test file after execution

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := templategen.GenerateTemplate(tt.template, tt.data)
			if err != nil {
				t.Fatalf("Failed to generate template: %v", err)
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}
