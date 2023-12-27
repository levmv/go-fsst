package main

import (
	"bytes"
	"container/heap"
	"fmt"
	"sort"
)

type SymbolTable struct {
	Nsymbols int
	Index    [257]int
	Symbols  [][]byte
}

func NewSymbolTable() *SymbolTable {
	st := SymbolTable{}
	st.Symbols = make([][]byte, 256)
	/*for i := range st.Symbols {
		st.Symbols[i] = []byte
	}*/

	for i := 0; i < 256; i += 1 {
		st.Symbols[i] = append(st.Symbols[i], byte(i))
	}
	return &st
}

func (st *SymbolTable) Insert(s string) {
	st.Symbols = append(st.Symbols, []byte(s))
	st.Nsymbols += 1
}

func (st *SymbolTable) FindLongestSymbol(text []byte) uint16 {
	letter := text[0]
	for code := st.Index[letter]; code <= st.Index[letter+1]; code += 1 {
		if bytes.HasPrefix(text, st.Symbols[code]) {
			return uint16(code)
		}
	}
	return uint16(letter)
}

func (st *SymbolTable) CompressCount(count1 *[512]byte, count2 *[512][512]byte, text []string) {
	for _, str := range text {
		str := []byte(str)
		cur := 0
		end := len(str)
		if end == 0 {
			continue
		}
		code1 := st.FindLongestSymbol(str[cur:])
		for {
			count1[code1] += 1
			cur += len(st.Symbols[code1])
			if cur == end {
				break
			}
			code2 := st.FindLongestSymbol(str[cur:])
			count2[code1][code2] += 1
			code1 = code2
		}

	}
	/*for _, str := range text {

		str := []byte(str)
		pos := 0
		prev := st.FindLongestSymbol(str[pos:])
		code := uint16(prev)
		for (pos + len(st.Symbols[code])) < len(str) {
			pos += len(st.Symbols[code])
			prev = code
			code = uint16(st.FindLongestSymbol(str[pos:]))
			count1[code] += 1
			count2[prev][code] += 1
			if code >= 256 {
				nextByte := str[pos]
				count1[nextByte] += 1
				count2[prev][nextByte] += 1
			}

		}
	}*/
}

func (st *SymbolTable) MakeTable(count1 [512]byte, count2 [512][512]byte) *SymbolTable {
	pq := PriorityQueue{}
	nst := NewSymbolTable()
	candMap := make(map[string]int)
	addCandidate := func(symbol string, count int) {
		if len(symbol) > 8 {
			symbol = symbol[:8]
		}
		gain := len(symbol) * count
		/*	if gain < 156 {
			return
		}*/
		if len(symbol) == 1 {
			gain *= 8
		}
		_, found := candMap[symbol]
		if found {
			candMap[symbol] += gain
			return
		}
		candMap[symbol] = gain
		return
	}

	for code := 0; code < 512; code += 1 {
		if int(count1[code]) > 0 {
			addCandidate(string(st.Symbols[code]), int(count1[code]))
		}
	}

	for code1 := 0; code1 < 512; code1 += 1 {
		for code2 := 0; code2 < 512; code2 += 1 {
			if count2[code1][code2] > 5 {
				s1 := st.Symbols[code1]
				if len(s1) == 8 {
					continue
				}
				s := string(s1) + string(st.Symbols[code2])
				fmt.Printf("%v + %v|", string(s1), string(st.Symbols[code2]))
				addCandidate(s, int(count2[code1][code2]))
			}
		}
	}
	for s, gain := range candMap {
		heap.Push(&pq, &Item{
			value:    s,
			priority: gain,
		})
	}

	for nst.Nsymbols < 255 {
		item := heap.Pop(&pq).(*Item)
		//	fmt.Printf("%v(%v), ", item.value, item.priority)
		nst.Insert(item.value)
	}
	nst.MakeIndex()
	return nst
}

func (st *SymbolTable) MakeIndex() {
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

func BuildSymbolTable(text []string) *SymbolTable {
	st := NewSymbolTable()

	var count1 [512]byte

	for i := 0; i < 5; i += 1 {
		count1 = [512]byte{}
		count2 := [512][512]byte{}
		st.CompressCount(&count1, &count2, text)
		st = st.MakeTable(count1, count2)
	}
	return st
}

func (st *SymbolTable) Print() {
	fmt.Printf("%+v ", st.Symbols[25])
	for i := 256; i < 256+st.Nsymbols; i += 1 {
		fmt.Printf("%v ", string(st.Symbols[i]))
	}
}

func (st SymbolTable) Finnalize() []byte {
	var output = []byte{0, 0, 0, 0, 0, 0, 0, 0}

	for i := 8; i > 0; i -= 1 {
		var curLen uint8 = 0
		for j := 256; j < 256+st.Nsymbols; j += 1 {
			if len(st.Symbols[j]) == i {
				output = append(output, st.Symbols[j]...)
				curLen += 1
			}
		}
		output[8-i] = curLen
	}
	return output
}

func LoadSymbolTable(in []byte) *SymbolTable {
	st := NewSymbolTable()

	from := 8

	for i := 0; i < 8; i++ {
		n := int(in[i])
		fmt.Printf("%v symbols of size %v\n", n, 8-i)
		for j := 0; j < n; j++ {
			to := from + int(8-i)
			fmt.Printf("from %v to %v\n", from, to)
			st.Insert(string(in[from:to]))
			from = to
		}
	}
	st.MakeIndex()
	return st
}

func (st *SymbolTable) Compress(input []byte) []byte {
	var compressed []byte
	var pos = 0

	for pos < len(input) {
		code := st.FindLongestSymbol(input[pos:])
		if code < 256 {
			compressed = append(compressed, 0xff, input[pos])
			pos += 1
		} else {
			s := st.Symbols[code]
			pos += len(s)
			compressed = append(compressed, byte(code-256))
		}
	}

	return compressed
}

func (st *SymbolTable) Decompress(input []byte) string {
	var output []byte
	var pos = 0

	for pos < len(input) {
		if input[pos] == 255 {
			pos += 1
			output = append(output, input[pos])
		} else {
			output = append(output, st.Symbols[256+int(input[pos])]...)
		}
		pos += 1
	}

	return string(output)
}
