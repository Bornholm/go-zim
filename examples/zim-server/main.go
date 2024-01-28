package main

import (
	"flag"
	"net/http"

	"github.com/Bornholm/go-zim"
	zimFS "github.com/Bornholm/go-zim/fs"
)

var (
	zimPath  string
	httpAddr = ":8080"
)

func init() {
	flag.StringVar(&zimPath, "zim", zimPath, "zim file path")
	flag.StringVar(&httpAddr, "addr", httpAddr, "http server address")
}

func main() {
	flag.Parse()

	reader, err := zim.Open(zimPath)
	if err != nil {
		panic(err)
	}

	fs := zimFS.NewFS(reader)
	fileServer := http.FileServer(http.FS(fs))

	if err := http.ListenAndServe(httpAddr, fileServer); err != nil {
		panic(err)
	}
}
