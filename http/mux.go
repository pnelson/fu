package http

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/pnelson/fu/api"
)

// mux is a HTTP multiplexer.
type mux struct {
	core *api.Core
}

// Serve listens on the configured TCP address and dispatches
// incoming HTTP requests to the application request multiplexer.
func Serve(config api.Config) error {
	core, err := api.NewCore(config)
	if err != nil {
		return err
	}
	defer core.Close()
	go core.Run()
	return http.ListenAndServe(core.Config.Addr, &mux{core: core})
}

// ServeHTTP implements the http.Handler interface.
func (m *mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		m.get(w, req)
	case http.MethodPost:
		m.put(w, req)
	default:
		w.Header().Set("Allow", "GET, POST")
		abort(w, http.StatusMethodNotAllowed)
	}
}

func (m *mux) get(w http.ResponseWriter, req *http.Request) {
	name := req.URL.Path[1:]
	if name == "" {
		abort(w, http.StatusNotFound)
		return
	}
	name = filepath.Join(m.core.Config.UploadDir, name)
	http.ServeFile(w, req, name)
}

func (m *mux) put(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		abort(w, http.StatusNotFound)
		return
	}
	token := req.Header.Get("Authentication")
	if strings.HasPrefix(token, `fu token=`) {
		token = token[9:]
	}
	if subtle.ConstantTimeCompare(m.core.Config.Token, []byte(token)) != 1 {
		abort(w, http.StatusForbidden)
		return
	}
	err := req.ParseMultipartForm(m.core.Config.MaxUploadSize)
	if err != nil {
		resolve(w, err)
		return
	}
	duration, err := time.ParseDuration(strings.TrimSpace(req.FormValue("duration")))
	if err != nil {
		duration = time.Hour
	}
	file, h, err := req.FormFile("file")
	if err != nil {
		resolve(w, err)
		return
	}
	defer file.Close()
	form := api.UploadForm{
		Duration: duration,
		File:     file,
		Ext:      filepath.Ext(h.Filename),
	}
	f, err := m.core.Put(form)
	if err != nil {
		resolve(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(f)
	if err != nil {
		log.Println(err)
	}
}

func abort(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

func resolve(w http.ResponseWriter, err error) {
	switch err {
	case http.ErrNotMultipart:
		abort(w, http.StatusBadRequest)
	default:
		log.Println(err)
		abort(w, http.StatusInternalServerError)
	}
}
