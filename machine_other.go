//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const onAndroid = false

func platformMachineRaw() string {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(p); err == nil && len(strings.TrimSpace(string(b))) > 0 { return "nix|" + strings.TrimSpace(string(b)) }
	}
	h, _ := os.Hostname()
	return "host|" + h
}

func trialMarkPath() string {
	d, err := os.UserConfigDir()
	if err != nil { return "" }
	return filepath.Join(d, "NamLinhGroup", "digitalshop-t0")
}

func trialMarkGet() int64 {
	p := trialMarkPath()
	if p == "" { return 0 }
	b, _ := os.ReadFile(p)
	n, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	return n
}

func trialMarkSet(t int64) {
	p := trialMarkPath()
	if p == "" { return }
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(strconv.FormatInt(t, 10)), 0o600)
}
