//go:build !live_public

package sniffer

import (
	"embed"
	"io/fs"
)

//go:embed public/*
var staticFiles embed.FS

func getStaticFiles() fs.FS {
	staticFiles, err := fs.Sub(staticFiles, "public")
	if err != nil {
		panic(err)
	}
	return staticFiles
}
