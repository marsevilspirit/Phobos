package util

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Unzip(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	uzdata, err := io.ReadAll(gr)
	if err != nil {
		return nil, err
	}
	return uzdata, err
}

func Zip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
