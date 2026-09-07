package tasks

import (
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Run 启动 Wails 应用，assets 由调用方提供
func Run(assets fs.FS) {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "OneVNC",
		Width:         420,
		Height:        570,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal("Error: ", err.Error())
	}
}
