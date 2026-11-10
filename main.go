package main

import (
	"embed"
	"log"
	"sync/atomic"

	"github.com/buchfink/buchfink/internal/wailsbridge"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's embed package to embed the frontend files into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Eine zweite Instanz holt die erste nach vorn und endet, bevor sie einen
	// Mandanten öffnet: ein Mandant hat genau einen Schreiber.
	var window atomic.Pointer[application.WebviewWindow]
	app := application.New(application.Options{
		Name:        "Buchfink",
		Description: "Open-Source-Buchhaltung für GmbH und UG (SKR04, GoBD-konform, E-Bilanz)",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.buchfink.app",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if w := window.Load(); w != nil {
					w.Show()
					w.Focus()
				}
			},
		},
	})

	bridge, err := wailsbridge.NewBuchfinkBridge()
	if err != nil {
		log.Fatalf("Fehler beim Initialisieren des Buchfink-Service: %v", err)
	}
	app.RegisterService(application.NewService(bridge))

	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Buchfink — Buchhaltung",
		Width:  1280,
		Height: 820,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 48,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(250, 249, 246),
		URL:              "/",
		// Dateien, die auf eine Ablagezone gezogen werden, kommen in den
		// Eingang (wailsbridge.AttachFileDrop).
		EnableFileDrop: true,
	})
	window.Store(w)
	wailsbridge.AttachFileDrop(w, bridge)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
