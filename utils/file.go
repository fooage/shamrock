package utils

import (
	"bufio"
	"io"
	"math"
	"os"
	"syscall"
)

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

// DiskUsage Gets the disk usage, the first return parameter is the usage, and
// the second return parameter is the total amount.
func DiskUsage(path string) (uint64, uint64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return 0, 0, err
	}
	used := fs.Blocks - fs.Bfree
	return used * uint64(fs.Bsize), fs.Blocks * uint64(fs.Bsize), nil
}

// BufferReadFile is name implies, with a 2MB buffer read small file.
func BufferReadFile(file *os.File) ([]byte, error) {
	reader := bufio.NewReader(file)
	var (
		data = make([]byte, 0, 16384)
		buf  = make([]byte, 2048)
	)
	for {
		// read chunk data with 2MB buffer and total size expect 16MB
		n, err := reader.Read(buf)
		if err == io.EOF {
			return data, nil
		} else if err != nil {
			return nil, err
		}
		data = append(data, buf[:n]...)
	}
}

// BufferWriteFile is name implies, with a 2MB buffer write small file.
func BufferWriteFile(file *os.File, data []byte) error {
	writer := bufio.NewWriter(file)
	var begin, end, limit = 0, 0, 2048
	for {
		end = int(math.Min(float64(len(data)), float64(end+limit)))
		_, err := writer.Write(data[begin:end])
		if err != nil {
			return err
		}
		if end == len(data) {
			break
		}
		begin = begin + limit
	}
	// flush writer's buffer to disk storage
	err := writer.Flush()
	if err != nil {
		return err
	}
	return nil
}
