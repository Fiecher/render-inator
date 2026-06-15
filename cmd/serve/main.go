package main

import (
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	_ = mime.AddExtensionType(".wasm", "application/wasm")
	dir := defaultDocsDir()
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	addr := "127.0.0.1:8080"
	if len(os.Args) > 2 {
		addr = os.Args[2]
	}
	log.Printf("serving %s at http://%s", dir, addr)
	log.Fatal(http.ListenAndServe(addr, http.FileServer(http.Dir(dir))))
}

func defaultDocsDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "docs"
	}
	return filepath.Join(filepath.Dir(exe), "..", "..", "docs")
}
