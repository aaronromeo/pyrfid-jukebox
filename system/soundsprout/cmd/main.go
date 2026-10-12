package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"aaronromeo.com/soundsprout/bluetooth"
	"aaronromeo.com/soundsprout/btconnect"
	"aaronromeo.com/soundsprout/portal"
	"aaronromeo.com/soundsprout/templategen"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// readHeaderTimeout bounds reading the portal's HTTP request headers.
const readHeaderTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := godotenv.Load("/home/pi/.soundsprout/conf")
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
						bluetooth.OSRunner{},
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
			{
				Name:    "templategen",
				Aliases: []string{"t"},
				Usage:   "Generate the templates needed for this service",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "output-dir",
						Aliases:  []string{"o"},
						Value:    "./outputs",
						Required: true,
					},
				},
				Action: func(ctx *cli.Context) error {
					var outputPath string
					outputPath, err = filepath.Abs(ctx.String("output-dir"))
					if err != nil {
						return err
					}

					_, err = os.Stat(outputPath)
					if err != nil && os.IsNotExist(err) {
						return fmt.Errorf("output path '%s' must exist", outputPath)
					}

					templateService := templategen.NewTemplateGenService(
						logger,
						outputPath,
					)

					err = templateService.Run()
					if err != nil {
						logger.Error(
							"templategen failure",
							"error", err,
						)
					}
					return nil
				},
			},
			{
				Name:  "serve",
				Usage: "Run the web portal",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Value: "0.0.0.0:8080",
					},
				},
				Action: func(ctx *cli.Context) error {
					portalServer, srvErr := portal.NewService(logger)
					if srvErr != nil {
						return srvErr
					}
					logger.Info("portal listening", "addr", ctx.String("addr"))
					httpServer := &http.Server{
						Addr:              ctx.String("addr"),
						Handler:           portalServer.Routes(),
						ReadHeaderTimeout: readHeaderTimeout,
					}
					return httpServer.ListenAndServe()
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
