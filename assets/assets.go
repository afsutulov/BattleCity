package assets

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
)

//go:embed original.gen
var original []byte

const SHA256 = "1cad5b6fb2ccbbb4d096ee951927457883ba8931f7df8f5ef19a08dc293a0a2c"

func ROM() ([]byte, error) {
	if fmt.Sprintf("%x", sha256.Sum256(original)) != SHA256 {
		return nil, fmt.Errorf("embedded game failed integrity check")
	}
	return append([]byte(nil), original...), nil
}

// PlayableROM applies only the requested presentation changes to a private
// cartridge copy. Original bytes and third-party source notices are retained.
func PlayableROM() ([]byte, error) {
	b, err := ROM()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(b[0x5d16:0x5d1a], []byte{0x61, 0x00, 0x00, 0xf8}) {
		return nil, fmt.Errorf("unexpected SEGA splash call; refusing ROM patch")
	}
	// Skip only the SEGA logo routine; the preceding game initialization stays.
	copy(b[0x5d16:0x5d1a], []byte{0x4e, 0x71, 0x4e, 0x71})
	if !bytes.Equal(b[0x5d1a:0x5d1e], []byte{0x61, 0x00, 0x02, 0x40}) {
		return nil, fmt.Errorf("unexpected credits call; refusing ROM patch")
	}
	// BSR.W showMessage -> two NOPs to skip the author splash.
	copy(b[0x5d1a:0x5d1e], []byte{0x4e, 0x71, 0x4e, 0x71})
	const row = 0x26a6d
	want := []byte{0, 0, 0, 0, 0, 0, 'W', 'W', 'W', 0x69, 'K', 'R', 'I', 'K', 0x1b, 0x1b, 0x69, 'C', 'O', 'M', 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(b[row:row+26], want) {
		return nil, fmt.Errorf("unexpected menu URL row; refusing ROM patch")
	}
	clear(b[row : row+26])
	return b, nil
}
