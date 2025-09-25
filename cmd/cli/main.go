package main

import (
	"coupongo/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
