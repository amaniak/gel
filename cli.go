package gel

import (
  "os"
  "github.com/codegangsta/cli"
)

func main() {

  app := cli.NewApp()
  app.Name = "gel"
  app.Usage = "SQL preprocessor"
  app.EnableBashCompletion = true

  app.Commands = []cli.Command {
    {
      Name: "compile",
      Aliases: []string{"c"},
      Usage: "Compiles source-files",
      Action: func(c *cli.Context) {
      },
    },
  }
  app.Run(os.Args)

}
