package ioext

import (
	"log"
	"os"
	"regexp"
	"time"
)

var (
	logFileFormatPattern = regexp.MustCompile(`^Network.*\.log$`)
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
		if !logFileFormatPattern.MatchString(info.Name()) {
			continue // Skip files that do not match the log file format
		}
		if lastFile == nil || info.ModTime().After(lastModifTime) {
			lastFile = file
		}
	}
	return lastFile
}
