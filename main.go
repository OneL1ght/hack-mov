package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"errors"
	_ "unicode"
	_ "unicode/utf8"

	"os"
)

const fccSize uint = 4
const minAtomSize int = 8

type fourCC = [fccSize]byte

type Ftyp struct {
	Size             uint32 // A 32-bit integer that specifies the number of bytes in the atom.
	Type             fourCC // A 32-bit unsigned integer that identifies the atom type, typically represented as a four-character code.
	MajorBrand       fourCC // A 32-bit unsigned integer that represents a file format code.
	MinorVersion     uint32 // A 32-bit field that indicates the file format specification version.
	CompatibleBrands fourCC // A series of unsigned 32-bit integers listing compatible file formats.
}

type Mdat struct {
	Size         int32  // A 32-bit integer that specifies the number of bytes in the atom.
	Type         fourCC // A 32-bit unsigned integer that identifies the atom type, typically represented as a four-character code.
	ExtendedSize int64  // A 64-bit integer that specifies the number of bytes in this media data atom.
	Data         []byte // content
}

func copyBytes(content *[]byte, amount uint32) []byte {
    res := make([]byte, amount)
	copy(res, (*content)[:amount])
    return res
}

func chopBytes(content *[]byte, amount uint64) []byte {
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

func getFtyp(atomData []byte) (Ftyp, error) {
	var ftyp Ftyp
	err := binary.Read(bytes.NewReader(atomData), binary.BigEndian, &ftyp)
	if err != nil {
		return ftyp, err
	}
	return ftyp, nil
}

func getMdat(data []byte) (Mdat, error) {
	const aHeaderFieldsSize = 16

	var mdat Mdat
	if len(data) < aHeaderFieldsSize {
		return mdat, errors.New("too low amount of bytes received!")
	}

	var size int32
	sizeBytes := copyBytes(&data, 4)
	err := binary.Read(bytes.NewReader(sizeBytes), binary.BigEndian, &size)
	if err != nil {
		return mdat, err
	}

	aType := [4]byte(data[4:8])

	var extendedSize int64
	err = binary.Read(bytes.NewReader(data[8:aHeaderFieldsSize]), binary.BigEndian, &extendedSize)
	if err != nil {
		return mdat, err
	}
	extendedSize = max(extendedSize, 0)

	aSize := int64(size)
	if aSize == 1 {
		aSize = extendedSize
	}
	aData := data[aHeaderFieldsSize:aSize]
	mdat = Mdat{
		Size: size,
		Type: aType,
		ExtendedSize: extendedSize,
		Data: aData,
	}
	return mdat, nil
}

func main() {
    path := "manul.mov"
    content, err := os.ReadFile(path)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Total bytes count: %v\n", len(content))
    for len(content) >= minAtomSize {
		var skipSize uint64
		var size uint32
        defaultSizeBytes := copyBytes(&content, 4)
        binary.Read(bytes.NewReader(defaultSizeBytes), binary.BigEndian, &size)
		skipSize = uint64(size)

		withoutSize := content[4:]
		atype := copyBytes(&withoutSize, 4)
		fmt.Printf("Atom size: %d, type: %s\n", size, atype)

		strAType := string(atype)
		if strAType == "ftyp" { // || strAType == "wide" 
			ftyp, err := getFtyp(content[:skipSize])
			if err != nil { panic(err) }
			ftypJson, err := json.MarshalIndent(ftyp, "", "  ")
			fmt.Printf("  ftyp: %v\n", string(ftypJson))
		} else if strAType == "mdat" {
			mdat, err := getMdat(content)
			if err != nil {
				panic(err)
			}
			fmt.Printf("  mdat s: %v, t: %s, es: %v, len d: %v\n", mdat.Size, mdat.Type, mdat.ExtendedSize, len(mdat.Data))
			if skipSize == 1 {
				skipSize = uint64(mdat.ExtendedSize)
			}
		}

		_ = chopBytes(&content, skipSize)
    }
}
