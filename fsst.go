package fsst

import (
	"fmt"
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

func NewSymbolTable() *symbolTable {
	st := symbolTable{}
	st.Symbols = make([][]byte, 256)

	for i := 0; i < 256; i += 1 {
		st.Symbols[i] = append(st.Symbols[i], byte(i))
	}
	return &st
}

func LoadTable(in []byte) *symbolTable {
	st := NewSymbolTable()
	from := 8
	for i := 0; i < 8; i++ {
		n := int(in[i])
		for j := 0; j < n; j++ {
			to := from + int(8-i)
			st.Insert(string(in[from:to]))
			from = to
		}
	}
	return st
}

func (st *symbolTable) Insert(s string) {
	st.Symbols = append(st.Symbols, []byte(s))
	st.Nsymbols += 1
}

func (st *symbolTable) FindLongestSymbol(text []byte) (uint16, int) {
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

func (st *symbolTable) MakeIndex() {
	sort.Slice(st.Symbols[256:256+st.Nsymbols], func(i, j int) bool {
		if st.Symbols[256+i][0] == st.Symbols[256+j][0] {
			return len(st.Symbols[256+i]) > len(st.Symbols[256+j])
		}
		return string(st.Symbols[256+i]) < string(st.Symbols[256+j])
	})
	for i := st.Nsymbols - 1; i > 0; i -= 1 {
		st.Index[int(st.Symbols[i+256][0])] = 256 + i
	}
	st.Index[256] = 256 + st.Nsymbols
}

func (st *symbolTable) Print() {
	for i := 256; i < 256+st.Nsymbols; i += 1 {
		fmt.Printf("%v ", string(st.Symbols[i]))
	}
	fmt.Println("")
}

func LoadSymbolTable(in []byte) *symbolTable {
	st := NewSymbolTable()

	from := 8
	for i := 0; i < 8; i++ {
		n := int(in[i])
		for j := 0; j < n; j++ {
			to := from + int(8-i)
			st.Insert(string(in[from:to]))
			from = to
		}
	}
	st.MakeIndex()
	return st
}

type Compressor struct {
	table *symbolTable
}

func NewCompressor(dict []byte) *Compressor {
	d := Compressor{}
	d.table = LoadTable(dict)
	d.table.MakeIndex()
	return &d
}

func (c *Compressor) Compress(input []byte) []byte {
	compressed := make([]byte, 0, len(input)/2)

	pos := 0

	for pos < len(input) {
		code, len := c.table.FindLongestSymbol(input[pos:])
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

type Decompressor struct {
	table *symbolTable
}

func NewDecompressor(dict []byte) *Decompressor {
	d := Decompressor{}
	d.table = LoadTable(dict)
	return &d
}

func (d *Decompressor) Decompress(input []byte) []byte {
	output := make([]byte, 0, len(input)*3)
	pos := 0

	for pos < len(input) {
		if input[pos] == 255 {
			pos++
			output = append(output, input[pos])
		} else {
			output = append(output, d.table.Symbols[256+int(input[pos])]...)
		}
		pos += 1
	}

	return output
}
