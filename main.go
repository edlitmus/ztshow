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
	"golang.org/x/crypto/ssh"
	yaml "gopkg.in/yaml.v3"
)

var usr, _ = user.Current()
var configFile = filepath.Join(usr.HomeDir, ".ztssh")
var onlineOnly bool
var hostStyle bool
var hostName string

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
		yamlData, err = ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal("error reading config file: ", err)
		}

		err = yaml.Unmarshal(yamlData, &ztConfig)
		if err != nil {
			log.Fatal(fmt.Sprintf("Unable to parse %s: %s\n", filename, err))
		}
	}
	ztnetwork := ztapi.GetNetworkInfo(
		ztConfig["ZT_API"],
		ztConfig["ZT_URL"],
		ztConfig["ZT_NETWORK"])
	if ztnetwork == nil {
		log.Fatal("network error")
	}

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "list peers",
			Action: func(c *cli.Context) error {
				log.Infof("Getting Members of Network: %s", ztnetwork.Config.Name)
				lst := ztapi.GetMemberList(ztConfig["ZT_API"], ztConfig["ZT_URL"], ztnetwork.ID)
				if lst == nil {
					log.Fatal("Unable to get member list")
				}
				log.Infof("Got %d members", len(*lst))
				names := memberNames(*lst, onlineOnly)
				for _, name := range names {
					if hostStyle {
						fmt.Printf("%s\t%s\n", strings.Join(name.Config.IPAssignments, ", "), name.Name)
					} else {
						fmt.Printf("Name: %s, Online: %t, IPs: %s\n", name.Name, name.Online, strings.Join(name.Config.IPAssignments, ", "))
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
