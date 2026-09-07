package main

import (
	"embed"

	"winvncgo/tasks"
)

//go:embed frontend
var assets embed.FS

func main() {
	tasks.Run(assets)
}
