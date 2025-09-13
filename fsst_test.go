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

			c, _ := NewCompressor(dict)
			d, _ := NewDecompressor(dict)

			for _, origStr := range tc.inputs {
				origBytes := []byte(origStr)
				compBytes := c.Compress(origBytes)
				decompBytes, _ := d.Decompress(compBytes)

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

func calculateWeightedAverageRatio(t *testing.T, filePaths []string) float64 {
	t.Helper()

	var totalOriginalSize, totalCompressedSize int64

	for _, path := range filePaths {
		lines, originalSize := readTestData(t, filepath.Base(path))
		if len(lines) == 0 {
			continue
		}

		totalOriginalSize += originalSize

		dict := Build(lines)
		c, _ := NewCompressor(dict)

		for _, line := range lines {
			totalCompressedSize += int64(len(c.Compress([]byte(line))))
		}
	}

	if totalOriginalSize == 0 {
		t.Fatal("Total original size for ratio calculation was zero.")
	}

	return float64(totalCompressedSize) / float64(totalOriginalSize)
}

func TestIndividualCompressionRatios(t *testing.T) {
	testCases := []struct {
		filename string
		maxRatio float64
	}{
		{"urls", 0.5},
		{"email", 0.49},
		{"ruwiki", 0.391},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			path := filepath.Join("testdata", tc.filename)

			ratio := calculateWeightedAverageRatio(t, []string{path})
			t.Logf("Ratio for %s: %.4f", tc.filename, ratio)

			if ratio > tc.maxRatio {
				t.Errorf("Compression ratio has regressed! %v: got %.4f, want <= %.4f", tc.filename, ratio, tc.maxRatio)
			}
		})
	}
}

func TestAverageCompressionRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping average compression ratio test in short mode.")
	}
	const expectedMaxAverageRatio = 0.45

	testFiles, err := filepath.Glob(filepath.Join("testdata", "*"))
	if err != nil {
		t.Fatalf("Failed to glob for test data files: %v", err)
	}
	if len(testFiles) == 0 {
		t.Skip("No test data files found to calculate average ratio.")
	}

	averageRatio := calculateWeightedAverageRatio(t, testFiles)

	t.Logf("Weighted average compression ratio across %d files: %.4f", len(testFiles), averageRatio)

	if averageRatio > expectedMaxAverageRatio {
		t.Errorf("Average compression ratio has regressed! got %.4f, want <= %.4f",
			averageRatio, expectedMaxAverageRatio)
	}
}

func benchmarkCompress(b *testing.B, filename string) {
	lines, totalBytes := readTestData(b, filename)
	dict := Build(lines)
	c, _ := NewCompressor(dict)

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
	c, _ := NewCompressor(dict)
	d, _ := NewDecompressor(dict)

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
		_, _ = d.Decompress(compressedLines[lineIdx])

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
