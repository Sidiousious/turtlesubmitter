package main

import (
	"flag"
	"fmt"
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

	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage: %s [flags]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(w, "\nEnvironment variables:\n")
		fmt.Fprintf(w, "  TURTLE_SESSION: Turtle session ID (alternative to -turtle)\n")
		fmt.Fprintf(w, "  TURTLE_PASSWORD: Turtle session password (alternative to -turtle)\n")
		fmt.Fprintf(w, "  TURTLE_URL: Turtle share URL (alternative to -turtle)\n")
		fmt.Fprintf(w, "  IINACTPATH: ACT or IINACT log directory (alternative to -logdir)\n")
		fmt.Fprintf(w, "\nFlags will take precedence over environment variables.\nA new turtle session will be created, if existing password and session are not provided.\n")
	}

	expansions := flag.String("expansions", "", "which expansions to scout, e.g. DT,EW")
	url := flag.String("turtle", turtleUrl, "share URL from turtle, e.g. https://scout.wobbuffet.net/scout/foo/bar")
	lookback := flag.Duration("lookback", 4*time.Hour, "how long to look back in the log file, e.g. 4h. Uses Go duration format. Only looks back in the latest log file.")
	logdir := flag.String("logdir", dir, "directory where the log files are located. Defaults to ACT or IINACT log directory, if those exist (dynamic detection)")
	world := flag.String("world", "Cactuar", "World name to filter by")
	flag.Parse()

	printGPLNotice()

	var enabledExpansions []string
	if expansions != nil && *expansions != "" {
		enabledExpansions = strings.Split(strings.ToUpper(*expansions), ",")
	}

	if *url != "" {
		parts := strings.Split(*url, "/")
		sess = parts[len(parts)-2]
		pass = parts[len(parts)-1]
	}

	if pass == "" {
		newSess, err := scouter.CreateTurtle()
		if err != nil {
			log.Println("Failed to create new turtle session. You can provide an existing session with -turtle or TURTLE_URL. See -help for more information.")
			log.Fatal(err)
		}
		sess = newSess.SessionID
		pass = newSess.Password
		fmt.Println("--------------------------------------------")
		fmt.Printf("  Turtle URL: %s\n", newSess.URL)
		fmt.Printf("Readonly URL: %s\n", newSess.ReadURL)
		fmt.Printf("--------------------------------------------\n\n")
	}

	if sess == "" || pass == "" {
		log.Println("Session ID and password are required. See -help for more information.")
		os.Exit(1)
	}

	scouter := scouter.Scouter{Session: sess, Password: pass, Expansions: enabledExpansions, Lookback: time.Now().Add(-*lookback), World: *world}
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
