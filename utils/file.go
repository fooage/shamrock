package utils

import "os"

// PathExist determine whether a file or folder exists in the specified path.
func PathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

// IsDir determine whether the given path is a folder.
func IsDir(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

// IsFile determine whether the given path is a file.
func IsFile(path string) bool {
	return !IsDir(path)
}
