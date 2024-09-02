package main

import (
	"flag"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Sidiousious/turtlesubmitter/scouter"
)

func main() {
	sess := os.Getenv("TURTLE_SESSION")
	pass := os.Getenv("TURTLE_PASSWORD")
	dir := detectDefaultLogDirectory()
	turtleUrl := os.Getenv("TURTLE_URL")

	expansions := flag.String("expansions", "", "which expansions to scout, e.g. DT,EW")
	url := flag.String("turtle", turtleUrl, "share URL from turtle, e.g. https://scout.wobbuffet.net/scout/foo/bar")
	lookback := flag.Duration("lookback", 4*time.Hour, "how long to look back in the log file, e.g. 4h. Uses Go duration format. Only looks back in the latest log file.")
	logdir := flag.String("logdir", dir, "directory where the log files are located")
	flag.Parse()

	printGPLNotice()

	var enabledExpansions []string
	if expansions != nil && *expansions != "" {
		enabledExpansions = strings.Split(*expansions, ",")
	}

	if *url != "" {
		parts := strings.Split(*url, "/")
		sess = parts[len(parts)-2]
		pass = parts[len(parts)-1]
	}

	if pass == "" {
		log.Fatal("Turtle session password was not provided. Please provide the share URL as an argument or set TURTLE_PASSWORD")
	}
	if sess == "" {
		log.Fatal("Turtle session was not provided. Please provide the share URL as an argument or set TURTLE_SESSION")
	}

	scouter := scouter.Scouter{Session: sess, Password: pass, Expansions: enabledExpansions, Lookback: *lookback}
	scouter.Run(*logdir)
}

// detectDefaultLogDirectory returns the default log directory for ACT or IINACT
// it uses the IINACTPATH environment variable if set, otherwise it uses checks if
// IINACT's or ACT's default log directory exists and uses that
func detectDefaultLogDirectory() string {
	dir := os.Getenv("IINACTPATH")
	if dir != "" {
		return dir
	}

	defaultACTLogDirectory := path.Join(os.Getenv("APPDATA"), "Advanced Combat Tracker", "FFXIVLogs")
	if _, err := os.Stat(defaultACTLogDirectory); err == nil {
		return defaultACTLogDirectory
	}

	defaultIINACTLogDirectory := path.Join(os.Getenv("HOME"), "Documents", "IINACT")
	if _, err := os.Stat(defaultIINACTLogDirectory); err == nil {
		return defaultIINACTLogDirectory
	}

	log.Fatal("Could not detect default log directory. Please set IINACTPATH")
	return ""
}

func printGPLNotice() {
	log.Println("This software is licensed under the terms of the GNU General Public License v3.0.")
	log.Println("Source code and full license is available at https://github.com/Sidiousious/turtlesubmitter")
}
