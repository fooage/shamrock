package filestore

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
)

func (f *filestoreServer) bufferReadFile(file *os.File) ([]byte, error) {
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

func (f *filestoreServer) bufferWriteFile(file *os.File, data []byte) error {
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

func (f *filestoreServer) generateFilePath(name string) string {
	return fmt.Sprintf("%s/%s", f.storePath, name)
}
