package hdiutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-cmd/cmd"
	"howett.net/plist"
)

type HDIUtil struct {
}

func New() (*HDIUtil, error) {
	h := &HDIUtil{}
	_, err := h.GetMountedImages()
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (h *HDIUtil) Create(imagePath, volName, size string) (string, error) {
	args := []string{"create", "-size", size, "-layout", "GPTSPUD", "-fs", "JHFS+", "-nospotlight", "-type", "UDIF", "-volname", volName, imagePath, "-plist"}
	out, err := h.runSync(args...)
	if err != nil {
		return "", err
	}
	var s []string
	_, err = plist.Unmarshal([]byte(out), &s)
	if err != nil {
		return "", nil
	}
	if len(s) == 0 {
		return "", errors.New("no output")
	}
	return s[0], nil
}

func (h *HDIUtil) Attach(imagePath string, mountPoint string, readOnly bool) (*ImageInfo, error) {
	args := []string{"attach", imagePath, "-plist"}
	if mountPoint != "" {
		args = append(args, "-mountpoint", mountPoint)
	}
	if readOnly {
		args = append(args, "-readonly")
	} else {
		args = append(args, "-readwrite")
	}
	out, err := h.runSync(args...)
	if err != nil {
		return nil, err
	}
	var i ImageInfo
	_, err = plist.Unmarshal([]byte(out), &i)
	if err != nil {
		return nil, err
	}
	if mountPoint != "" {
		var found bool
		for _, e := range i.Entities {
			if e.MountPoint == mountPoint {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("failed to mount at requested mount point %s", mountPoint)
		}
	}
	i.ImagePath = imagePath
	return &i, nil
}

func (h *HDIUtil) Detach(path string) error {
	_, err := h.runSync("detach", "-force", path)
	return err
}

func (h *HDIUtil) FindDevEntry(mountPoint string) (partition, parent string, err error) {
	imgs, err := h.GetMountedImages()
	if err != nil {
		return "", "", err
	}
	var img *ImageInfo
	for _, i := range imgs {
		for _, e := range i.Entities {
			if e.MountPoint == mountPoint {
				img = &i
				partition = e.Device
				break
			}
		}
		if img != nil {
			break
		}
	}
	if img == nil {
		return "", "", nil
	}

	for _, e := range img.Entities {
		if parent == "" || len(e.Device) < len(parent) {
			parent = e.Device
		}
	}
	return
}

func (h *HDIUtil) GetMountedImages() ([]ImageInfo, error) {
	out, err := h.runSync("info", "-plist")
	if err != nil {
		return nil, err
	}
	var i infoResult
	_, err = plist.Unmarshal([]byte(out), &i)
	if err != nil {
		return nil, err
	}
	return i.Images, nil
}

func (h *HDIUtil) runSync(args ...string) (string, error) {
	c := cmd.NewCmd("hdiutil", args...)
	s := <-c.Start()
	if s.Error == nil && len(s.Stderr) != 0 {
		return "", errors.New(strings.Join(s.Stderr, "\n"))
	}
	return strings.Join(s.Stdout, "\n"), s.Error
}

type infoResult struct {
	Images []ImageInfo `plist:"images"`
}

type ImageInfo struct {
	ImagePath string        `plist:"image-path"`
	Entities  []ImageEntity `plist:"system-entities"`
}

type ImageEntity struct {
	Device     string `plist:"dev-entry"`
	MountPoint string `plist:"mount-point"`
}
