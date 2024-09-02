package ioext

import (
	"io"
	"os"
	"time"
)

// CC-BY-SA 4.0 https://stackoverflow.com/a/31122253/1780502 Cerise Limón
type TailReader struct {
	io.ReadCloser
}

func (t TailReader) Read(b []byte) (int, error) {
	for {
		n, err := t.ReadCloser.Read(b)
		if n > 0 {
			return n, nil
		} else if err != io.EOF {
			return n, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func NewTailReader(fileName string) (TailReader, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return TailReader{}, err
	}

	// if _, err := f.Seek(0, 2); err != nil {
	// 	return TailReader{}, err
	// }
	return TailReader{f}, nil
}
