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
const ftypHex = 0x66747970
const wideHex = 0x77696465
const mdatHex = 0x6d646174
const moovHex = 0x6d6f6f76
const mvhdHex = 0x6d766864
const trakHex = 0x7472616b
const tkhdHex = 0x746b6864
const edtsHex = 0x65647473
const mdiaHex = 0x6d646961
const udtaHex = 0x75647461
const _swrHex = 0xa9737772

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
	var size, typeInt uint32
	defaultSizeBytes := copyBytes(&content, 4)
	binary.Read(bytes.NewReader(defaultSizeBytes), binary.BigEndian, &size)
	withoutSize := content[4:]
	typeBytes := copyBytes(&withoutSize, 4)
	binary.Read(bytes.NewReader(typeBytes), binary.BigEndian, &typeInt)
	return AtomHeader{size, typeInt}
}

func printAtoms(content []byte, indent int) {
	dopInfoIndent   := indent + 1
	nextLevelIndent := indent + 4
    if len(content) >= minAtomSize {
		var skipSize uint64
		atomHeader := readAtomHeader(content)
		skipSize = uint64(atomHeader.Size)

		typeSymbs := make([]byte, 4)
		binary.BigEndian.PutUint32(typeSymbs, atomHeader.Type)

		printWithIndent(
			fmt.Sprintf("Atom %s |%#x| size: %d",
			typeSymbs, atomHeader.Type, atomHeader.Size), indent)

		switch atomHeader.Type {
		case ftypHex:
			ftyp, err := getFtyp(content[:skipSize])
			if err != nil {
				panic(err)
			}
			ftypTxt := fmt.Sprintf("type: %s, mb: %s, cmb: %s, mv: %d",
				ftyp.Type, ftyp.MajorBrand, ftyp.CompatibleBrands, ftyp.MinorVersion)
			printWithIndent(ftypTxt, dopInfoIndent)
		case mdatHex:
			mdat, err := getMdat(content)
			if err != nil {
				panic(err)
			}
			mdatTxt := fmt.Sprintf(
				"s: %d, es: %v, ds: %d", mdat.Size, mdat.ExtendedSize, len(mdat.Data))
			printWithIndent(mdatTxt, dopInfoIndent)
			if skipSize == 1 {
				skipSize = uint64(mdat.ExtendedSize)
			}
		case mvhdHex:
			var mvhd MovieHeaderAtom
			err := getStruct(content[:atomHeader.Size], &mvhd)
			if err != nil {
				panic(err)
			}
			ts := mvhd.TimeScale
			if ts == 0 {
				ts = 1
			}
			durationSec := float32(mvhd.Duration) / float32(ts)
			// TODO: get the matrix
			printWithIndent(
				fmt.Sprintf("duration: %fs, timeScale: %d", durationSec, mvhd.TimeScale),
				dopInfoIndent)
		case _swrHex:
			sgi := content[8:skipSize]
			sgiTxt := string(sgi)
			sgiTxt = strings.ReplaceAll(sgiTxt, "\n", "")
			sgiTxt = strings.ReplaceAll(sgiTxt, "\r", "")
			printWithIndent(fmt.Sprintf("software generated info: %s", sgiTxt), dopInfoIndent)
		case moovHex, udtaHex, trakHex, mdiaHex: // atoms contains children
			printAtoms(content[8:skipSize], nextLevelIndent)
		}

		printAtoms(content[skipSize:], indent)
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
