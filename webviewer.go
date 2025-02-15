package grpc_json_sniffer

import (
	"bufio"
	"context"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"
)

type GrpcWebViewer struct {
	publicFiles fs.FS
	addr        string
	messages    string
}

func NewGrpcWebViewer(addr string, messages string) *GrpcWebViewer {
	return &GrpcWebViewer{
		addr:        addr,
		messages:    messages,
		publicFiles: GetStaticFiles(),
	}
}

func (v *GrpcWebViewer) Serve() {
	server := &http.Server{
		Addr:              v.addr,
		ReadHeaderTimeout: time.Duration(5) * time.Second,
		Handler:           v,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func (v *GrpcWebViewer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/messages") {
		v.messagesHandler(w, r)
		return
	}

	v.filesHandler(w, r)
}

func (v *GrpcWebViewer) messagesHandler(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer c.CloseNow()

	messagesFile, err := os.OpenFile(v.messages, os.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer messagesFile.Close()

	ctx := r.Context()
	messages := fileReader(ctx, messagesFile) // Launch file reader goroutine.

	// Read messages sent by the file reader and write them to the websocket.
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return
			}
			if err := c.Write(ctx, websocket.MessageText, []byte(msg)); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (v *GrpcWebViewer) filesHandler(w http.ResponseWriter, r *http.Request) {
	relativePath := strings.TrimPrefix(r.URL.Path, "/")
	if relativePath == "" {
		relativePath = "index.html"
	}

	file, err := v.publicFiles.Open(relativePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	ext := filepath.Ext(relativePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	_, _ = io.Copy(w, file)

}

func fileReader(ctx context.Context, file *os.File) <-chan string {
	reader := bufio.NewReader(file)
	messages := make(chan string)
	go func() {
		defer close(messages)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := readline(reader)
				if err != nil {
					return
				}
				select {
				case messages <- line:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return messages
}

func readline(reader *bufio.Reader) (string, error) {
	for {
		line, err := reader.ReadString('\n')
		if err == nil {
			return line, nil
		}
		if err != io.EOF {
			return "", err
		}

		time.Sleep(100 * time.Millisecond)
	}
}
