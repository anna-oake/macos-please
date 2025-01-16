package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/anna-oake/macos-please/diskutil"
	"github.com/anna-oake/macos-please/hdiutil"
	"github.com/anna-oake/macos-please/mist"
)

var hdi *hdiutil.HDIUtil
var du *diskutil.DiskUtil
var mi *mist.Mist

func main() {
	args := parseArgs()

	if os.Geteuid() != 0 {
		log.Fatalln("Please run as root")
	}

	log.Printf("Starting with output directory %s", args.OutputDir)

	indexPath, err := getIndexPath(args.OutputDir)
	if err != nil {
		log.Fatalln("Index:", err)
	}
	index, err := loadIndex(indexPath)
	if err != nil {
		log.Fatalln("Index:", err)
	}

	hdi, err = hdiutil.New()
	if err != nil {
		log.Fatalln("hdiutil init error:", err)
	}
	du, err = diskutil.New()
	if err != nil {
		log.Fatalln("diskutil init error:", err)
	}
	mi, err = mist.New(args.MistCache, args.MistTimeout)
	if err != nil {
		log.Fatalln("mist init error:", err)
	}

	if len(index) > 0 {
		log.Println("Verifying existing installers")
	}

	for id, i := range index {
		log.Printf("- Verifying %s", i.Filename)
		valid, err := verifyInstaller(i, args.OutputDir)
		if err != nil {
			log.Println("-- Couldn't verify:", err)
		}
		if valid {
			continue
		}
		log.Println("-- Deleting...")
		os.Remove(filepath.Join(args.OutputDir, i.Filename))
		delete(index, id)
		saveIndex(indexPath, index)
	}

	var opts []string
	if args.OnlyLatest {
		opts = append(opts, "only latest minor versions")
	}
	if args.MajorLimit > 0 {
		opts = append(opts, fmt.Sprintf("for %d major versions", args.MajorLimit))
	}
	msg := "Fetching installers"
	if len(opts) > 0 {
		msg += fmt.Sprintf(" (%s)", strings.Join(opts, ", "))
	}

	log.Println(msg)
	ins, err := mi.ListInstallers(args.OnlyLatest, args.MajorLimit)
	if err != nil {
		log.Fatalln("- Couldn't fetch:", err)
	}

	if !args.KeepOld {
		toDelete := findToDelete(index, ins)
		if len(toDelete) > 0 {
			log.Printf("Deleting %d old installers", len(toDelete))
		}
		for _, i := range toDelete {
			log.Printf("- Deleting %s", i.Filename)
			os.Remove(filepath.Join(args.OutputDir, i.Filename))
			delete(index, i.ID)
			saveIndex(indexPath, index)
		}
	}

	toDownload := findToDownload(index, ins)

	if len(toDownload) == 0 {
		log.Println("Nothing to download, finished")
		return
	}

	log.Printf("Downloading %d installers", len(toDownload))

	for _, i := range toDownload {
		name := fmt.Sprintf("%s %s %s", i.Name, i.Version, i.Build)
		name = strings.ReplaceAll(name, " ", "-") + ".img"
		log.Printf("- Downloading and producing %s -", name)
		path := filepath.Join(args.OutputDir, name)
		err := produceImage(i, path)
		if err != nil {
			log.Println("-- Couldn't produce:", err)
			continue
		}
		log.Println("-- Generating metadata")
		meta, err := generateMetadata(i, path)
		if err != nil {
			log.Println("-- Couldn't generate metadata:", err)
			continue
		}
		index[meta.ID] = *meta
		saveIndex(indexPath, index)
		log.Println("-- Done")
	}

	log.Println("Finished")
}
