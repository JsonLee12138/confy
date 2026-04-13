package internal

import "os"

func Exists(path string) (isDir bool, exists bool, err error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return info.IsDir(), true, nil
}
