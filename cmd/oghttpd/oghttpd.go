package main

import (
	"flag"

	"github.com/ricomputermuseum/oghttpd/internal/httpd"
)

func main() {
	var wwwRoot string
	flag.StringVar(&wwwRoot, "root", "./pub", "WWW root")
	flag.Parse()

	h, err := httpd.NewHTTPd(":80", wwwRoot)
	if err != nil {
		panic(err)
	}

	defer h.Close()

	err = h.Start()
	if err != nil {
		panic(err)
	}
}
