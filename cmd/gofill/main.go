package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/crawshaw/gofill"
)

var (
	httpAddr = flag.String("http", "localhost:6060", "HTTP service address")
	verbose  = flag.Bool("v", false, "verbose mode")
	//fs       = vfs.NameSpace{}
)

func main() {
	flag.Parse()

	log.Printf("gofill service")

	http.Handle("/gofill", gofill.SimpleHandler())

	startTime := time.Now()
	for name, content := range gofill.StaticFiles {
		name := "/"+name
		data := bytes.NewReader([]byte(content))
		http.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
			http.ServeContent(w, r, name, startTime, data)
		})
	}

	if err := http.ListenAndServe(*httpAddr, http.DefaultServeMux); err != nil {
		log.Fatalf("ListenAndServe %s: %v", *httpAddr, err)
	}
}
