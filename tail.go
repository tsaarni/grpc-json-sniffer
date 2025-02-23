package grpc_json_sniffer

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"
)

func tailFile(ctx context.Context, file *os.File, lines chan string) {
	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return
			}
			lines <- line
		}
	}
}
