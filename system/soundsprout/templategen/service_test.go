//nolint:gocognit
package templategen_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aaronromeo.com/soundsprout/templategen"
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
		name            string
		envVars         map[string]string
		expectError     bool
		baselinesPath   string
		destinationPath string
	}{
		{
			name: "Happy Path - All Env Vars Present",
			envVars: map[string]string{
				"PJ_BLUETOOTH_DEVICE": "TestBluetoothDevice",
			},
			expectError:     false,
			baselinesPath:   "happypath",
			destinationPath: "/home/pi",
		},
		{
			name: "Match Some Services",
			envVars: map[string]string{
				"PJ_BLUETOOTH_DEVICE": "TestBluetoothDevice",
			},
			expectError:     false,
			baselinesPath:   "matchsomeservices",
			destinationPath: "./testdata/destination/matchsomeservices/pi",
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

			basePath, err := filepath.Abs("./")
			if err != nil {
				assert.Fail(t, "Did not expect error getting absolute path, but got: %v", err)
			}

			outputDir := t.TempDir()

			// Create the service with the mock logger
			service := templategen.NewTemplateGenService(slog.New(mockLogger), outputDir)
			for i, template := range service.Templates {
				service.Templates[i].DestinationFile = strings.Replace(template.DestinationFile, "/home/pi", tt.destinationPath, 1)
			}

			// Execute the Run method
			err = service.Run()

			if tt.expectError != (err != nil) {
				assert.Fail(t, "Expected error, but got none")
			}

			if tt.expectError {
				return
			}

			if err != nil {
				assert.Fail(t, "Did not expect error", err)
			}

			for _, templates := range service.Templates {
				actualFileName := filepath.Join(
					outputDir,
					templates.TemplateFile,
				)

				assert.FileExists(t, actualFileName)

				var expectedFile []byte
				expectedFile, err = os.ReadFile(
					filepath.Join(basePath, "testdata", "baselines", tt.baselinesPath, templates.TemplateFile),
				)
				if err != nil {
					assert.Fail(t, "Did not expect error getting expected file", err)
				}
				var actualFile []byte
				actualFile, err = os.ReadFile(actualFileName)
				if err != nil {
					assert.Fail(t, "Did not expect error getting expected file", err)
				}
				assert.Equal(t, string(expectedFile), string(actualFile))
			}
			actualFile, err := os.ReadFile(
				filepath.Join(
					outputDir,
					templategen.RunnerFilename,
				),
			)
			assert.NoError(t, err)
			actualFile = []byte(strings.ReplaceAll(string(actualFile), outputDir, "<TEMPDIR>"))

			expectedFile, err := os.ReadFile(
				filepath.Join(basePath, "testdata", "baselines", tt.baselinesPath, templategen.RunnerFilename),
			)
			assert.NoError(t, err)
			assert.Equal(t, string(expectedFile), string(actualFile))
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
			expected: "Name: John Doe, Age: 30, Country: USA\n",
		},
	}

	// Create a test template file
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := templategen.GenerateTemplate(
				filepath.Join("templates", tt.template),
				tt.data,
			)
			if err != nil {
				t.Fatalf("Failed to generate template: %v", err)
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}
