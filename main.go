package main

import (
	"os"

	"github.com/uphy/feedgen/app"
)

func main() {
	a := app.New()
	if err := a.Run(os.Args); err != nil {
		panic(err)
	}
}
