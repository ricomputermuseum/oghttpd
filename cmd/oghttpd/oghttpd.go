package main

import (
	"github.com/ricomputermuseum/oghttpd/internal/httpd"
)

func main() {
	h, err := httpd.NewHTTPd(":80", "./pub")
	if err != nil {
		panic(err)
	}

	defer h.Close()

	err = h.Start()
	if err != nil {
		panic(err)
	}
}
