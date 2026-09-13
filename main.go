package main

import (
	"embed"
	"log"
	"time"

	"github.com/buchfink/buchfink/internal/wailsbridge"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Wails uses Go's embed package to embed the frontend files into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	bridge, err := wailsbridge.NewBuchfinkBridge()
	if err != nil {
		log.Fatalf("Fehler beim Initialisieren des Buchfink-Service: %v", err)
	}

	app := application.New(application.Options{
		Name:        "Buchfink",
		Description: "Open-Source-Buchhaltung für kleine Unternehmen (SKR04, GoBD-konform, E-Bilanz)",
		Services: []application.Service{
			application.NewService(bridge),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		EnableFileDrop: true,
		Title:          "Buchfink — Buchhaltung",
		Width:          1280,
		Height:         820,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 48,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(250, 249, 246),
		URL:              "/",
	})

	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		target := event.Context().DropTargetDetails()
		if target == nil || target.ElementID == "" {
			return
		}
		window.DispatchWailsEvent(&application.CustomEvent{
			Name: "buchfink:files-dropped",
			Data: map[string]any{"targetId": target.ElementID, "paths": event.Context().DroppedFiles()},
		})
	})

	go func() {
		for {
			now := time.Now().Format("15:04:05")
			app.Event.Emit("tick", now)
			time.Sleep(10 * time.Second)
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
