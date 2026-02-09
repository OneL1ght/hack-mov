package main

import (
	"bytes"
	"strings"
	"encoding/binary"
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
	if amount > uint64(len((*content))) {
		panic("trying chop too many bytes, out of range!")
	}
    val := (*content)[0:amount]
    (*content) = (*content)[amount:]
    return val
}

func chopFourCC(content *[]byte) []byte {
    if len(*content) < 4 {
        panic("cannot chop fourCC from content of smaller length!")
    }
    return chopBytes(content, 4)
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

func getStruct[T any](atomData []byte, res *T) (error) {
	err := binary.Read(bytes.NewReader(atomData), binary.BigEndian, res)
	if err != nil {
		return err
	}
	return nil
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

func printWithIndent(txt string, indent int) {
	spaces := strings.Repeat(" ", indent)
	fmt.Printf("%s%s\n", spaces, txt)
}

func readAtomHeader(content []byte) AtomHeader {
	var size uint32
	defaultSizeBytes := copyBytes(&content, 4)
	binary.Read(bytes.NewReader(defaultSizeBytes), binary.BigEndian, &size)
	withoutSize := content[4:]
	atype := copyBytes(&withoutSize, 4)
	return AtomHeader{size, [4]byte(atype)}
}

func printAtoms(content []byte, indent int) {
    for len(content) >= minAtomSize {
		var skipSize uint64
		atomHeader := readAtomHeader(content)
		skipSize = uint64(atomHeader.Size)

		printWithIndent(fmt.Sprintf("Atom %s size: %d", atomHeader.Type, atomHeader.Size), indent)

		indent += 2
		switch strAtomType := string(atomHeader.Type[:]); strAtomType {
		case "ftyp":
			ftyp, err := getFtyp(content[:skipSize])
			if err != nil { panic(err) }
			printWithIndent(
				fmt.Sprintf(
					"type: %s, mb: %s, cmb: %s, mv: %d",
					ftyp.Type, ftyp.MajorBrand, ftyp.CompatibleBrands, ftyp.MinorVersion),
				indent)
		case "mdat":
			mdat, err := getMdat(content)
			if err != nil {
				panic(err)
			}
			mdatTxt := fmt.Sprintf(
				"s: %d, es: %v, ds: %d", mdat.Size, mdat.ExtendedSize, len(mdat.Data))
			printWithIndent(mdatTxt, indent)
			if skipSize == 1 {
				skipSize = uint64(mdat.ExtendedSize)
			}
		case "moov":
			printAtoms(content[8:], indent + 2)
		case "mvhd":
			var mvhd MovieHeaderAtom
			err := getStruct(content[:atomHeader.Size], &mvhd)
			if err != nil { panic(err) }
			ts := mvhd.TimeScale
			if ts == 0 {
				ts = 1
			}
			durationSec := float32(mvhd.Duration) / float32(ts)
			printWithIndent(
				fmt.Sprintf("duration: %fs, timeScale: %d", durationSec, mvhd.TimeScale),
				indent)
		}
		indent -= 2

		_ = chopBytes(&content, skipSize)
    }
}

func main() {
    path := "manul.mov"
    content, err := os.ReadFile(path)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Total bytes count: %v\n", len(content))
	printAtoms(content, 0)
}
