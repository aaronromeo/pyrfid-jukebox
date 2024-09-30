package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"aaronromeo.com/go-pyrfid-juke-support/btconnect"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := godotenv.Load(filepath.Join("home", "pi", ".soundsprout", "conf"))
	if err != nil {
		logger.Error("Error loading .env file")
	}

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "btconnect",
				Aliases: []string{"b"},
				Usage:   "Maintain a connection to bluetooth device",
				Action: func(c *cli.Context) error {
					connectService := btconnect.NewBtConnectService(
						&btconnect.OSCommandExecutor{},
						logger,
					)
					err = connectService.Run()
					if err != nil {
						logger.Error(
							"btconnect failure",
							"error", err,
						)
					}
					return nil
				},
			},
		},
	}

	if err = app.Run(os.Args); err != nil {
		logger.Error(
			"failure on run",
			"args", os.Args,
			"error", err,
		)
	}
}

// Additional functions to replicate other parts of btconnect.sh
