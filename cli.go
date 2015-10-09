package gel

import (
	"github.com/codegangsta/cli"
	"os"
)

func main() {

	app := cli.NewApp()
	app.Name = "gel"
	app.Usage = "SQL preprocessor"
	app.EnableBashCompletion = true

	app.Commands = []cli.Command{
		{
			Name:    "compile",
			Aliases: []string{"c"},
			Usage:   "Compiles source-files",
			Action: func(c *cli.Context) {
			},
		},
	}
	app.Run(os.Args)

}
