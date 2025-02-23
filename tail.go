package grpc_json_sniffer

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"
)

func tailFile(ctx context.Context, file *os.File, messages chan string) {
	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := readline(reader)
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return
			}
			select {
			case messages <- line:
			case <-ctx.Done():
				return
			}
		}
	}
}

func readline(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line, nil
}
