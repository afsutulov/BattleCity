// Package assets contains converted images, maps and PCM sounds only.
// No cartridge program or console firmware is embedded in this application.
package assets

import (
	"embed"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"sync"
)

//go:embed *.png maps.json sounds.json sounds
var files embed.FS

type Maps struct {
	Levels   [][][]uint16 `json:"levels"`
	Menu     [][]uint16   `json:"menu"`
	Editor   [][]uint16   `json:"editor"`
	GameOver [][]uint16   `json:"gameover"`
	Score1   [][]uint16   `json:"score1"`
	Score2   [][]uint16   `json:"score2"`
}

type Resources struct {
	Tiles [7]image.Image
	Water [2]image.Image
	Font  image.Image
	Icon  image.Image
	Maps  Maps
}

var once sync.Once
var resources Resources
var loadErr error

func Load() (*Resources, error) {
	once.Do(func() {
		loadImage := func(name string) image.Image {
			f, err := files.Open(name)
			if err != nil {
				loadErr = err
				return nil
			}
			defer f.Close()
			im, _, err := image.Decode(f)
			if err != nil {
				loadErr = err
			}
			return im
		}
		for i := range resources.Tiles {
			resources.Tiles[i] = loadImage(fmt.Sprintf("tiles-%d.png", i))
		}
		for i := range resources.Water {
			resources.Water[i] = loadImage(fmt.Sprintf("water-%d.png", i))
		}
		resources.Font = loadImage("font.png")
		resources.Icon = loadImage("icon.png")
		b, err := files.ReadFile("maps.json")
		if err != nil {
			loadErr = err
			return
		}
		if err := json.Unmarshal(b, &resources.Maps); err != nil {
			loadErr = err
		}
		if len(resources.Maps.Levels) != 35 {
			loadErr = fmt.Errorf("invalid map collection")
		}
	})
	return &resources, loadErr
}

func Sound(name string, pal bool) ([]byte, error) {
	region := "ntsc"
	if pal {
		region = "pal"
	}
	return files.ReadFile("sounds/" + region + "/" + name + ".wav")
}
