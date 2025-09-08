package fsst

import (
	"sort"
)

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

func newSymbolTableFromDict(in []byte) *symbolTable {
	st := newSymbolTable()

	from := MaxSymbolLength
	for i := 0; i < MaxSymbolLength; i++ {
		n := int(in[i])
		for j := 0; j < n; j++ {
			to := from + int(MaxSymbolLength-i)
			st.insert(string(in[from:to]))
			from = to
		}
	}

	st.makeIndex()

	return st
}

func (st *symbolTable) insert(s string) {
	st.Symbols = append(st.Symbols, []byte(s))
	st.Nsymbols += 1
}

func (st *symbolTable) findLongestSymbol(text []byte) (uint16, int) {
	letter := text[0]
	code := st.Index[int(letter)]
	textLen := len(text)
	for {
		symLen := len(st.Symbols[code])
		if textLen >= len(st.Symbols[code]) && string(text[0:symLen]) == string(st.Symbols[code]) {
			return uint16(code), symLen
		}
		code += 1
		if code >= len(st.Symbols) {
			return uint16(letter), 1
		}
		if st.Symbols[code][0] != letter {
			break
		}
	}
	return uint16(letter), 1
}

func (st *symbolTable) makeIndex() {
	sort.Slice(st.Symbols[256:256+st.Nsymbols], func(i, j int) bool {
		if st.Symbols[256+i][0] == st.Symbols[256+j][0] {
			return len(st.Symbols[256+i]) > len(st.Symbols[256+j])
		}
		return string(st.Symbols[256+i]) < string(st.Symbols[256+j])
	})
	for i := st.Nsymbols - 1; i >= 0; i-- {
		st.Index[int(st.Symbols[i+256][0])] = 256 + i
	}
	st.Index[256] = 256 + st.Nsymbols
}

type Compressor struct {
	table *symbolTable
}

func NewCompressor(dict []byte) *Compressor {
	d := Compressor{}
	d.table = newSymbolTableFromDict(dict)
	return &d
}

func (c *Compressor) Compress(input []byte) []byte {
	compressed := make([]byte, 0, len(input)/2)

	pos := 0

	for pos < len(input) {
		code, len := c.table.findLongestSymbol(input[pos:])
		if code < 256 {
			compressed = append(compressed, 0xff, input[pos])
			pos += 1
		} else {
			pos += len
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

func NewDecompressor(dict []byte) *Decompressor {
	d := Decompressor{}
	st := newSymbolTableFromDict(dict)

	off := 0
	for i := 0; i < st.Nsymbols; i++ {
		symIdx := 256 + i
		sym := st.Symbols[symIdx]
		symLen := len(sym)
		d.lens[i] = byte(symLen)
		copy(d.data[off:off+symLen], sym)
		off += MaxSymbolLength
	}
	return &d
}

func (d *Decompressor) Decompress(input []byte) []byte {
	outputSize := 0
	var scanPos uint
	for scanPos < uint(len(input)) {
		code := input[scanPos]
		if code == 0xff {
			outputSize++
			scanPos += 2 // Skip escape marker and literal byte
		} else {
			outputSize += int(d.lens[code])
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
	return output
}
