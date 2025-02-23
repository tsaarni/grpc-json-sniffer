package grpc_json_sniffer

import (
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
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer messagesFile.Close()

	ctx := r.Context()
	messages := make(chan string)
	go tailFile(ctx, messagesFile, messages)

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return
			}
			if err := sock.Write(ctx, websocket.MessageText, []byte(msg)); err != nil {
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
