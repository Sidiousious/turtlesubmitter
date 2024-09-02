package ioext

import (
	"log"
	"os"
	"time"
)

func GetLatestFile(dir string) os.DirEntry {
	// Get the list of all the files in the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	// Get file modified last
	var lastFile os.DirEntry
	var lastModifTime time.Time
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			log.Fatal(err)
		}
		if lastFile == nil || info.ModTime().After(lastModifTime) {
			lastFile = file
		}
	}
	return lastFile
}
