package util

import "os"

func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	if err == nil {
		return true
	}

	return false
}
