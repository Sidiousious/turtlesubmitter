package scouter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Sidiousious/turtlesubmitter/ioext"
)

type Scouter struct {
	Session    string
	Password   string
	Expansions []string
	Lookback   time.Duration
}

func (s *Scouter) Run(dir string) {
	log.Printf("Scouting to https://scout.wobbuffet.net/scout/%s/%s", s.Session, s.Password)

	logFile := ioext.GetLatestFile(dir)
	logFilePath := path.Join(dir, logFile.Name())
	log.Printf("Latest file: %s", logFilePath)

	// Continuously read the file for new lines
	h, err := ioext.NewTailReader(logFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer h.Close()

	scanner := bufio.NewScanner(h)

	acceptedMobs := []string{}
	if s.Expansions == nil {
		for _, mobs := range mobNames {
			for mob := range mobs {
				acceptedMobs = append(acceptedMobs, mob)
			}
		}
	} else {
		for _, expansion := range s.Expansions {
			if mobs, ok := mobNames[expansion]; ok {
				for mob := range mobs {
					acceptedMobs = append(acceptedMobs, mob)
				}
			}
		}
	}

	mutex := sync.Mutex{}
	dueForSend := false
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	mobs := make(map[string]*Mob)

	go func() {
		for range ticker.C {
			func() {
				if dueForSend {
					mutex.Lock()
					dueForSend = false
					defer mutex.Unlock()
					s.sendMobs(mobs)
				}
			}()
		}
	}()

	for scanner.Scan() {
		line := scanner.Text()
		mob := s.parseLine(line)
		if mob != nil {
			if !contains(acceptedMobs, mob.Name) {
				continue
			}
			mutex.Lock()
			mobs[mob.Name+strconv.FormatUint(uint64(mob.Instance), 10)] = mob
			dueForSend = true
			mutex.Unlock()
		}
	}
}

type TurtleSightings struct {
	CollaboratorPassword string           `json:"collaborator_password"`
	Sightings            []TurtleSighting `json:"sightings"`
}

type TurtleSighting struct {
	ZoneID         uint   `json:"zone_id"`
	MobID          uint   `json:"mob_id"`
	InstanceNumber uint   `json:"instance_number"`
	X              string `json:"x"`
	Y              string `json:"y"`
}

func NewTurtleSighting(mob *Mob) *TurtleSighting {
	id := uint(1)
outer:
	for _, mobs := range mobNames {
		for name, mobId := range mobs {
			if name == mob.Name {
				id = mobId
				break outer
			}
		}
	}

	return &TurtleSighting{
		ZoneID:         mob.Zone,
		MobID:          id,
		InstanceNumber: mob.Instance,
		X:              strconv.FormatFloat(mob.PosX, 'f', -1, 64),
		Y:              strconv.FormatFloat(mob.PosY, 'f', -1, 64),
	}
}

type Mob struct {
	Name     string
	PosX     float64
	PosY     float64
	Zone     uint
	Instance uint
}

// sendMobs sends the list of found mobs to the turtle server
func (s *Scouter) sendMobs(mobs map[string]*Mob) {
	// PATCH https://scout.wobbuffet.net/api/v1/scout/<session>
	// {"collaborator_password": "<pass>", "sightings": [{"zone_id": uint, "mob_id": uint, "instance_number": uint, "x": string, "y": string}]}

	url := "https://scout.wobbuffet.net/api/v1/scout/" + s.Session
	log.Print("Sending mobs to ", url)
	sightings := TurtleSightings{
		CollaboratorPassword: s.Password,
		Sightings:            make([]TurtleSighting, 0, len(mobs)),
	}
	for _, mob := range mobs {
		sightings.Sightings = append(sightings.Sightings, *NewTurtleSighting(mob))
	}

	body, err := json.Marshal(sightings)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		log.Print(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Print(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		resBody, _ := io.ReadAll(res.Body)
		log.Print("Failed to send mobs:", res.Status)
		log.Print(string(resBody))
	} else {
		log.Print("Mobs successfully sent")
	}
}

func (s *Scouter) parseLine(line string) *Mob {
	// 261|2024-08-21T17:07:49.3900000+03:00|Add|40034AD3|BNpcID|43DC|BNpcNameID|3459|CastTargetID|E0000000|CurrentMP|10000|CurrentWorldID|65535|Heading|1.6686|Level|100|MaxHP|32956266|MaxMP|10000|Name|Keheniheyamewi|NPCTargetID|10812C10|PosX|500.9187|PosY|83.0748|PosZ|-2.0892|Radius|8.5000|Type|2|WorldID|65535|2c4bf29b9acde190

	parts := strings.Split(line, "|")

	date := parts[1]
	d, err := time.Parse(time.RFC3339, date)
	if err == nil {
		if d.Before(time.Now().Add(-s.Lookback)) {
			return nil
		}
	}

	switch parts[0] {
	case "00":
		return parseChatFlag(parts, d)
		// case "261":
		// 	return parseAdd(parts)
		// case "01":
		// 	s.parseZone(parts)
		// 	return nil
	}
	return nil
}

// parseChatFlag parses a chat flag line and returns a Mob with its spawnpoint coordinates, if the message contains a mob name
func parseChatFlag(parts []string, timestamp time.Time) *Mob {
	msg := parts[4]
	mobName := findMobName(msg)
	if mobName == "" {
		return nil
	}
	mob := &Mob{
		Name: mobName,
	}

	re := regexp.MustCompile(`\x{E0BB}(?P<Zone>[A-Za-z' ]+?)(?P<Instance>[\x{E0B1}-\x{E0B6}])? \( ?(?P<PosX>\d+\.\d+) *?, ?(?P<PosY>\d+\.\d+)`)
	matches := re.FindStringSubmatch(msg)
	if len(matches) == 0 {
		return nil
	}
	zone := matches[1]
	instance := matches[2]
	// Cast instance to a number
	if instance != "" {
		instNum, _ := utf8.DecodeRuneInString(instance)
		mob.Instance = uint(instNum - 57520)
	} else {
		mob.Instance = 1
	}
	mob.Zone = zones[zone]
	mob.PosX = asFloat(matches[3])
	mob.PosY = asFloat(matches[4])
	log.Printf("Mob: %+v in %s at %v", mob, zone, timestamp)
	spawn := findClosestSpawnpoint(mob)
	mob.PosX = spawn.X
	mob.PosY = spawn.Y
	return mob
}

// findClosestSpawnpoint returns the closest defined spawnpoint for the given mob
func findClosestSpawnpoint(mob *Mob) Point {
	var closest Point
	closestDistance := -1.0
	for _, spawnpoint := range spawnpoints[mob.Zone] {
		distance := math.Abs(spawnpoint.X-mob.PosX) + math.Abs(spawnpoint.Y-mob.PosY)
		if closestDistance == -1 || distance < closestDistance {
			closest = spawnpoint
			closestDistance = distance
		}
	}
	return closest
}

// findMobName returns the name of the mob in the given message, if one exists, or empty string otherwise
func findMobName(msg string) string {
	for _, mobs := range mobNames {
		for mob := range mobs {
			if strings.Contains(msg, mob) {
				return mob
			}
		}
	}
	return ""
}

// func (s *Scouter) parseZone(parts []string) {
// 	zone, err := strconv.ParseUint(parts[2], 16, 32)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	s.currentZone = uint(zone)
// 	log.Printf("Zone: %d", s.currentZone)
// }

// func parseAdd(parts []string) *Mob {
// 	mob := &Mob{}
// 	for i := 4; i < len(parts); i += 2 {
// 		switch parts[i] {
// 		case "Name":
// 			mob.Name = parts[i+1]
// 		case "PosX":
// 			mob.PosX = asFloat(parts[i+1])
// 		case "PosY":
// 			mob.PosY = asFloat(parts[i+1])
// 		}
// 	}
// 	return mob
// }

func asFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -2132831721
	}
	return f
}

func contains[T comparable](haystack []T, needle T) bool {
	for _, x := range haystack {
		if x == needle {
			return true
		}
	}
	return false
}

// func mapZoneCoordSize(zone uint) float64 {
// 	if zone >= 397 && zone <= 402 {
// 		return 43.1
// 	}
// 	return 41.0
// }
