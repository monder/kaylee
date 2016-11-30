package main

import (
	"flag"
	"github.com/monder/kaylee/commands"
	"gopkg.in/urfave/cli.v1"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Version = "2.3.0"
	app.Usage = "Container orchestration system for fleet"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "etcd-endpoints",
			Value:  "http://127.0.0.1:4001,http://127.0.0.1:2379",
			Usage:  "a comma-delimited list of etcd endpoints",
			EnvVar: "ETCDCTL_ENDPOINT",
		},
		cli.StringFlag{
			Name:  "etcd-prefix",
			Value: "/kaylee2",
			Usage: "a keyspace for unit data in etcd",
		},
	}

	app.Commands = []cli.Command{
		commands.Server,
		commands.NewRunCommand(),
		commands.Ls,
		commands.Restart,
	}

	flag.CommandLine.Parse(nil)
	app.Run(os.Args)
}
