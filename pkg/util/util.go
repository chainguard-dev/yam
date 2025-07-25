// Package util provides utility functions for the yam YAML formatter.
//
//nolint:revive // util package name is acceptable for internal utilities
package util

import "path/filepath"

func IsYAML(path string) bool {
	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		return true
	}

	return false
}

const ConfigFileName = ".yam.yaml"
