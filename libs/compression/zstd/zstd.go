package zstd

import (
	"bytes"
	"io"

	"github.com/klauspost/compress/zstd"
)

func CompressData(input []byte) ([]byte, error) {
	in := bytes.NewReader(input)
	var out bytes.Buffer

	err := Compress(in, &out)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func Compress(in io.Reader, out io.Writer) error {
	enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.SpeedBetterCompression))
	if err != nil {
		return err
	}
	_, err = io.Copy(enc, in)
	if err != nil {
		enc.Close()
		return err
	}
	return enc.Close()
}

func DecompressData(input []byte) ([]byte, error) {
	in := bytes.NewReader(input)
	var out bytes.Buffer

	err := Decompress(in, &out)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func Decompress(in io.Reader, out io.Writer) error {
	d, err := zstd.NewReader(in)
	if err != nil {
		return err
	}
	defer d.Close()

	// Copy content...
	_, err = io.Copy(out, d)
	return err
}
