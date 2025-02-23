//go:build live_public

package grpc_json_sniffer

import (
	"io/fs"
	"os"
)

func getStaticFiles() fs.FS {
	return os.DirFS("./public")
}
