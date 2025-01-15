package mist

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
)

const supportedVersion = "2.1.1"

type Mist struct {
	CacheDownloads bool
	Timeout        time.Duration
}

type Installer struct {
	ID      string `json:"identifier"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Build   string `json:"build"`
	Size    int64  `json:"size"`
}

func New(cache bool, timeout time.Duration) (*Mist, error) {
	m := &Mist{
		CacheDownloads: cache,
		Timeout:        timeout,
	}
	ver, err := m.getVersion()
	if err != nil {
		return nil, err
	}
	if ver != supportedVersion {
		return nil, fmt.Errorf("mist version %s is not supported, please install mist %s", ver, supportedVersion)
	}
	return m, nil
}

func (m *Mist) CreateBootableInstaller(version, volume string) error {
	args := []string{"download", "installer", version, "bootableinstaller", "--bootable-installer-volume", volume}
	if m.CacheDownloads {
		args = append(args, "--cache-downloads")
	}
	_, err := m.runWithTimeout(m.Timeout, args...)
	return err
}

func (m *Mist) ListInstallers(onlyLatestPerMajor bool, majorLimit int) ([]Installer, error) {
	res, err := m.runSync("list", "installer", "-o", "json", "-q")
	if err != nil {
		return nil, err
	}
	var installers []Installer
	if err := json.Unmarshal([]byte(res), &installers); err != nil {
		return nil, err
	}

	if !onlyLatestPerMajor && majorLimit <= 0 {
		return installers, nil
	}

	majors := make(map[string]bool)
	var filtered []Installer
	for _, i := range installers {
		if onlyLatestPerMajor && majors[i.Name] {
			continue
		}
		majors[i.Name] = true
		if majorLimit > 0 && len(majors) > majorLimit {
			break
		}
		filtered = append(filtered, i)
	}

	return filtered, nil
}

func (m *Mist) getVersion() (string, error) {
	res, err := m.runSync("--version")
	if err != nil {
		return "", err
	}
	ver, _, found := strings.Cut(res, " (")
	if !found || ver == "" {
		return "", errors.New("unexpected version format")
	}
	return ver, nil
}

func (m *Mist) runWithTimeout(timeout time.Duration, args ...string) (string, error) {
	args = append(args, "--no-ansi")
	c := cmd.NewCmd("mist", args...)
	statusChan := c.Start()
	select {
	case s := <-statusChan:
		if s.Error != nil {
			return "", s.Error
		}
		if s.Exit != 0 || len(s.Stderr) != 0 {
			errText := strings.Join(s.Stderr, "\n")
			if errText == "" {
				errText = strings.Join(s.Stdout, "\n")
			}
			if errText == "" {
				return "", fmt.Errorf("mist exited with code %d", s.Exit)
			}
			return "", errors.New(errText)
		}
		return strings.Join(s.Stdout, "\n"), nil
	case <-time.After(timeout):
		c.Stop()
		return "", errors.New("timeout")
	}
}

func (m *Mist) runSync(args ...string) (string, error) {
	args = append(args, "--no-ansi")
	c := cmd.NewCmd("mist", args...)
	s := <-c.Start()
	if s.Error == nil && len(s.Stderr) != 0 {
		return "", errors.New(strings.Join(s.Stderr, "\n"))
	}
	return strings.Join(s.Stdout, "\n"), s.Error
}
