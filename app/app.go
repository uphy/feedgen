package app

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/labstack/echo/v4"
	"github.com/uphy/feedgen/config"
	"github.com/uphy/feedgen/converter"
	"github.com/uphy/feedgen/generator"
	"github.com/uphy/feedgen/generator/browser"
	"github.com/uphy/feedgen/generator/template"
	"github.com/uphy/feedgen/repo"
	"github.com/urfave/cli/v2"
)

type App struct {
	app           *cli.App
	repository    *repo.Repository
	configFile    string
	feedGenerator *generator.FeedGenerators
}

func New() *App {
	a := cli.NewApp()
	app := &App{
		app: a,
	}

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
		&cli.BoolFlag{
			Name:  "no-sandbox",
			Value: false,
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
		app.configFile = configFile
		return app.reloadConfig(c)
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

func (a *App) reloadConfig(c *cli.Context) error {
	// load config
	cnf, err := config.ParseConfig(a.configFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: configFile=%s, err=%w", a.configFile, err)
	}
	// build feed generator
	gen := generator.New(a.repository)
	gen.Register("template", template.TemplateFeedGenerator{})
	gen.RegisterFactory("browser", func() generator.FeedGenerator {
		noSandbox := c.Bool("no-sandbox")
		return browser.New(noSandbox)
	})
	if err := gen.LoadConfig(cnf); err != nil {
		return err
	}
	a.feedGenerator = gen
	return nil
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
			&cli.StringSliceFlag{
				Name:    "query-parameter",
				Aliases: []string{"q"},
				Usage:   "Query parameter for the generators",
			},
		},
		ArgsUsage: "Name of the feed in config file",
		Action: func(c *cli.Context) error {
			format := c.String("format")
			feedName := c.Args().First()

			parameterMap := make(map[string]string, 0)
			{
				parameters := c.StringSlice("parameter")
				for _, parameter := range parameters {
					eqIndex := strings.Index(parameter, "=")
					key := parameter[0:eqIndex]
					value := parameter[eqIndex+1:]
					parameterMap[key] = value
				}
			}

			queryParams := make(url.Values, 0)
			{
				parameters := c.StringSlice("query-parameter")
				for _, parameter := range parameters {
					eqIndex := strings.Index(parameter, "=")
					key := parameter[0:eqIndex]
					value := parameter[eqIndex+1:]
					queryParams.Add(key, value)
				}
			}

			result, err := a.generateFeed(feedName, format, parameterMap, queryParams)
			if err != nil {
				return err
			}
			fmt.Println(result.Result)
			return nil
		},
	}
}

func (a *App) generateFeed(feedName string, format string, parameters map[string]string, queryParameters url.Values) (*converter.Result, error) {
	feed, err := a.feedGenerator.Generate(feedName, parameters, queryParameters)
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
				EnvVars: []string{"FEED_GEN_PORT", "PORT"},
				Value:   8080,
			},
			&cli.BoolFlag{
				Name:    "watch",
				Aliases: []string{"w"},
			},
		},
		Action: func(c *cli.Context) error {
			port := c.Int("port")
			watch := c.Bool("watch")

			restartCh := make(chan struct{}, 1)
			if watch {
				go a.watchConfigFile(func() {
					log.Println("Reload config")
					if err := a.reloadConfig(c); err != nil {
						log.Printf("Failed to reload config: %s", err)
						return
					}
					log.Println("Restart server")
					restartCh <- struct{}{}
				})
			}

			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		l:
			for {
				stopCh := make(chan struct{}, 1)
				go a.startServer(port, stopCh)
				select {
				case <-sig:
					break l
				case <-restartCh:
					stopCh <- struct{}{}
				}
			}
			return nil
		},
	}
}

func (a *App) startServer(port int, stopChan <-chan struct{}) {
	e := echo.New()
	e.HideBanner = true
	for name, g := range a.feedGenerator.Generators {
		e.GET(g.Endpoint, a.generateFeedHandlerFunc(name, g))
	}

	log.Printf("Start server at %d", port)
	go e.Start(fmt.Sprintf(":%d", port))

	<-stopChan
	log.Println("Shutdown server")
	e.Shutdown(context.TODO())
}

func (a *App) generateFeedHandlerFunc(name string, g *generator.FeedGeneratorWrapper) echo.HandlerFunc {
	return func(c echo.Context) error {
		parameters := make(map[string]string)
		for _, paramName := range c.ParamNames() {
			parameters[paramName] = c.Param(paramName)
		}
		format := c.QueryParam("format")
		if format == "" {
			format = "rss"
		}
		result, err := a.generateFeed(name, format, parameters, c.QueryParams())
		if err != nil {
			c.Logger().Errorf("failed to generate: name=%s, err=%s", name, err)
			return err
		}
		return c.Blob(200, result.ContentType, []byte(result.Result))
	}
}

func (a *App) watchConfigFile(onChange func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	watcher.Add(a.configFile)

	for event := range watcher.Events {
		if event.Op&fsnotify.Write == fsnotify.Write {
			onChange()
		}
	}
	return nil
}

func (a *App) Run(args []string) error {
	return a.app.Run(args)
}
