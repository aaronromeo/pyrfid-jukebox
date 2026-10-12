package portal

import (
	"os"
	"strings"
	"syscall"
)

type Settings struct {
	path     string
	lockPath string
}

// NewSettings returns a dotenv store at path, locked via lockPath for writes.
func NewSettings(path, lockPath string) *Settings {
	return &Settings{path: path, lockPath: lockPath}
}

// Get returns the value for key, or "" if absent.
func (s *Settings) Get(key string) (string, error) {
	values, err := s.read()
	if err != nil {
		return "", err
	}
	return values[key], nil
}

// Set updates one dotenv key in place, preserving other keys and comments.
func (s *Settings) Set(key, value string) error {
	lock, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()

	lines, err := s.readLines()
	if err != nil {
		return err
	}
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			lines[i] = key + "=" + value
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, key+"="+value)
	}
	content := strings.Join(lines, "\n") + "\n"
	return writeFileAtomic(s.path, content)
}

func (s *Settings) read() (map[string]string, error) {
	values := map[string]string{}
	lines, err := s.readLines()
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if found {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return values, nil
}

func (s *Settings) readLines() ([]string, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	content := strings.TrimRight(string(raw), "\n")
	if content == "" {
		return nil, nil
	}
	return strings.Split(content, "\n"), nil
}
