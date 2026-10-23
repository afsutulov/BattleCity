<img src="assets/icon.png" alt="Battle City tank icon" width="96" align="right">

# Battle City

[Русская версия](README_RU.md)

Defend the eagle, destroy enemy tanks and fight through 35 classic stages.
Play solo, team up with a friend on the same computer, or build your own battlefield
in Construction mode. This desktop edition recreates the supplied Battle City
v1.03 with its original pixel art, maps and sounds.

**The game rules run in Go.** No Mega Drive emulator, cartridge program, ROM file,
68000/Z80 interpreter or console firmware is used or required at runtime.
Ebitengine provides the desktop window, input, graphics presentation and audio output.

## Screenshots

| Main menu | Stage selection |
|---|---|
| ![Main menu](screenshots/menu.png) | ![Stage selection](screenshots/stage-selection.png) |
| Single player | Local cooperative play |
| ![Single-player gameplay](screenshots/gameplay.png) | ![Two-player gameplay](screenshots/cooperative.png) |

## Features

- 35 original stages, solo play, local two-player mode and Construction mode.
- Native Go movement, enemy AI, spawning, projectiles, collisions, destructible terrain, upgrades, bonuses, scoring and scene transitions.
- Original graphics converted to PNG, maps to JSON and sound effects to stereo PCM WAV.
- Original 256×224 picture, bitmap font and nearest-neighbor scaling.
- Keyboard and standard gamepad controls; 60 Hz and 50 Hz game modes.
- Save/load, automatic high-score saving, fullscreen and mute.
- English STAGE hints, the custom application icon and direct startup at the game menu.

## Play

Linux needs a graphical desktop with X11/XWayland, OpenGL and PulseAudio or ALSA.
The macOS bundles target macOS 12 or newer. They are not Apple-notarized; local
builds can be ad-hoc signed by the build helper. Windows and macOS binaries are
cross-compiled; physical-device testing on those systems has not been completed.

### Controls

| Action | Player 1 | Player 2 |
|---|---|---|
| Move | Arrow keys | W A S D |
| Fire (controller A / B / C) | Z / X / C; Space also maps to B | G / H / J |
| Start / in-game pause | Enter | Tab |

On **STAGE**: **Enter** confirms, **Space / X** advances, **C** goes back.
In **Construction**, move with the arrows and use **Space / X** or **C** to place
blocks. Pressing again in the same position cycles the block forward or backward.
**Enter** accepts the map and returns to the menu; it is used for the next stage.
Gamepads use the D-pad/left stick, West/South/East face buttons and Start.

| Shortcut | Action |
|---|---|
| Esc | Exit from the main menu; return to the menu from every other scene |
| F1 | Show controls |
| F5 / F9 | Save / load |
| F8 | Restart the game |
| F11 / Alt+Enter | Toggle fullscreen |
| M | Mute / unmute |
| F10 | Exit |

### Options and saves

```sh
./bin/BattleCityMD-linux-amd64 -fullscreen
./bin/BattleCityMD-linux-amd64 -pal
./bin/BattleCityMD-linux-amd64 -aspect pixel -scale 4
```

The same flags work on all platforms. Defaults are 60 updates/s, a 4:3 television
aspect ratio and scale 3. `-pal` selects 50 updates/s; `-aspect pixel` uses square
pixels with integer scaling; `-scale 1..6` sets the initial window size; `-mute`
starts muted. On macOS, pass flags with `open -a <app-path> --args ...` or run the
executable inside `Contents/MacOS/` directly.

High scores and native JSON saves are stored in the `BattleCityMD-Go` subdirectory
of the user configuration folder: `%APPDATA%` on Windows, `$XDG_CONFIG_HOME` or
`~/.config` on Linux, and `~/Library/Application Support` on macOS. The high-score
file from 0.1.x is reused. Emulator save states from 0.1.x cannot be loaded into
this native implementation. Native 50 Hz and 60 Hz saves use separate slots.

## Build

Requires **Go 1.25.7+**. `main.go` is at the project root. Dependencies use Go
modules; the source distribution contains no vendor tree or validation reports.

```sh
go mod download
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o BattleCityMD .
```

Windows PowerShell:

```powershell
$env:CGO_ENABLED="0"
go build -trimpath -ldflags="-s -w -H=windowsgui" -o BattleCityMD.exe .
```

Cross-compile a desktop executable by setting `GOOS` and `GOARCH`:

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w -H=windowsgui" -o bin/BattleCityMD-windows-arm64.exe .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/BattleCityMD-linux-arm64 .
```

Build both macOS app bundles from Windows, Linux or macOS:

```sh
go run build_macos.go
# Or just one CPU:
go run build_macos.go -arch amd64
go run build_macos.go -arch arm64
```

Run the helper with the Go toolchain for the computer performing the build. It
sets the child builds' target environment, packages the ICNS icon and Info.plist,
and uses ad-hoc signing when run on macOS with `codesign` available. A C compiler
and Xcode are not required for compilation. The Go linker signs ARM64 Mach-O code.

Windows icon resources for both CPUs are included. Regenerate after editing the ICO:

```sh
go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico assets/icon.ico -o icon_windows_amd64.syso
go run github.com/akavel/rsrc@v0.10.2 -arch arm64 -ico assets/icon.ico -o icon_windows_arm64.syso
```

## Implementation and fidelity

`internal/game` contains the simulation and PNG renderer; `internal/audioqueue`
plays converted WAV clips; `internal/storage` handles local saves. Assets are
embedded at build time, so installed builds need no external data files.

The original program was used offline for disassembly, resource conversion and
frame/state comparisons. Validation covered level initialization and 100 ticks
on each of the 35 stages, longer gameplay traces, all six bonuses, ice, PAL/pause,
save restoration and selected pixel-for-pixel frames. These checks do not prove
identical behavior in every possible situation. Hardware-derived random entropy
is replaced by a Go clock/seed; analog video/audio and hardware sprite limits are
not reproduced. Desktop input and audio latency also depend on the host system.

Go application: **GPL-3.0**, see [LICENSE](LICENSE). Original resources retain
their ownership. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for attribution.
