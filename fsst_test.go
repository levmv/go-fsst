package fsst

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundtrip(t *testing.T) {
	testCases := []struct {
		name   string
		inputs []string
	}{
		{
			name: "realistic_data",
			inputs: []string{
				"hello world, this is a test",
				"hello world, this is another test",
				"https://www.google.com/search?q=fsst",
				"https://www.google.com/search?q=golang",
				"log_level=INFO, component=query_engine, status=success",
				"log_level=WARN, component=storage, status=retrying",
			},
		},
		{
			name: "edge_cases",
			inputs: []string{
				"",
				"a",
				"ab",
				"aaaaaaaaaaaaaaa",
				"\x00\x01\x02\x03",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dict := Build(tc.inputs)

			c := NewCompressor(dict)
			d := NewDecompressor(dict)

			for _, origStr := range tc.inputs {
				origBytes := []byte(origStr)
				compBytes := c.Compress(origBytes)
				decompBytes := d.Decompress(compBytes)

				if !bytes.Equal(origBytes, decompBytes) {
					t.Errorf(`Roundtrip mismatch:
					  Original:     %q
					  Decompressed: %q`, origStr, string(decompBytes))
				}
			}
		})
	}
}

func readTestData(t testing.TB, fileName string) ([]string, int64) {
	t.Helper()
	path := filepath.Join("testdata", fileName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read test data file %q: %v", fileName, err)
	}

	byteLines := bytes.Split(content, []byte("\n"))

	lines := make([]string, len(byteLines))
	var totalSize int64
	for i, lineBytes := range byteLines {
		lines[i] = string(lineBytes)
		totalSize += int64(len(lineBytes))
	}

	return lines, totalSize
}

func benchmarkCompress(b *testing.B, filename string) {
	lines, totalBytes := readTestData(b, filename)
	dict := Build(lines)
	c := NewCompressor(dict)

	byteLines := make([][]byte, len(lines))
	for i, s := range lines {
		byteLines[i] = []byte(s)
	}

	avgLineBytes := totalBytes / int64(len(lines))
	b.SetBytes(avgLineBytes)
	b.ResetTimer()
	b.ReportAllocs()

	lineIdx := 0
	for i := 0; i < b.N; i++ {
		_ = c.Compress(byteLines[lineIdx])

		lineIdx++
		if lineIdx >= len(byteLines) {
			lineIdx = 0
		}
	}
}

func benchmarkDecompress(b *testing.B, filename string) {
	lines, totalBytes := readTestData(b, filename)
	dict := Build(lines)
	c := NewCompressor(dict)
	d := NewDecompressor(dict)

	compressedLines := make([][]byte, len(lines))
	for i, s := range lines {
		compressedLines[i] = c.Compress([]byte(s))
	}

	avgLineBytes := totalBytes / int64(len(lines))
	b.SetBytes(avgLineBytes)
	b.ResetTimer()
	b.ReportAllocs()

	lineIdx := 0
	for i := 0; i < b.N; i++ {
		_ = d.Decompress(compressedLines[lineIdx])

		lineIdx++
		if lineIdx >= len(compressedLines) {
			lineIdx = 0
		}
	}
}

func BenchmarkCompress_URLs(b *testing.B)     { benchmarkCompress(b, "urls") }
func BenchmarkDecompress_URLs(b *testing.B)   { benchmarkDecompress(b, "urls") }
func BenchmarkCompress_Cities(b *testing.B)   { benchmarkCompress(b, "city") }
func BenchmarkDecompress_Cities(b *testing.B) { benchmarkDecompress(b, "city") }
