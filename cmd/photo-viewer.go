package main

import (
	"flag"
	"os"

	"github.com/gookit/slog"
	"github.com/henrylau/zip-photo-viewer/internal/loader"
	"github.com/henrylau/zip-photo-viewer/internal/viewer"
)

func main() {
	slog.Configure(func(logger *slog.SugaredLogger) {
		f := logger.Formatter.(*slog.TextFormatter)
		f.EnableColor = true
	})

	flag.Parse()
	file := flag.Arg(0)

	if file == "" {
		slog.Fatal("Zip / Image file not provided")
		os.Exit(1)
	}

	fileLoader, err := loader.NewLoader(file)
	if err != nil {
		slog.Fatal(err)
		os.Exit(1)
	}

	viewer := viewer.NewViewer(fileLoader)
	viewer.Main()
}
