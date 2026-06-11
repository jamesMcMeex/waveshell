package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var varRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func interpolate(s string) (string, error) {
	var missing []string
	result := varRe.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1]
		val, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		return val
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("environment variables not set: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

func expandTilde(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func interpolateConfig(cfg *Config) error {
	var err error

	for i, p := range cfg.Library.Paths {
		cfg.Library.Paths[i], err = interpolate(p)
		if err != nil {
			return fmt.Errorf("library path %d: %w", i, err)
		}
	}

	cfg.Player.MPVSocket, err = interpolate(cfg.Player.MPVSocket)
	if err != nil {
		return fmt.Errorf("player.mpvsocket: %w", err)
	}

	cfg.Logging.Path, err = interpolate(cfg.Logging.Path)
	if err != nil {
		return fmt.Errorf("logging.path: %w", err)
	}

	cfg.Hooks.OnTrackChange, err = interpolate(cfg.Hooks.OnTrackChange)
	if err != nil {
		return fmt.Errorf("hooks.on_track_change: %w", err)
	}

	cfg.Hooks.OnPlaybackStart, err = interpolate(cfg.Hooks.OnPlaybackStart)
	if err != nil {
		return fmt.Errorf("hooks.on_playback_start: %w", err)
	}

	cfg.Hooks.OnPlaybackStop, err = interpolate(cfg.Hooks.OnPlaybackStop)
	if err != nil {
		return fmt.Errorf("hooks.on_playback_stop: %w", err)
	}

	cfg.Scrobbling.Username, err = interpolate(cfg.Scrobbling.Username)
	if err != nil {
		return fmt.Errorf("scrobbling.username: %w", err)
	}

	return nil
}

func applyTildeExpansion(cfg *Config) {
	for i, p := range cfg.Library.Paths {
		cfg.Library.Paths[i] = expandTilde(p)
	}
	cfg.Player.MPVSocket = expandTilde(cfg.Player.MPVSocket)
	cfg.Logging.Path = expandTilde(cfg.Logging.Path)
}
