package main

import (
  "github.com/amaniak/gel"
)

func main() {

  parser := gel.SQLParser("./")
  parser.Parse()

}
