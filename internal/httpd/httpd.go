package httpd

import (
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"strings"
)

type Httpd struct {
	net.Listener

	r fs.FS // the directory to root
	Addr string // the address to listen to
}

func NewHTTPd(addr, root string) (*Httpd, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	h := &Httpd{
		Listener: l,
		Addr: addr,
		r: os.DirFS(root),
	}

	return h, nil
}

func (h *Httpd) Start() error {
	for {
		conn, err := h.Accept()
		if err != nil {
			return err
		}

		go h.handle(conn)
	}
}

func (h *Httpd) handle(c net.Conn) {
	defer c.Close()

	buf := make([]byte, 1024)

	rlen, err := c.Read(buf)
	if err != nil {
		log.Print(err)
		return
	}

	if buf[rlen-1] != 0x0a {
		log.Printf("request did not end with LF (is %d)", buf[rlen-1])
		return
	}
	reqStr := string(buf[:rlen])
	tokens := strings.Split(strings.TrimSuffix(reqStr, "\x0a"), " ")
	if len(tokens) < 2 {
		log.Printf("bad request %s", reqStr)
		c.Write([]byte("HTTP 400 Bad Request\x0a"))
		return
	}

	verb, rPath := tokens[0], tokens[1]

	switch verb {
	case "GET":
		log.Printf("serving %s", rPath)
		err := h.serveGet(c, rPath)
		if err != nil {
			log.Printf("serveGet: %s", err)
		}
	default:
		log.Printf("Unknown verb %s in %s", verb, reqStr)
		return
	}
}

func (h *Httpd) serveGet(c net.Conn, rPath string) error {
	rPath = strings.TrimPrefix(rPath, "/")
	if rPath == "" {
		rPath = "index.html"
	}
	f, err := h.r.Open(rPath)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(c, f)

	if err != nil {
		return err
	}

	return nil
}
