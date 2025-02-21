//go:build live_public

package sniffer

import (
	"io/fs"
	"os"
)

func getStaticFiles() fs.FS {
	return os.DirFS("./public")
}
