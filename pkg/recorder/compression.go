package recorder

import (
	"bytes"
	"io"

	"github.com/klauspost/compress/zstd"
)

// CompressionType defines the compression algorithm to use
type CompressionType int

const (
	// NoCompression indicates no compression
	NoCompression CompressionType = iota
	// ZstdCompression indicates Zstandard compression
	ZstdCompression
)

var (
	// DefaultCompression is the default compression algorithm
	DefaultCompression = ZstdCompression

	// encoder and decoder for zstd are reusable and thread-safe
	zstdEncoder, _ = zstd.NewWriter(nil)
	zstdDecoder, _ = zstd.NewReader(nil)
)

// CompressData compresses a byte slice using the specified compression algorithm
func CompressData(data []byte, compressionType CompressionType) ([]byte, error) {
	if compressionType == NoCompression {
		return data, nil
	}

	// Currently we only support Zstd
	return zstdEncoder.EncodeAll(data, make([]byte, 0, len(data))), nil
}

// DecompressData decompresses a byte slice using the specified compression algorithm
func DecompressData(data []byte, compressionType CompressionType) ([]byte, error) {
	if compressionType == NoCompression {
		return data, nil
	}

	// Currently we only support Zstd
	return zstdDecoder.DecodeAll(data, nil)
}

// NewCompressedWriter returns a writer that compresses data before writing
func NewCompressedWriter(w io.Writer, compressionType CompressionType) io.Writer {
	if compressionType == NoCompression {
		return w
	}

	// Currently we only support Zstd
	encoder, _ := zstd.NewWriter(w)
	return encoder
}

// NewCompressedReader returns a reader that decompresses data after reading
func NewCompressedReader(r io.Reader, compressionType CompressionType) (io.Reader, error) {
	if compressionType == NoCompression {
		return r, nil
	}

	// Currently we only support Zstd
	return zstd.NewReader(r)
}

// CloseCompressedWriter closes the compressed writer if needed
func CloseCompressedWriter(w io.Writer, compressionType CompressionType) error {
	if compressionType == NoCompression {
		return nil
	}

	// Close the writer if it's a zstd writer
	if zw, ok := w.(*zstd.Encoder); ok {
		return zw.Close()
	}
	return nil
}

// CompressAndWriteBytes compresses a byte slice and writes it to a writer
func CompressAndWriteBytes(w io.Writer, data []byte, compressionType CompressionType) (int, error) {
	// If no compression, write directly
	if compressionType == NoCompression {
		return w.Write(data)
	}

	// Compress the data
	compressed, err := CompressData(data, compressionType)
	if err != nil {
		return 0, err
	}

	// Write the compressed data
	return w.Write(compressed)
}

// ReadAndDecompressBytes reads compressed bytes from a reader and decompresses them
func ReadAndDecompressBytes(r io.Reader, compressionType CompressionType) ([]byte, error) {
	// Read all bytes
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}

	// If no compression, return the bytes as is
	if compressionType == NoCompression {
		return buf.Bytes(), nil
	}

	// Decompress the bytes
	return DecompressData(buf.Bytes(), compressionType)
}
