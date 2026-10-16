package ui

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// nativeFolderPick asks the OS for a directory using the native portal
// picker (kdialog on KDE, zenity elsewhere) instead of Fyne's custom dialog.
//
// Returns (dir, chosen, available, err):
//   - chosen=false when the user cancelled
//   - available=false when no native picker binary exists (caller should
//     fall back to the Fyne dialog)
func nativeFolderPick(title, startDir string) (dir string, chosen, available bool, err error) {
	if p, lerr := exec.LookPath("kdialog"); lerr == nil {
		out, cerr := exec.Command(p, "--title", title, "--getexistingdirectory", startDir).Output()
		if cerr != nil {
			return "", false, true, cerr // user cancelled (or kdialog failed): treat as cancel
		}
		dir = strings.TrimSpace(string(out))
		if dir == "" {
			return "", false, true, nil
		}
		return dir, true, true, nil
	}
	if p, lerr := exec.LookPath("zenity"); lerr == nil {
		out, cerr := exec.Command(p, "--file-selection", "--directory",
			"--title", title, "--filename", filepath.Join(startDir, "")).Output()
		if cerr != nil {
			return "", false, true, cerr
		}
		dir = strings.TrimSpace(string(out))
		if dir == "" {
			return "", false, true, nil
		}
		return dir, true, true, nil
	}
	return "", false, false, nil
}

// nativeFilePick asks the OS for a file using the native picker (kdialog on
// KDE, zenity elsewhere). filter is a space separated glob list such as
// "*.gif *.GIF". Returns (path, chosen, available, err).
func nativeFilePick(title, startDir, filter string) (path string, chosen, available bool, err error) {
	if p, lerr := exec.LookPath("kdialog"); lerr == nil {
		out, cerr := exec.Command(p, "--title", title, "--getopenfilename", startDir, filter+"|GIF images").Output()
		if cerr != nil {
			return "", false, true, cerr
		}
		path = strings.TrimSpace(string(out))
		if path == "" {
			return "", false, true, nil
		}
		return path, true, true, nil
	}
	if p, lerr := exec.LookPath("zenity"); lerr == nil {
		out, cerr := exec.Command(p, "--file-selection",
			"--title", title,
			"--file-filter", "GIF images | "+filter,
			"--filename", filepath.Join(startDir, "")).Output()
		if cerr != nil {
			return "", false, true, cerr
		}
		path = strings.TrimSpace(string(out))
		if path == "" {
			return "", false, true, nil
		}
		return path, true, true, nil
	}
	return "", false, false, nil
}
