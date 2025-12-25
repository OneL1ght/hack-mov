package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	_ "unicode"
	_ "unicode/utf8"

	"os"
)

func chopBytes(content *[]byte, amount uint) []byte {
    val := (*content)[0:amount]
    (*content) = (*content)[amount:]
    return val
}

func chopFourCC(content *[]byte) []byte {
    if len(*content) < 4 {
        panic("cannot chop fourCC from content of smaller length!")
    }
    fourcc := chopBytes(content, 4)
    return []byte(fourcc)
}

func chopUint32(content *[]byte, order binary.ByteOrder) uint32 {
    if len(*content) < 4 {
        panic("cannot chop fourCC from content of smaller length!")
    }
    var u32 uint32
    data := chopBytes(content, 4)
    err := binary.Read(bytes.NewReader(data), order, &u32)
    if err != nil { panic(err) }
    return u32
}

func main() {
    path := "manul.mov"
    content, err := os.ReadFile(path)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Total bytes count: %v\n", len(content))

    for len(content) > 0 {
        var sizeU32 uint32

        size := chopFourCC(&content)
        binary.Read(bytes.NewReader(size), binary.BigEndian, &sizeU32)
        fmt.Printf("Current size: %v, %#x, %d\n", size, size, sizeU32)

        atype := chopFourCC(&content)
        fmt.Printf("Current type: %v, %#x, |%s|\n", atype, atype, atype)

		_ = chopBytes(&content, uint(sizeU32) - 8)
    }
}
