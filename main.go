package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

func BuildDict(infile string, outfile string) {
	inFile, err := os.ReadFile(infile)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(inFile), "\n")
	fmt.Println(len(lines))

	table := BuildSymbolTable(lines[:2000])

	f, err := os.OpenFile(outfile, os.O_CREATE+os.O_WRONLY, fs.ModePerm)
	defer f.Close()
	if err != nil {
		panic(err)
	}

	n, err := f.Write(table.Finnalize())
	fmt.Println(n)

	if err != nil {
		panic(err)
	}
}

func CompressFile(dictfile string, infile string, outfile string) {
	f, err := os.ReadFile(dictfile)
	if err != nil {
		panic(err)
	}
	table := LoadSymbolTable(f)

	inFile, err := os.ReadFile(infile)
	if err != nil {
		panic(err)
	}

	fout, err := os.OpenFile(outfile, os.O_CREATE+os.O_WRONLY, fs.ModePerm)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(inFile), "\n")
	for _, line := range lines {
		compressed := table.Compress([]byte(line))
		fmt.Printf("%v -> %v\n", len(line), len(compressed))
		fout.Write(table.Compress([]byte(line)))
	}
	fout.Close()
}

func main() {

	//BuildDict("urls5.csv", "t2.dict")
	//CompressFile("t2.dict", "urls5.csv", "urls5.comp")

	BuildDict("urls2.txt", "t3.dict")
	CompressFile("t3.dict", "urls2.txt", "urls2.comp2")
}
