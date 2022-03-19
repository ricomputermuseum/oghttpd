package httpd

import (
	"fmt"
	"html/template"
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

type FileResponse struct {
	fs.File
}

func (fr *FileResponse) WriteTo(w io.Writer) (int, error) {
	sz, err := io.Copy(w, fr.File)

	if err != nil {
		log.Printf("file write: %s", err.Error())
	}

	return int(sz), err
}

type DirResponse struct {
	fs.File
	path string
}

var DirTemplate = template.Must(template.New("dir").Parse(`<HTML>
<TITLE>Index of {{ .Path }}</TITLE>
<UL>
{{ range .Files }}<LI><A HREF="{{.Name}}">{{.Name}}{{if .IsDir }}/{{end}}</A></LI>{{end}}
</UL>
</HTML>`))

type dirListing struct {
	Path string
	Files []fs.DirEntry
}

func (dr *DirResponse) WriteTo(w io.Writer) (int, error) {
	var err error
	var written int
	if dir, isDir := dr.File.(fs.ReadDirFile); isDir {
		ents, err := dir.ReadDir(-1)
		if err != nil {
			er := &ErrorResponse{
				Err: err,
			}
			return er.WriteTo(w)
		}

		dl := dirListing{
			Path: dr.path,
			Files: ents,
		}

		return -1, DirTemplate.Execute(w, dl)
	} else {
		panic("is not dir!")
	}
	return written, err
}

type ErrorResponse struct {
	Path string
	Err error
}

func (er *ErrorResponse) Close() error { return nil }

func (er *ErrorResponse) WriteTo(w io.Writer) (int, error) {
	if er.Err == nil {
		return fmt.Fprintf(w, "Not found: %s", er.Path)
	} else {
		return fmt.Fprintf(w, "Error: %s", er.Err.Error())
	}
}

type Response interface {
	WriteTo(io.Writer) (int, error)
	io.Closer
}

func (h *Httpd) makeRequest(p string) Response {
	rPath := strings.TrimLeft(p, "/")
	if rPath == "" || strings.HasSuffix(rPath, "/") {
		f, err := h.r.Open(rPath + "index.html")
		if err == nil {
			return &FileResponse{
				File: f,
			}
		}
	}

	f, err := h.r.Open(rPath)
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			return &ErrorResponse{
				Path: p,
			}
		}

		return &ErrorResponse{
			Path: p,
			Err: err,
		}
	}

	fsi, err := f.Stat()
	if err != nil {
		return &ErrorResponse{
			Path: p,
			Err: err,
		}
	} else {
		if fsi.IsDir() {
			return &DirResponse{
				File: f,
			}
		} else {
			return &FileResponse{
				File: f,
			}
		}
	}
}

func (h *Httpd) serveGet(c net.Conn, oPath string) error {
	r := h.makeRequest(oPath)
	if r != nil {
		defer r.Close()
		written, err := r.WriteTo(c)
		log.Printf("wrote %d bytes", written)
		return err
	}

	return nil
}
