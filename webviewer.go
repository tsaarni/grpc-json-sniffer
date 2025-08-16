package grpc_json_sniffer

import (
	"context"
	"fmt"
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
		publicFiles: getStaticFiles(),
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
	sock, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer sock.CloseNow() // nolint:errcheck

	messagesFile, err := os.OpenFile(v.messages, os.O_RDONLY, 0)
	if err != nil {
		sock.Close(websocket.StatusInternalError, "Capture messages file not found") //nolint:errcheck
		return
	}
	defer messagesFile.Close() //nolint:errcheck

	tailCtx, cancelTail := context.WithCancel(context.Background())
	defer cancelTail()
	messages := make(chan string)
	go tailFile(tailCtx, messagesFile, messages)

	// The web client will never write anything to the socket.
	// If read returns, the client has disconnected.
	go func() {
		_, _, _ = sock.Reader(r.Context())
		cancelTail()
	}()

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				fmt.Println("Messages channel closed")
				return
			}
			if err := sock.Write(tailCtx, websocket.MessageText, []byte(msg)); err != nil {
				return
			}
		case <-tailCtx.Done():
			sock.Close(websocket.StatusInternalError, "Cannot read captured messages from file") //nolint:errcheck
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
	defer file.Close() //nolint:errcheck

	ext := filepath.Ext(relativePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	_, _ = io.Copy(w, file)

}
