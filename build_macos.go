//go:build ignore

// Run from the project root: go run build_macos.go [-arch amd64|arm64|all].
// This helper runs on Windows, Linux and macOS; the game itself starts at main.go.
package main

import (
	game "battlecity/internal/runtime"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	arch := flag.String("arch", "all", "macOS architecture: amd64, arm64 or all")
	flag.Parse()
	arches := []string{*arch}
	if *arch == "all" {
		arches = []string{"amd64", "arm64"}
	}
	if flag.NArg() != 0 || (*arch != "all" && *arch != "amd64" && *arch != "arm64") {
		flag.Usage()
		os.Exit(2)
	}
	if err := build(arches); err != nil {
		fmt.Fprintln(os.Stderr, "macOS build:", err)
		os.Exit(1)
	}
}

func build(arches []string) error {
	if _, err := os.Stat("main.go"); err != nil {
		return fmt.Errorf("run this command from the project root: %w", err)
	}
	icon, err := os.ReadFile("assets/icon.icns")
	if err != nil {
		return err
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		return err
	}
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}
	for _, arch := range arches {
		if err := bundle(goTool, arch, icon); err != nil {
			return err
		}
	}
	return nil
}

func bundle(goTool, arch string, icon []byte) error {
	stage, err := os.MkdirTemp("bin", ".macos-"+arch+"-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	app := filepath.Join(stage, "BattleCityMD.app")
	contents := filepath.Join(app, "Contents")
	for _, dir := range []string{"MacOS", "Resources"} {
		if err := os.MkdirAll(filepath.Join(contents, dir), 0755); err != nil {
			return err
		}
	}
	cmd := exec.Command(goTool, "build", "-buildvcs=false", "-trimpath", "-ldflags=-s -w",
		"-o", filepath.Join(contents, "MacOS", "BattleCityMD"), ".")
	// Replace target settings rather than appending duplicate environment keys.
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch strings.ToUpper(name) {
		case "GOOS", "GOARCH", "CGO_ENABLED":
		default:
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "GOOS=darwin", "GOARCH="+arch, "CGO_ENABLED=0")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	fmt.Println("Building macOS/" + arch)
	if err := cmd.Run(); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Join(contents, "MacOS", "BattleCityMD"), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contents, "Resources", "icon.icns"), icon, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(fmt.Sprintf(plist, game.Version, game.Version)), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contents, "PkgInfo"), []byte("APPL????"), 0644); err != nil {
		return err
	}
	// Local Mac builds receive an ad-hoc signature, without a developer account.
	// Cross-builds retain the Go linker's executable signature where applicable.
	if runtime.GOOS == "darwin" {
		if codesign, err := exec.LookPath("codesign"); err == nil {
			cmd := exec.Command(codesign, "--force", "--sign", "-", "--timestamp=none", app)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("ad-hoc signing: %w", err)
			}
		}
	}
	dest := filepath.Join("bin", "BattleCityMD-macos-"+arch+".app")
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.Rename(app, dest); err != nil {
		return err
	}
	fmt.Println("Created " + dest)
	return nil
}

const plist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>Battle City</string>
    <key>CFBundleDisplayName</key><string>Battle City</string>
    <key>CFBundleIdentifier</key><string>game.battlecitymd.go</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleExecutable</key><string>BattleCityMD</string>
    <key>CFBundleIconFile</key><string>icon.icns</string>
    <key>CFBundleShortVersionString</key><string>%s</string>
    <key>CFBundleVersion</key><string>%s</string>
    <key>LSMinimumSystemVersion</key><string>12.0</string>
    <key>LSApplicationCategoryType</key><string>public.app-category.arcade-games</string>
    <key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
`
