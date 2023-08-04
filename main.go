package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli"

	ztcentral "github.com/zerotier/go-ztcentral"
	"github.com/zerotier/go-ztcentral/pkg/spec"
	yaml "gopkg.in/yaml.v3"
)

var usr, _ = user.Current()
var configFile = filepath.Join(usr.HomeDir, ".ztshow")
var onlineOnly bool
var hostStyle bool

// Number of seconds in 48 hours
const FortyEightHours = 172800

// ZtShowData config data
type ZtShowData map[string]string

var ztConfig ZtShowData

func main() {
	logger := log.New(os.Stderr, "", 0)
	filename, err := filepath.Abs(configFile)
	if err != nil {
		logger.Fatal(err)
	}
	var yamlData []byte

	if _, err = os.Stat(filename); !os.IsNotExist(err) {
		yamlData, err = os.ReadFile(filename)
		if err != nil {
			logger.Fatal("error reading config file: ", err)
		}

		err = yaml.Unmarshal(yamlData, &ztConfig)
		if err != nil {
			logger.Fatalf(fmt.Sprintf("Unable to parse %s: %s\n", filename, err))
		}
	}

	ztnetwork, err := ztcentral.NewClient(ztConfig["ZT_API"])
	if err != nil {
		logger.Fatal("network error")
	}

	ctx := context.Background()

	// get list of networks
	networks, err := ztnetwork.GetNetworks(ctx)
	if err != nil {
		logger.Println("error:", err.Error())
		os.Exit(1)
	}

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "list peers",
			Action: func(c *cli.Context) error {
				// print networks and members
				for _, n := range networks {
					logger.Printf("Getting Members of Network: %s", *n.Config.Name)
					members, err := ztnetwork.GetMembers(ctx, *n.Id)
					if err != nil {
						logger.Fatal("Unable to get member list")
						os.Exit(1)
					}

					names := memberNames(members, onlineOnly)
					logger.Printf("Got %d members", len(names))

					for _, name := range names {
						if hostStyle {
							fmt.Printf("%s\t%s\n", strings.Join(*name.Config.IpAssignments, ", "), *name.Name)
						} else {
							fmt.Printf("Name: %s, Online: %t, IPs: %s\n", *name.Name, isOnline(name), strings.Join(*name.Config.IpAssignments, ", "))
						}
					}
				}

				return nil
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "online, o",
					Usage:       "online hosts only",
					Destination: &onlineOnly,
				},
				cli.BoolFlag{
					Name:        "hostfile",
					Usage:       "hosts file output",
					Destination: &hostStyle,
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}

func memberNames(list []*spec.Member, status bool) []*spec.Member {
	var names []*spec.Member
	for _, m := range list {
		online := isOnline(m)

		if status && !online {
			continue
		}
		names = append(names, m)
	}
	return names
}

func isOnline(m *spec.Member) bool {
	online := timedOut(*m.LastOnline)
	return online
}

func timedOut(lastSeen int64) bool {
	if lastSeen > 0 {
		now := time.Now()
		last := time.UnixMilli(lastSeen)
		diff := now.Unix() - last.Unix()
		return diff < FortyEightHours
	}
	return false
}

// DumpThing is useful for debugging
func dumper(thing interface{}) {
	json, err := json.MarshalIndent(thing, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(json))
}
