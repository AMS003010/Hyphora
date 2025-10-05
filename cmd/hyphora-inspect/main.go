package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
)

const recordHeaderSize = 1 + 8 + 8

func main() {
	filePath := flag.String("file", "", "Path to .db file to inspect")
	fullOutput := flag.Bool("full", false, "Show full value output")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Usage: hyphora-inspect -file=data-0.db [--full]")
		return
	}

	f, err := os.Open(*filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fileInfo, err := f.Stat()
	if err != nil {
		panic(err)
	}
	fileSize := fileInfo.Size()

	r := bufio.NewReader(f)
	var offset int64 = 0
	for {
		hdr := make([]byte, recordHeaderSize)
		_, err := io.ReadFull(r, hdr)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("error at offset %d: %v\n", offset, err)
			break
		}

		flags := hdr[0]
		keyLen := int64(binary.BigEndian.Uint64(hdr[1:9]))
		valLen := int64(binary.BigEndian.Uint64(hdr[9:17]))

		key := make([]byte, keyLen)
		if _, err := io.ReadFull(r, key); err != nil {
			panic(err)
		}

		val := make([]byte, valLen)
		if valLen > 0 {
			if _, err := io.ReadFull(r, val); err != nil {
				panic(err)
			}
		}

		valStr := string(val)
		if !*fullOutput && len(valStr) > 4 {
			valStr = valStr[0:4]
		}

		fmt.Printf("offset=%d flags=%02x key=%q value=%q\n",
			offset, flags, string(key), valStr)

		offset += recordHeaderSize + keyLen + valLen
	}

	fmt.Println()
	mb := fileSize / (1024 * 1024)
	kb := (fileSize % (1024 * 1024)) / 1024
	bytes := fileSize % 1024

	fmt.Printf("Total file size: %d MB, %d KB, %d bytes\n", mb, kb, bytes)
}
