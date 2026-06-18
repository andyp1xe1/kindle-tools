package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
)

func handleClaim(w http.ResponseWriter, r *http.Request) {
	name, err := decodeName(r, "name")
	if err != nil {
		httpErr(w, err, 400)
		return
	}
	if err := withRW(func() error { return claimScreensaver(name) }); err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(201)
}

// claimScreensaver moves the contents of a regular screensaver/ dir into a new
// preset and replaces the dir with a symlink to it. Idempotent only on
// failure-then-retry: refuses if screensaver/ is already a symlink.
func claimScreensaver(name string) error {
	root, err := os.OpenRoot(blanket)
	if err != nil {
		return err
	}
	defer root.Close()

	fi, err := root.Lstat("screensaver")
	if err != nil {
		return fmt.Errorf("no screensaver/ to claim: %w", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return errors.New("screensaver/ is already a symlink")
	}
	if !fi.IsDir() {
		return errors.New("screensaver/ is not a directory")
	}
	if _, err := root.Stat("wallpapers/" + name); err == nil {
		return fmt.Errorf("preset %q already exists", name)
	}
	if err := root.Mkdir("wallpapers/"+name, 0755); err != nil {
		return err
	}

	f, err := root.Open("screensaver")
	if err != nil {
		return err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := root.Rename("screensaver/"+e.Name(), "wallpapers/"+name+"/"+e.Name()); err != nil {
			return err
		}
	}
	if err := root.Remove("screensaver"); err != nil {
		return err
	}
	return root.Symlink("wallpapers/"+name, "screensaver")
}

func handleCreatePreset(w http.ResponseWriter, r *http.Request) {
	name, err := decodeName(r, "name")
	if err != nil {
		httpErr(w, err, 400)
		return
	}
	err = withRW(func() error {
		root, err := os.OpenRoot(blanket)
		if err != nil {
			return err
		}
		defer root.Close()
		return root.Mkdir("wallpapers/"+name, 0755)
	})
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(201)
}

func handleDeletePreset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !presetNameRE.MatchString(name) {
		httpErr(w, errors.New("bad preset name"), 400)
		return
	}
	err := withRW(func() error {
		root, err := os.OpenRoot(blanket)
		if err != nil {
			return err
		}
		defer root.Close()
		if isActive, _ := activePresetIs(root, name); isActive {
			return errors.New("preset is active — activate another first")
		}
		return root.RemoveAll("wallpapers/" + name)
	})
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

func handleActivatePreset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !presetNameRE.MatchString(name) {
		httpErr(w, errors.New("bad preset name"), 400)
		return
	}
	if err := withRW(func() error { return activatePreset(name) }); err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

// activatePreset points screensaver/ at wallpapers/<name>. Replacing an
// existing symlink is atomic (tmp+rename). A regular dir at screensaver/ is
// rejected unless empty — otherwise activation would silently shadow files.
func activatePreset(name string) error {
	root, err := os.OpenRoot(blanket)
	if err != nil {
		return err
	}
	defer root.Close()
	if _, err := root.Stat("wallpapers/" + name); err != nil {
		return fmt.Errorf("no such preset: %w", err)
	}
	target := "wallpapers/" + name

	if fi, err := root.Lstat("screensaver"); err == nil && fi.Mode()&os.ModeSymlink == 0 && fi.IsDir() {
		f, err := root.Open("screensaver")
		if err != nil {
			return err
		}
		ents, err := f.ReadDir(-1)
		f.Close()
		if err != nil {
			return err
		}
		for _, e := range ents {
			if !e.IsDir() {
				return errors.New("screensaver/ has unclaimed files — claim or delete first")
			}
		}
		if err := root.Remove("screensaver"); err != nil {
			return err
		}
		return root.Symlink(target, "screensaver")
	}
	_ = root.Remove("screensaver.tmp")
	if err := root.Symlink(target, "screensaver.tmp"); err != nil {
		return err
	}
	return root.Rename("screensaver.tmp", "screensaver")
}

func handleRenamePreset(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !presetNameRE.MatchString(name) {
		httpErr(w, errors.New("bad preset name"), 400)
		return
	}
	to, err := decodeName(r, "to")
	if err != nil {
		httpErr(w, err, 400)
		return
	}
	if name == to {
		w.WriteHeader(204)
		return
	}
	err = withRW(func() error {
		root, err := os.OpenRoot(blanket)
		if err != nil {
			return err
		}
		defer root.Close()
		if _, err := root.Stat("wallpapers/" + to); err == nil {
			return fmt.Errorf("preset %q already exists", to)
		}
		wasActive, _ := activePresetIs(root, name)
		if err := root.Rename("wallpapers/"+name, "wallpapers/"+to); err != nil {
			return err
		}
		if wasActive {
			_ = root.Remove("screensaver.tmp")
			if err := root.Symlink("wallpapers/"+to, "screensaver.tmp"); err != nil {
				return err
			}
			return root.Rename("screensaver.tmp", "screensaver")
		}
		return nil
	})
	if err != nil {
		httpErr(w, err, 500)
		return
	}
	w.WriteHeader(204)
}

func activePresetIs(root *os.Root, name string) (bool, error) {
	fi, err := root.Lstat("screensaver")
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false, err
	}
	target, err := root.Readlink("screensaver")
	if err != nil {
		return false, err
	}
	return target == "wallpapers/"+name, nil
}
