package portal

import (
	"log/slog"

	"aaronromeo.com/soundsprout/bluetooth"
)

const (
	DefaultMusicDir     = "/home/pi/mp3s"
	DefaultMapFile      = "/home/pi/workspace/pyrfid-jukebox/server/rfid_map.json"
	DefaultMapLock      = "/tmp/pyrfid_jukebox.map.lock"
	DefaultSettingsFile = "/home/pi/.soundsprout/conf"
	DefaultSettingsLock = "/tmp/pyrfid_jukebox.settings.lock"
	DefaultAsoundrc     = "/home/pi/.asoundrc"
	DefaultCmusRc       = "/home/pi/.config/cmus/rc"
)

// Config holds every injectable path the portal uses.
type Config struct {
	MusicDir     string
	MapFile      string
	MapLock      string
	SettingsFile string
	SettingsLock string
	AsoundrcPath string
	CmusRcPath   string
}

// DefaultConfig returns the canonical on-device paths.
func DefaultConfig() Config {
	return Config{
		MusicDir:     DefaultMusicDir,
		MapFile:      DefaultMapFile,
		MapLock:      DefaultMapLock,
		SettingsFile: DefaultSettingsFile,
		SettingsLock: DefaultSettingsLock,
		AsoundrcPath: DefaultAsoundrc,
		CmusRcPath:   DefaultCmusRc,
	}
}

// NewService wires the portal's dependencies and returns a ready Server.
func NewService(logger *slog.Logger) (*Server, error) {
	cfg := DefaultConfig()
	runner := bluetooth.OSRunner{}
	mapStore := NewMapStore(cfg.MapFile, cfg.MapLock)
	library := NewLibrary(cfg.MusicDir)
	settings := NewSettings(cfg.SettingsFile, cfg.SettingsLock)
	audio := NewAudioConfig(cfg.AsoundrcPath, cfg.CmusRcPath, runner)
	bt := NewBluetooth(runner, settings, audio)
	return NewServer(mapStore, library, bt, logger)
}
