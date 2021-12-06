package app

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/generator"
	"github.com/uphy/feedgen/generator/cssselector"
	"github.com/uphy/feedgen/repo"
	"github.com/urfave/cli/v2"
)

type App struct {
	app           *cli.App
	repository    *repo.Repository
	feedGenerator *generator.FeedGenerators
}

func New() *App {
	a := cli.NewApp()
	app := &App{a, nil, nil}

	a.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Value:   "config.yml",
		},
	}
	a.Before = func(c *cli.Context) error {
		// load repository
		repository, err := repo.NewBadgerRepository("data")
		if err != nil {
			return err
		}
		app.repository = repository

		// load config
		configFile := c.String("config")
		cnf, err := config.ParseConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: configFile=%s, err=%w", configFile, err)
		}
		// build feed generator
		gen := generator.New(repository)
		gen.Register("css-selector", cssselector.CSSSelectorFeedGenerator{})
		if err := gen.LoadConfig(cnf); err != nil {
			return err
		}
		app.feedGenerator = gen
		return nil
	}
	a.After = func(c *cli.Context) error {
		app.repository.Close()
		return nil
	}

	a.Commands = []*cli.Command{
		app.generateCommand(),
		app.startServerCommand(),
	}
	return app
}

func (a *App) generateCommand() *cli.Command {
	return &cli.Command{
		Name: "generate",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "atom",
				Usage:   "Export format (atom/rss)",
			},
		},
		ArgsUsage: "Name of the feed in config file",
		Action: func(c *cli.Context) error {
			format := c.String("format")
			feedName := c.Args().First()

			s, err := a.generateFeed(feedName, format)
			if err != nil {
				return err
			}
			fmt.Println(s)
			return nil
		},
	}
}

func (a *App) generateFeed(feedName string, format string) (string, error) {
	feed, err := a.feedGenerator.Generate(feedName)
	if err != nil {
		return "", err
	}

	switch format {
	case "atom":
		atom, err := feed.ToAtom()
		if err != nil {
			return "", err
		}
		return atom, nil
	case "rss":
		rss, err := feed.ToRss()
		if err != nil {
			return "", err
		}
		return rss, nil
	}
	return "", nil
}

func (a *App) startServerCommand() *cli.Command {
	return &cli.Command{
		Name: "start-server",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				EnvVars: []string{"FEED_GEN_PORT"},
				Value:   8080,
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")

			e := echo.New()
			e.GET("feed/:name/:format", func(c echo.Context) error {
				name := c.Param("name")
				format := c.Param("format")
				s, err := a.generateFeed(name, format)
				if err != nil {
					c.Logger().Error("failed to generate: name=%s, err=%w", name, err)
					return err
				}
				return c.Blob(200, "application/xml", []byte(s))
			})
			e.Start(fmt.Sprintf(":%d", port))
			return nil
		},
	}
}
func (a *App) Run(args []string) error {
	return a.app.Run(args)
}
