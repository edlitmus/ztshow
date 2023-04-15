package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
const FourtyEightHours = 172800

// ZtShowData config data
type ZtShowData map[string]string

var ztConfig ZtShowData

func main() {
	filename, err := filepath.Abs(configFile)
	if err != nil {
		log.Fatal(err)
	}
	var yamlData []byte

	if _, err = os.Stat(filename); !os.IsNotExist(err) {
		yamlData, err = os.ReadFile(filename)
		if err != nil {
			log.Fatal("error reading config file: ", err)
		}

		err = yaml.Unmarshal(yamlData, &ztConfig)
		if err != nil {
			log.Fatal(fmt.Sprintf("Unable to parse %s: %s\n", filename, err))
		}
	}

	ztnetwork, err := ztcentral.NewClient(ztConfig["ZT_API"])
	if err != nil {
		log.Fatal("network error")
	}

	ctx := context.Background()

	// get list of networks
	networks, err := ztnetwork.GetNetworks(ctx)
	if err != nil {
		log.Println("error:", err.Error())
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
					log.Infof("Getting Members of Network: %s", *n.Config.Name)
					members, err := ztnetwork.GetMembers(ctx, *n.Id)
					if err != nil {
						log.Fatal("Unable to get member list")
						os.Exit(1)
					}

					names := memberNames(members, onlineOnly)
					log.Infof("Got %d members", len(names))

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
		log.Fatal(err)
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
	online := true
	if *m.Config.LastAuthorizedTime == int64(0) || *m.LastOnline == int64(0) || timedOut(*m.Config.LastAuthorizedTime) {
		online = false
	}
	return online
}

func timedOut(lastSeen int64) bool {
	now := time.Now().Unix()
	return now-lastSeen >= FourtyEightHours
}
