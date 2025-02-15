//go:build live_public

package grpc_json_sniffer

import (
	"io/fs"
	"os"
)

func GetStaticFiles() fs.FS {
	return os.DirFS("./public")
}
