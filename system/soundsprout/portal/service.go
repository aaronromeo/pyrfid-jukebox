package portal

import "log/slog"

// NewService wires the portal's dependencies and returns a ready Server.
func NewService(logger *slog.Logger) (*Server, error) {
	return NewServer(logger)
}
