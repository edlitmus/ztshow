package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/uxbh/ztdns/ztapi"
	yaml "gopkg.in/yaml.v2"
)

var usr, _ = user.Current()
var configFile = filepath.Join(usr.HomeDir, ".ztssh")
var onlineOnly bool
var hostName string

// ZtSSHData config data
type ZtSSHData map[string]string

func main() {
	var ztConfig ZtSSHData
	filename, err := filepath.Abs(configFile)
	if err != nil {
		log.Fatal(err)
	}
	var yamlData []byte

	if _, err = os.Stat(filename); !os.IsNotExist(err) {
		yamlData, err = ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal("error reading config file: ", err)
		}

		err = yaml.Unmarshal(yamlData, &ztConfig)
		if err != nil {
			log.Fatal(fmt.Sprintf("Unable to parse %s: %s\n", filename, err))
		}
	}
	ztnetwork := ztapi.GetNetworkInfo(ztConfig["ZT_API"], ztConfig["ZT_URL"], ztConfig["ZT_NETWORK"])

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "list peers",
			Action: func(c *cli.Context) error {
				log.Infof("Getting Members of Network: %s", ztnetwork.Config.Name)
				lst := ztapi.GetMemberList(ztConfig["ZT_API"], ztConfig["ZT_URL"], ztnetwork.ID)
				log.Infof("Got %d members", len(*lst))
				names := memberNames(*lst, onlineOnly)
				for _, name := range names {
					fmt.Printf("Name: %s, Online: %t, IPs: %s\n", name.Name, name.Online, strings.Join(name.Config.IPAssignments, ", "))
				}
				return nil
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "online, o",
					Usage:       "online hosts only",
					Destination: &onlineOnly,
				},
			},
		},
		{
			Name:    "connect",
			Aliases: []string{"l"},
			Usage:   "Connect to a ZeroTier host",
			Action: func(c *cli.Context) error {
				log.Infof("Looking for host %s", hostName)
				lst := ztapi.GetMemberList(ztConfig["ZT_API"], ztConfig["ZT_URL"], ztnetwork.ID)
				names := memberNames(*lst, onlineOnly)
				for _, name := range names {
					if name.Name == hostName {
						log.Infof("Found %s…", hostName)
					}
				}
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "hostname, n",
					Usage:       "ZeroTier host name",
					Destination: &hostName,
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func memberNames(list []ztapi.Member, status bool) []ztapi.Member {
	var names []ztapi.Member
	for index := 0; index < len(list); index++ {
		if status && !list[index].Online {
			continue
		}
		names = append(names, list[index])
	}
	return names
}
