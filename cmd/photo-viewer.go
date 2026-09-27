package main

import (
	"flag"
	"os"

	"github.com/gookit/slog"
	"github.com/henrylau/zip-photo-viewer/internal/loader"
	"github.com/henrylau/zip-photo-viewer/internal/viewer"
)

func main() {
	var debug bool
	var password string
	slog.Configure(func(logger *slog.SugaredLogger) {
		f := logger.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
		logger.Level = slog.InfoLevel
	})

	flag.BoolVar(&debug, "debug", false, "Enable debug log")
	flag.StringVar(&password, "password", "", "Password for encrypted 7z/RAR archives")
	flag.Parse()
	file := flag.Arg(0)

	if debug {
		slog.SetLogLevel(slog.DebugLevel)
	}

	if file == "" {
		slog.Fatal("Folder / archive / image file not provided")
		os.Exit(1)
	}

	fileLoader, err := loader.NewLoaderWithPassword(file, password)
	if err != nil {
		if loader.IsPasswordError(err) {
			v := viewer.NewViewer(nil, file)
			v.AskPassword(file, err)
			v.Main()
			return
		}
		slog.Fatal(err)
		os.Exit(1)
	}

	v := viewer.NewViewer(fileLoader, file)
	v.Main()
}
