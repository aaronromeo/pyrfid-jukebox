package main

import (
	"log"
	"os"

	"aaronromeo.com/rfid-jukebox/system/bin/btconnect"
	"aaronromeo.com/rfid-jukebox/system/bin/helper"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "btconnect",
				Aliases: []string{"b"},
				Usage:   "Maintain a connection to bluetooth device",
				Action: func(c *cli.Context) error {
					connectService := btconnect.NewBtConnectService(&helper.OSCommandExecutor{})
					err := connectService.Run()
					if err != nil {
						log.Fatalf("Command execution failed: %v", err)
					}
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
