package main

import (
	"flag"
	"fmt"
	"os"

	sniffer "github.com/tsaarni/grpc-json-sniffer"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "Address to serve the web viewer")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -addr <address> <path>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	messagesFile := flag.Args()[0]

	if *addr == "" || messagesFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(messagesFile); os.IsNotExist(err) {
		fmt.Printf("No such file: %s\n", messagesFile)
		os.Exit(1)
	}

	viewer := sniffer.NewGrpcWebViewer(*addr, messagesFile)
	fmt.Printf("Starting gRPC JSON sniffer viewer on %s\n", *addr)
	viewer.Serve()
}
