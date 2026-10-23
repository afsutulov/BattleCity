package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d = filepath.Join(d, "BattleCityMD-Go")
	return d, os.MkdirAll(d, 0700)
}

// WriteAtomic keeps the previous save intact if a write fails or is interrupted.
func WriteAtomic(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".battlecity-save-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return fmt.Errorf("replace save: %w", err)
	}
	return nil
}
