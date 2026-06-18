package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !presetNameRE.MatchString(name) {
		httpErr(w, errors.New("bad preset name"), 400)
		return
	}
	entries, err := os.ReadDir(filepath.Join(blanket, "wallpapers", name))
	if err != nil {
		httpErr(w, err, 404)
		return
	}
	type file struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	out := []file{}
	for _, e := range entries {
		if e.IsDir() || !ssNameRE.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, file{e.Name(), info.Size()})
	}
	writeJSON(w, out)
}

func handleGetFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	file := r.PathValue("file")
	if !presetNameRE.MatchString(name) || !ssNameRE.MatchString(file) {
		httpErr(w, errors.New("bad name"), 400)
		return
	}
	http.ServeFile(w, r, filepath.Join(blanket, "wallpapers", name, file))
}

func handleUploadFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !presetNameRE.MatchString(name) {
		httpErr(w, errors.New("bad preset name"), 400)
		return
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		httpErr(w, err, 400)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpErr(w, err, 400)
		return
	}
	defer file.Close()
	fname := filepath.Base(header.Filename)
	if !ssNameRE.MatchString(fname) {
		httpErr(w, fmt.Errorf("bad filename: %s", fname), 400)
		return
	}
	err = withRW(func() error {
		root, err := os.OpenRoot(blanket)
		if err != nil {
			return err
		}
		defer root.Close()
		out, err := root.Create("wallpapers/" + name + "/" + fname)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, file)
		return err
	})
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(201)
}

func handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	file := r.PathValue("file")
	if !presetNameRE.MatchString(name) || !ssNameRE.MatchString(file) {
		httpErr(w, errors.New("bad name"), 400)
		return
	}
	err := withRW(func() error {
		root, err := os.OpenRoot(blanket)
		if err != nil {
			return err
		}
		defer root.Close()
		return root.Remove("wallpapers/" + name + "/" + file)
	})
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

// handleImport copies bg_ssNN.png files from a user-supplied absolute path
// into a new preset. Renumbers contiguously from 00 in sort order; the source
// is left untouched.
func handleImport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpErr(w, err, 400)
		return
	}
	src := filepath.Clean(strings.TrimSpace(body.Path))
	name := strings.TrimSpace(body.Name)
	if !presetNameRE.MatchString(name) {
		httpErr(w, errors.New("bad preset name"), 400)
		return
	}
	if !filepath.IsAbs(src) {
		httpErr(w, errors.New("path must be absolute"), 400)
		return
	}
	fi, err := os.Stat(src)
	if err != nil {
		httpErr(w, err, 404)
		return
	}
	if !fi.IsDir() {
		httpErr(w, errors.New("not a directory"), 400)
		return
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	var pngs []string
	for _, e := range entries {
		if !e.IsDir() && ssNameRE.MatchString(e.Name()) {
			pngs = append(pngs, e.Name())
		}
	}
	if len(pngs) == 0 {
		httpErr(w, errors.New("no bg_ssNN.png files in source"), 400)
		return
	}
	sort.Strings(pngs)

	err = withRW(func() error {
		root, err := os.OpenRoot(blanket)
		if err != nil {
			return err
		}
		defer root.Close()
		if _, err := root.Stat("wallpapers/" + name); err == nil {
			return fmt.Errorf("preset %q already exists", name)
		}
		if err := root.Mkdir("wallpapers/"+name, 0755); err != nil {
			return err
		}
		for i, fname := range pngs {
			dst := fmt.Sprintf("wallpapers/%s/bg_ss%02d.png", name, i)
			if err := copyInto(root, filepath.Join(src, fname), dst); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(201)
}

func copyInto(root *os.Root, srcPath, dstRel string) error {
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := root.Create(dstRel)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
