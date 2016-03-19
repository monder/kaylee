package main

import (
	"github.com/codegangsta/cli"
	"github.com/monder/kaylee/commands"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0"
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
			Value: "/kaylee",
			Usage: "a keyspace for unit data in etcd",
		},
	}

	app.Commands = []cli.Command{
		commands.NewServerCommand(),
		commands.NewRunCommand(),
		commands.NewLsCommand(),
		commands.NewRestartCommand(),
	}

	app.Run(os.Args)
}
