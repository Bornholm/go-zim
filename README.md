# go-zim

[![Go Reference](https://pkg.go.dev/badge/github.com/Bornholm/go-zim.svg)](https://pkg.go.dev/github.com/Bornholm/go-zim)

A Golang library to read and serve [ZIM](https://wiki.openzim.org/wiki/OpenZIM) archives.

Inspired by [`github.com/tim-st/go-zim`](https://pkg.go.dev/github.com/tim-st/go-zim).

## Usage

### Reading a ZIM file

```go
package main

import (
	"flag"
	"net/http"

	"github.com/Bornholm/go-zim"
)

func main() {
	archive, err := zim.Open("my-archive.zim")
	if err != nil {
		panic(err)
	}

	defer func() {
        if err := archive.Close(); err != nil {
            panic(err)
        }
    }()

    fmt.Println("Entries count:", archive.EntryCount())
    
    mainPage, err := archive.MainPage()
    if err != nil {
        panic(err)
    }

    fmt.Println("Main Page Title:", mainPage.Title())
    fmt.Println("Main Page Full URL:", mainPage.FullURL())
}
```

### Serving a ZIM file with a HTTP server

```go
package main

import (
	"flag"
	"net/http"

	"github.com/Bornholm/go-zim"
	zimFS "github.com/Bornholm/go-zim/fs"
)

func main() {
	reader, err := zim.Open(zimPath)
	if err != nil {
		panic(err)
	}

	fs := zimFS.NewFS(reader)
	fileServer := http.FileServer(http.FS(fs))

	if err := http.ListenAndServe(":8080", fileServer); err != nil {
		panic(err)
	}
}
```

See [`examples/zim-server`](./examples/zim-server) for an runnable example.

## License

[MIT](./LICENSE)

