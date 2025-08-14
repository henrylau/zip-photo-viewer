package main

import (
	"flag"

	"github.com/henrylau/zip-photo-viewer/internal/loader"
	"github.com/henrylau/zip-photo-viewer/internal/viewer"
)

func main() {
	flag.Parse()
	file := flag.Arg(0)
	if file == "" {
		panic("Zip file not provided")
	}

	fileLoader, err := loader.NewLoader(file)
	if err != nil {
		panic(err)
	}

	viewer := viewer.NewViewer(fileLoader)
	viewer.Main()
}
