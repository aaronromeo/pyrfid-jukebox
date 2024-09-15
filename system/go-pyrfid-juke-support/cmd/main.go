package main

import (
	"log"
	"os"

	"aaronromeo.com/go-pyrfid-juke-support/btconnect"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

func main() {
	err := godotenv.Load("$HOME/.soundsprout/conf")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "btconnect",
				Aliases: []string{"b"},
				Usage:   "Maintain a connection to bluetooth device",
				Action: func(c *cli.Context) error {
					connectService := btconnect.NewBtConnectService(&btconnect.OSCommandExecutor{})
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

// Additional functions to replicate other parts of btconnect.sh
