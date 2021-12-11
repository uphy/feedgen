package app

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/converter"
	"github.com/uphy/feedgen/generator"
	"github.com/uphy/feedgen/generator/template"
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
		&cli.BoolFlag{
			Name:    "no-cache",
			Aliases: []string{"n"},
			Value:   false,
		},
	}
	a.Before = func(c *cli.Context) error {
		// load repository
		if c.Bool("no-cache") {
			app.repository = repo.NewMemoryRepository()
		} else {
			repository, err := repo.NewBadgerRepository("data")
			if err != nil {
				return err
			}
			app.repository = repository
		}

		// load config
		configFile := c.String("config")
		cnf, err := config.ParseConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config file: configFile=%s, err=%w", configFile, err)
		}
		// build feed generator
		gen := generator.New(app.repository)
		gen.Register("template", template.TemplateFeedGenerator{})
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
			&cli.StringSliceFlag{
				Name:    "parameter",
				Aliases: []string{"p"},
				Usage:   "Parameter for the generators",
			},
		},
		ArgsUsage: "Name of the feed in config file",
		Action: func(c *cli.Context) error {
			format := c.String("format")
			feedName := c.Args().First()

			parameterMap := make(map[string]string, 0)
			parameters := c.StringSlice("parameter")
			for _, parameter := range parameters {
				eqIndex := strings.Index(parameter, "=")
				key := parameter[0:eqIndex]
				value := parameter[eqIndex+1:]
				parameterMap[key] = value
			}

			result, err := a.generateFeed(feedName, format, parameterMap)
			if err != nil {
				return err
			}
			fmt.Println(result.Result)
			return nil
		},
	}
}

func (a *App) generateFeed(feedName string, format string, parameters map[string]string) (*converter.Result, error) {
	feed, err := a.feedGenerator.Generate(feedName, parameters)
	if err != nil {
		return nil, err
	}

	converter := converter.GetConverter(format)
	if converter == nil {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	return converter.Convert(feed)
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
			for name, g := range a.feedGenerator.Generators {
				generatorName := name
				e.GET(g.Endpoint, func(c echo.Context) error {
					parameters := make(map[string]string)
					for _, paramName := range c.ParamNames() {
						parameters[paramName] = c.Param(paramName)
					}
					format := c.QueryParam("format")
					if format == "" {
						format = "rss"
					}
					result, err := a.generateFeed(generatorName, format, parameters)
					if err != nil {
						c.Logger().Errorf("failed to generate: name=%s, err=%s", name, err)
						return err
					}
					return c.Blob(200, result.ContentType, []byte(result.Result))
				})
			}
			e.Start(fmt.Sprintf(":%d", port))
			return nil
		},
	}
}
func (a *App) Run(args []string) error {
	return a.app.Run(args)
}
