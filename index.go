package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/anna-oake/macos-please/mist"
)

type Installer struct {
	ID       string `json:"identifier"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Build    string `json:"build"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MD5      string `json:"md5"`
}

func getIndexPath(outputDir string) (string, error) {
	path := filepath.Join(outputDir, "index.json")
	exists, err := pathExists(path)
	if err != nil {
		return "", err
	}
	if !exists {
		err = os.WriteFile(path, []byte("[]"), 0644)
		if err != nil {
			return "", err
		}
	}
	return path, nil
}

func loadIndex(path string) (map[string]Installer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ins []Installer
	if err := json.Unmarshal(b, &ins); err != nil {
		return nil, err
	}
	m := make(map[string]Installer)
	for _, i := range ins {
		m[i.ID] = i
	}
	return m, nil
}

func saveIndex(path string, index map[string]Installer) error {
	var ins []Installer
	if len(index) == 0 {
		return os.WriteFile(path, []byte("[]"), 0644)
	}
	for _, i := range index {
		ins = append(ins, i)
	}
	b, err := json.MarshalIndent(ins, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func generateMetadata(im mist.Installer, imagePath string) (*Installer, error) {
	m := Installer{
		ID:       im.ID,
		Name:     im.Name,
		Version:  im.Version,
		Build:    im.Build,
		Filename: filepath.Base(imagePath),
	}
	fi, err := os.Stat(imagePath)
	if err != nil {
		return nil, err
	}
	m.Size = fi.Size()
	md5sum, err := fileMD5(imagePath)
	if err != nil {
		return nil, err
	}
	m.MD5 = md5sum
	return &m, nil
}

func fileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)[:16]), nil
}

func verifyInstaller(m Installer, dir string) (bool, error) {
	path := filepath.Join(dir, m.Filename)
	// check existence
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Println("-- Image doesn't exist")
		return false, nil
	}
	// check size
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if fi.Size() != m.Size {
		log.Println("-- Image size doesn't match")
		return false, nil
	}
	// check md5
	md5sum, err := fileMD5(path)
	if err != nil {
		return false, err
	}
	if md5sum != m.MD5 {
		log.Println("-- Image md5 doesn't match")
		return false, nil
	}
	return true, nil
}

func findToDelete(index map[string]Installer, fresh []mist.Installer) []Installer {
	ids := make(map[string]bool)
	for _, i := range fresh {
		ids[i.ID] = true
	}

	var toDelete []Installer
	for id, installer := range index {
		if !ids[id] {
			toDelete = append(toDelete, installer)
		}
	}
	return toDelete
}

func findToDownload(index map[string]Installer, fresh []mist.Installer) []mist.Installer {
	var toDownload []mist.Installer
	for _, i := range fresh {
		_, ok := index[i.ID]
		if !ok {
			toDownload = append(toDownload, i)
		}
	}
	return toDownload
}
