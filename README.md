# go-fsst

Simple and straightforward implementation of [FSST](https://www.vldb.org/pvldb/vol13/p2649-boncz.pdf) compression algoritm.


```go
package main

import (
	"bytes"
	"fmt"
	"github.com/levmv/go-fsst"
)

func main() {
	// A collection of sample strings to train the dictionary.
	trainingData := []string{
		"log_level=INFO, component=query_engine, status=success",
		"log_level=WARN, component=storage, status=retrying",
		"log_level=INFO, component=query_engine, status=failed",
	}

	// Build a dictionary from the samples.
	dict := fsst.Build(trainingData)

	// Create a compressor and a decompressor with the dictionary.
	compressor, err := fsst.NewCompressor(dict)
	if err != nil {/*...*/}

	decompressor, err := fsst.NewDecompressor(dict)
	if err != nil {/*...*/}

	// Compress and decompress new strings.
	original := []byte("log_level=INFO, component=storage, status=success")
	compressed := compressor.Compress(original)
	decompressed, _ := decompressor.Decompress(compressed)

	fmt.Printf("Original size:     %d\n", len(original))
	fmt.Printf("Compressed size:   %d\n", len(compressed))
	fmt.Printf("Roundtrip success: %t\n", bytes.Equal(original, decompressed))
	// Output:
	// Original size:     49
	// Compressed size:   10
	// Roundtrip success: true
}
```