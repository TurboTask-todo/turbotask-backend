package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// Compressor defines the interface for compression operations
type Compressor interface {
	Compress(data []byte) ([]byte, string, error)
	Decompress(data []byte, compressionType string) ([]byte, error)
}

// compressor implements the Compressor interface
type compressor struct{}

// NewCompressor creates a new compressor instance
func NewCompressor() Compressor {
	return &compressor{}
}

// Compress compresses data using the best algorithm based on size
func (c *compressor) Compress(data []byte) ([]byte, string, error) {
	if len(data) < 1024 {
		return data, "none", nil
	}

	// Use gzip for general compression
	compressed, err := c.compressGzip(data)
	if err != nil {
		return data, "none", err
	}

	// Only return compressed if it's actually smaller
	if len(compressed) < len(data) {
		return compressed, "gzip", nil
	}

	return data, "none", nil
}

// Decompress decompresses data based on the compression type
func (c *compressor) Decompress(data []byte, compressionType string) ([]byte, error) {
	switch compressionType {
	case "none":
		return data, nil
	case "gzip":
		return c.decompressGzip(data)
	default:
		return nil, fmt.Errorf("unsupported compression type: %s", compressionType)
	}
}

// compressGzip compresses data using gzip
func (c *compressor) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}

	err = gz.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressGzip decompresses gzip data
func (c *compressor) decompressGzip(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	decompressed, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}
