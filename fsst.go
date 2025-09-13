// Package fsst provides an implementation of the Fast Static Symbol Table (FSST)
// compression algorithm, as described in the paper "FSST: Fast Random Access String
// Compression" by Peter Boncz (CWI), Viktor Leis (FSU Jena), Thomas Neumann (TU Munchen)
// (https://www.vldb.org/pvldb/vol13/p2649-boncz.pdf).
//
// This library is designed for the efficient compression and decompression of
// individual, repetitive short strings such as log lines, URLs, or other
// record-oriented data. It uses simple and idiomatic Go implementation of the core
// FSST logic and does not include the platform-specific SIMD batching
// optimizations mentioned in the paper.
//
// The typical workflow is:
//  1. Generate a dictionary from a collection of strings using Build.
//  2. Create Compressor and Decompressor instances using the generated dictionary.
//  3. Use these instances to compress and decompress data.
//
// Example:
//
//	sampleData := []string{
//	    "log_level=INFO, component=query_engine, status=success",
//	    "log_level=WARN, component=storage, status=retrying",
//	}
//
//	dict := fsst.Build(sampleData)
//
//	compressor, _ := fsst.NewCompressor(dict)
//	decompressor, _ := fsst.NewDecompressor(dict)
//
//	original := []byte("log_level=INFO, component=storage, status=success")
//	compressed := compressor.Compress(original)
//	decompressed, _ := decompressor.Decompress(compressed)
package fsst

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

const escapeMarker byte = 0xff

func fsst_hash(w uint64) uint64 {
	w = w * 2971215073
	return w ^ (w >> 15)
}

type symbolTable struct {
	Nsymbols int
	Index    [257]int
	Symbols  [][]byte
}

func newSymbolTable() *symbolTable {
	st := symbolTable{}
	st.Symbols = make([][]byte, 256)

	for i := 0; i < 256; i += 1 {
		st.Symbols[i] = append(st.Symbols[i], byte(i))
	}
	return &st
}

func newSymbolTableFromDict(dict []byte) (*symbolTable, error) {
	if len(dict) < MaxSymbolLength {
		return nil, fmt.Errorf("invalid dictionary: data is too short (got %d bytes, expected at least %d)", len(dict), MaxSymbolLength)
	}

	st := newSymbolTable()

	from := MaxSymbolLength
	for i := 0; i < MaxSymbolLength; i++ {
		n := int(dict[i])
		symLen := MaxSymbolLength - i

		for j := 0; j < n; j++ {
			to := from + symLen
			if to > len(dict) {
				return nil, fmt.Errorf("invalid dictionary: header claims more data than available (corruption suspected)")
			}

			st.insert(string(dict[from:to]))
			from = to
		}
	}

	if st.Nsymbols > NumSymbols-1 {
		return nil, fmt.Errorf("invalid dictionary: too many symbols (%d)", st.Nsymbols)
	}

	st.makeIndex()

	return st, nil
}

func (st *symbolTable) insert(s string) {
	st.Symbols = append(st.Symbols, []byte(s))
	st.Nsymbols += 1
}

func (st *symbolTable) findLongestSymbol(text []byte) (uint16, int) {
	letter := text[0]
	code := st.Index[int(letter)]
	textLen := len(text)

	for code < st.Index[int(letter)+1] {
		sym := st.Symbols[code]
		symLen := len(sym)

		if textLen >= symLen && bytes.Equal(text[:symLen], sym) {
			return uint16(code), symLen
		}
		code++
	}
	return uint16(letter), 1
}

func (st *symbolTable) makeIndex() {
	syms := st.Symbols[256:]

	sort.Slice(syms, func(i, j int) bool {
		s1 := syms[i]
		s2 := syms[j]

		if s1[0] == s2[0] {
			return len(s1) > len(s2)
		}
		return bytes.Compare(s1, s2) < 0
	})

	sentinel := 256 + st.Nsymbols
	for i := 0; i < 257; i++ {
		st.Index[i] = sentinel
	}
	for i := st.Nsymbols - 1; i >= 0; i-- {
		st.Index[int(st.Symbols[i+256][0])] = 256 + i
	}
}

type Compressor struct {
	table *symbolTable
}

// NewCompressor creates and initializes a new Compressor using the provided
// dictionary. It returns an error if the dictionary is structurally malformed
// or corrupt.
func NewCompressor(dict []byte) (*Compressor, error) {
	st, err := newSymbolTableFromDict(dict)
	if err != nil {
		return nil, err
	}
	d := Compressor{}
	d.table = st
	return &d, nil
}

// Compress encodes a byte slice using the compressor's dictionary and
// returns the compressed result.
func (c *Compressor) Compress(input []byte) []byte {
	compressed := make([]byte, 0, len(input)/2)

	pos := 0

	for pos < len(input) {
		code, ln := c.table.findLongestSymbol(input[pos:])
		if code < 256 {
			compressed = append(compressed, escapeMarker, input[pos])
			pos += 1
		} else {
			pos += ln
			compressed = append(compressed, byte(code-256))
		}
	}
	return compressed
}

const (
	MaxSize = NumSymbols * MaxSymbolLength
)

type Decompressor struct {
	data [MaxSize]byte
	lens [NumSymbols]byte
}

// NewDecompressor creates a Decompressor from a serialized FSST dictionary.
// Returns an error if the dictionary blob is invalid.
func NewDecompressor(dict []byte) (*Decompressor, error) {
	st, err := newSymbolTableFromDict(dict)
	if err != nil {
		return nil, err
	}

	d := Decompressor{}

	off := 0
	for i := 0; i < st.Nsymbols; i++ {
		symIdx := 256 + i
		sym := st.Symbols[symIdx]
		symLen := len(sym)
		d.lens[i] = byte(symLen)
		copy(d.data[off:off+symLen], sym)
		off += MaxSymbolLength
	}
	return &d, nil
}

// Decompress decodes a byte slice and returns the original data.
// This function will return an error if the compressed input is malformed
func (d *Decompressor) Decompress(input []byte) ([]byte, error) {
	outputSize := 0
	var scanPos uint
	for scanPos < uint(len(input)) {
		code := input[scanPos]
		if code == escapeMarker {
			if scanPos+1 >= uint(len(input)) {
				return nil, errors.New("malformed input, stream ends with an escape code")
			}
			outputSize++
			scanPos += 2 // Skip escape marker and literal byte
		} else {
			ln := int(d.lens[code])
			if ln == 0 {
				return nil, fmt.Errorf("malformed input: symbol code %d do not exist in dictionary", code)
			}
			outputSize += ln
			scanPos++
		}
	}

	output := make([]byte, outputSize)

	var pos, dst uint

	for pos < uint(len(input)) {
		code := input[pos]
		if code == 0xff {
			output[dst] = input[pos+1]
			dst++
			pos += 2
		} else {
			symLen := uint(d.lens[code])
			from := uint(code) << 3
			copy(output[dst:dst+symLen], d.data[from:from+symLen])
			dst += symLen
			pos++
		}
	}
	return output, nil
}
