package main

import (
	"github.com/codegangsta/cli"
	"github.com/monder/kaylee/command"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Usage = "Container orchestration system for fleet"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "etcd-endpoints",
			Value: "http://127.0.0.1:4001,http://127.0.0.1:2379",
			Usage: "a comma-delimited list of etcd endpoints",
		},
		cli.StringFlag{
			Name:  "etcd-prefix",
			Value: "/kaylee",
			Usage: "a keyspace for unit data in etcd",
		},
	}

	app.Commands = []cli.Command{
		command.Server,
		command.Run,
	}

	app.Run(os.Args)
}
