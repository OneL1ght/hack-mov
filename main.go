package main

import (
	// "encoding/binary"
	"bytes"
	"encoding/binary"
	"fmt"
	_ "unicode"
	_ "unicode/utf8"

	// "io"
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

    fmt.Printf("File content length: %v\n", len(content))

    _ = chopBytes(&content, 10); // skip reserver 10 bytes
    lockCount := chopBytes(&content, 2);
    var lcU32 uint32
    binary.Read(bytes.NewReader(lockCount), binary.LittleEndian, &lcU32)
    fmt.Printf("lockCount: %v, %#x, |%s|, %d\n", lockCount, lockCount, lockCount, lcU32)
    
    ahSize := chopFourCC(&content)
    var u32 uint32
    binary.Read(bytes.NewReader(ahSize), binary.LittleEndian, &u32)
    fmt.Printf("Current ahSize: %v, %#x, |%s|, %d\n", ahSize, ahSize, ahSize, u32)

    t := chopFourCC(&content)
    fmt.Printf("type: %v, |%s|\n", t, t)

    id := chopUint32(&content, binary.LittleEndian)
    fmt.Printf("atom id: %v, |%s|\n", id, id)
    
    childCount := chopBytes(&content, 2)
    var u16 uint16
    binary.Read(bytes.NewReader(childCount), binary.LittleEndian, &u16)
    fmt.Printf("child count: %v, |%d|\n", childCount, u16)

    _ = chopBytes(&content, 4) // skip reserved
    
    next := chopFourCC(&content)
    fmt.Printf("next fourcc: %v, |%s|\n", next, next)
    
    return
    // for range 12 {
    //     fourcc := chopFourCC(&content)
    //     var u32 uint32
    //     binary.Read(bytes.NewReader(fourcc), binary.LittleEndian, &u32)
    //     fmt.Printf("Current fourcc: %v, %#x, |%s|, %d\n", fourcc, fourcc, fourcc, u32)
    //     // fmt.Printf("File content length: %v\n", len(content))
    //     if string(fourcc) == "moov" { break }
    // }
}
