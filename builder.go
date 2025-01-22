package fsst

import (
	"container/heap"
)

const (
	SampleTarget = 1 << 14
	SampleMax    = 1 << 15
	SampleLine   = 512
)

type Builder struct {
	table  *symbolTable
	count1 [512]int16
	count2 [512][512]int16
}

func (b *Builder) Build(text []string) []byte {
	b.table = NewSymbolTable()

	for sampleFrac := 8; sampleFrac <= 128; sampleFrac += 30 {
		b.compressCount(sampleFrac, text)
		b.table = b.makeTable()

		b.count1 = [512]int16{}
		b.count2 = [512][512]int16{}
	}
	return b.Finnalize()
}

func (b *Builder) compressCount(sf int, text []string) int {
	var compressedSize int
	for i, str := range text {
		if sf < 128 && int(fsst_hash(uint64(i))&127) > sf {
			continue
		}
		str := []byte(str)
		cur := 0
		end := len(str)
		if end == 0 {
			continue
		}
		var code2 uint16
		code1, len1 := b.table.FindLongestSymbol(str[cur:])
		for {
			b.count1[code1] += 1
			cur += len1 // len(b.table.Symbols[code1])
			compressedSize += 1 + isEscapeCode(code1)

			if cur == end {
				break
			}

			code2, len1 = b.table.FindLongestSymbol(str[cur:])
			b.count2[code1][code2] += 1
			code1 = code2
		}
	}
	return compressedSize
}

func (b *Builder) makeTable() *symbolTable {
	pq := priorityQueue{}
	nst := NewSymbolTable()
	addCandidate := func(symbol string, count int) {
		gain := len(symbol) * count

		if len(symbol) == 1 {
			gain *= 8
		}

		heap.Push(&pq, &Item{
			value:    symbol,
			priority: gain,
		})
	}

	for code1 := 0; code1 < 512; code1 += 1 {
		if b.count1[code1] > 0 {
			addCandidate(string(b.table.Symbols[code1]), int(b.count1[code1]))
		}
		for code2 := 0; code2 < 512; code2 += 1 {
			if b.count2[code1][code2] > 0 {
				s1 := b.table.Symbols[code1]
				if len(s1)+len(b.table.Symbols[code2]) > 8 {
					continue
				}
				s := string(s1) + string(b.table.Symbols[code2])
				addCandidate(s, int(b.count2[code1][code2]))
			}
		}
	}

	for nst.Nsymbols < 255 && pq.Len() > 0 {
		item := heap.Pop(&pq).(*Item)
		nst.Insert(item.value)
	}
	nst.MakeIndex()
	return nst
}

func (b *Builder) Finnalize() []byte {
	output := []byte{0, 0, 0, 0, 0, 0, 0, 0}

	for i := 8; i > 0; i -= 1 {
		var curLen uint8 = 0
		for j := 256; j < 256+b.table.Nsymbols; j += 1 {
			if len(b.table.Symbols[j]) == i {
				output = append(output, b.table.Symbols[j]...)
				curLen += 1
			}
		}
		output[8-i] = curLen
	}
	return output
}

func isEscapeCode(code uint16) int {
	if code >= 256 {
		return 0
	} else {
		return 1
	}
}
