package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/anna-oake/macos-please/mist"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend/file"
)

func produceImage(installer mist.Installer, outPath string) error {
	log.Println("-- Creating a temporary directory")
	tmp, err := os.MkdirTemp("", "macos-please")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(tmp)

	size := installer.Size/1024/1024/1024 + 5
	path := filepath.Join(tmp, "installer.dmg")
	log.Printf("-- Preparing a %d GB image", size)
	imgPath, mountPoint, device, parentDevice, err := prepareInstallerImage(path, installer.Name, fmt.Sprintf("%dg", size))
	if err != nil {
		return err
	}
	defer func() {
		if parentDevice != "" {
			hdi.Detach(parentDevice)
		}
	}()
	path = imgPath
	log.Printf("-- Image %s mounted at %s, device %s, parent device %s", path, mountPoint, device, parentDevice)

	log.Printf("-- Downloading and creating a bootable installer for %s (will take some time!)", installer.Name)
	err = mi.CreateBootableInstaller(installer.Version, mountPoint)
	if err != nil {
		return err
	}

	log.Println("-- Waiting 30 seconds for the partition to settle")
	time.Sleep(30 * time.Second) // resize incorrectly calculate limits if done too quickly

	log.Println("-- Shrinking the installer partition")
	err = du.Resize(device, 0)
	if err != nil {
		return err
	}

	log.Println("-- Detaching the image")
	err = hdi.Detach(parentDevice)
	if err != nil {
		return err
	}

	log.Printf("-- Extracting the installer partition to %s", outPath)
	return extractInstallerPartition(path, outPath)
}

func prepareInstallerImage(imagePath, installerName, size string) (imgPath, mountPoint, device, parentDevice string, err error) {
	defer func() {
		if err != nil {
			if imgPath == "" {
				imgPath = imagePath
			}
			if imgPath == "" {
				return
			}
			if parentDevice != "" {
				hdi.Detach(parentDevice)
			} else if mountPoint != "" {
				hdi.Detach(mountPoint)
			}
			os.Remove(imgPath)
		}
	}()

	volumeName := "Install " + installerName
	mountPoint = "/Volumes/" + volumeName
	_, dev, err := hdi.FindDevEntry(mountPoint)
	if err != nil {
		return "", "", "", "", err
	}
	if dev != "" {
		err = hdi.Detach(dev)
		if err != nil {
			return
		}
	}
	imgPath, err = hdi.Create(imagePath, volumeName, size)
	if err != nil {
		return
	}
	_, err = hdi.Attach(imgPath, mountPoint, false)
	if err != nil {
		return
	}
	device, parentDevice, err = hdi.FindDevEntry(mountPoint)
	return
}

func extractInstallerPartition(inPath, outPath string) error {
	in, err := file.OpenFromPath(inPath, false)
	if err != nil {
		return err
	}
	disk, err := diskfs.OpenBackend(in)
	if err != nil {
		return err
	}
	defer disk.Close()
	table, err := disk.GetPartitionTable()
	if err != nil {
		return err
	}
	start := int64(-1)
	var size int64
	parts := table.GetPartitions()
	for _, p := range parts {
		ns := p.GetSize()
		if ns > size {
			start = p.GetStart()
			size = ns
		}
	}
	if start == -1 {
		return fmt.Errorf("no partitions found")
	}
	out, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = in.Seek(start, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = io.CopyN(out, in, size)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}
