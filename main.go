package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/uxbh/ztdns/ztapi"
	"golang.org/x/crypto/ssh"
	yaml "gopkg.in/yaml.v2"
)

var usr, _ = user.Current()
var configFile = filepath.Join(usr.HomeDir, ".ztssh")
var onlineOnly bool
var hostName string

// ZtSSHData config data
type ZtSSHData map[string]string

var ztConfig ZtSSHData

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
				connect(*ztnetwork)
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

func connect(ztnetwork ztapi.Network) {
	usrname := usr.Username
	var host string
	var err error
	port := 22

	if strings.Contains(hostName, "@") {
		userHost := strings.Split(hostName, "@")
		if len(userHost) > 0 {
			if userHost[0] != usrname {
				usrname = userHost[0]
			}
			hostName = userHost[1]
		}
	}
	if strings.Contains(hostName, ":") {
		hostPort := strings.Split(hostName, ":")
		if len(hostPort) > 0 {
			host = hostPort[0]
			port, err = strconv.Atoi(hostPort[1])
			if err != nil {
				log.Fatal(fmt.Sprintf("Unable to parse port '%s': %s\n", hostPort[1], err))
			}
		}
	}
	if host == "" {
		host = hostName
	}

	log.Infof("Looking for host %s in network %s", hostName, ztnetwork.ID)
	lst := ztapi.GetMemberList(ztConfig["ZT_API"], ztConfig["ZT_URL"], ztnetwork.ID)
	names := memberNames(*lst, onlineOnly)
	for _, name := range names {
		if name.Name == host {
			connectToHost(name, usrname, host, port)
			break
		}
	}
}

func connectToHost(name ztapi.Member, usrname string, host string, port int) {
	agent := sshAgent()
	if agent == nil {
		log.Warn("Can't connect with ssh-agent")
	}
	privateKeyPath := filepath.Join(usr.HomeDir, ".ssh/id_rsa")
	pubKeys := publicKeyFile(privateKeyPath)
	if pubKeys == nil {
		log.Fatalf("Can't get public keys from %s", privateKeyPath)
	}
	sshConfig := &ssh.ClientConfig{
		User: usrname,
		Auth: []ssh.AuthMethod{
			pubKeys,
			agent,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connStr := fmt.Sprintf("%s:%d", name.Config.IPAssignments[0], port)
	log.Infof("Trying to connect as %s via %s…", usrname, connStr)
	err := newSession(sshConfig, connStr)
	if err != nil {
		log.Fatal(err)
	}
}
