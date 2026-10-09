package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// MigrateProfileIconStorage preserves avatars from the former shared upload
// directory before retiring it. Renames are restart-safe and never overwrite
// an existing avatar. This complements the stored URL migration.
func MigrateProfileIconStorage(dataDir string) error {
	legacy := filepath.Join(dataDir, "channel-artwork")
	info, err := os.Lstat(legacy)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("legacy upload directory is not a directory")
	}
	oldIcons := filepath.Join(legacy, "profile-icons")
	if info, err := os.Lstat(oldIcons); err == nil && !info.IsDir() {
		return fmt.Errorf("legacy avatar directory is not a directory")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	entries, err := os.ReadDir(oldIcons)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) > 0 {
		destination := filepath.Join(dataDir, "profile-icons")
		if info, err := os.Lstat(destination); err == nil && !info.IsDir() {
			return fmt.Errorf("avatar destination is not a directory")
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(destination, 0o755); err != nil {
			return err
		}
		for _, entry := range entries {
			target := filepath.Join(destination, entry.Name())
			if _, err := os.Lstat(target); !os.IsNotExist(err) {
				return fmt.Errorf("avatar destination already exists or is inaccessible: %s", entry.Name())
			}
			if err := os.Rename(filepath.Join(oldIcons, entry.Name()), target); err != nil {
				return err
			}
		}
	}
	return os.RemoveAll(legacy)
}
