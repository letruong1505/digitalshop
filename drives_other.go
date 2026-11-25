//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

// listDrives (Linux/macOS, dùng khi thử nghiệm): ổ gắn ở /media, /run/media, /Volumes.
func listDrives() []Drive {
	out := []Drive{}
	for _, base := range []string{"/media", "/run/media", "/Volumes"} {
		ents, _ := os.ReadDir(base)
		for _, e := range ents {
			p := filepath.Join(base, e.Name())
			sub, _ := os.ReadDir(p)
			if base != "/Volumes" && len(sub) > 0 {
				for _, s := range sub { if s.IsDir() { out = append(out, Drive{Path: filepath.Join(p, s.Name()), Label: s.Name(), Kind: "usb"}) } }
				continue
			}
			out = append(out, Drive{Path: p, Label: e.Name(), Kind: "usb"})
		}
	}
	return append(out, cloudFolders()...)
}
