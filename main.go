package main

import (
	"embed"

	"wegovnc/tasks"
)

//go:embed frontend
var assets embed.FS

func main() {
	tasks.Run(assets)
}
