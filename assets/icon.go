package assets

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
)

//go:embed icon.png
var iconPNG []byte

func WindowIcon() (image.Image, error) { return png.Decode(bytes.NewReader(iconPNG)) }
