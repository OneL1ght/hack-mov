package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

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
const minfHex = 0x6d696e66
const stblHex = 0x7374626c
const stsdHex = 0x73747364
const stcoHex = 0x7374636f // chunk offsets
const stscHex = 0x73747363 // sample to chunk

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

func writeImg(data []byte, path string) error {
	if len(data) != imgSizeB {
		 return errors.New("got data of invalid length!")
	}
	img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{imgWidth, imgHeight}})
	for r := range imgHeight {
		for c := range imgWidth {
			pos := r * imgWidth * 3 + c * 3
			px  := data[pos:pos+3]
			img.SetRGBA(r, c, color.RGBA{px[0], px[1], px[2], 255})
		}
	}

	out, _ := os.Create(path)
	defer out.Close()
	return png.Encode(out, img)
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
				fmt.Sprintf("duration: %fs, timeScale: %d, matrix: %v", durationSec, mvhd.TimeScale, mvhd.Matrix),
				dopInfoIndent)
		case _swrHex:
			sgi := content[8:skipSize]
			sgiTxt := string(sgi)
			sgiTxt = strings.ReplaceAll(sgiTxt, "\n", "")
			sgiTxt = strings.ReplaceAll(sgiTxt, "\r", "")
			printWithIndent(fmt.Sprintf("software generated info: %s", sgiTxt), dopInfoIndent)
		case stsdHex:
			var dri int16
			binary.Read(bytes.NewReader(content[14:16]), binary.BigEndian, &dri)
			printWithIndent(fmt.Sprintf("Data ref idx: %v", dri), dopInfoIndent)
			printAtoms(content[16:atomHeader.Size], nextLevelIndent)
		case stscHex:
			var noe int32
			binary.Read(bytes.NewReader(content[12:16]), binary.BigEndian, &noe)
			printWithIndent(fmt.Sprintf("noe: %v", noe), dopInfoIndent)

			var lineSize int32 = 12
			sample2Chunk := content[16:atomHeader.Size]
			if int32(len(sample2Chunk)) / lineSize != noe {
				panic("wrong in parsing stts sample2Chunk table!")
			}
			for i := range noe {
				pos := i * lineSize
				var firstChunk int32
				binary.Read(bytes.NewReader(sample2Chunk[pos:pos+4]), binary.BigEndian, &firstChunk)
				var samplesPerChunk int32
				binary.Read(bytes.NewReader(sample2Chunk[pos+4:pos+8]), binary.BigEndian, &samplesPerChunk)
				var id int32
				binary.Read(bytes.NewReader(sample2Chunk[pos+8:pos+12]), binary.BigEndian, &id)
				printWithIndent(fmt.Sprintf("first: %v, S/Chunk: %v, SdescID: %v", firstChunk, samplesPerChunk, id), dopInfoIndent)
			}
		case stcoHex:
			version := content[8:9]
			flags   := content[9:12]
			var noe int32
			binary.Read(bytes.NewReader(content[12:16]), binary.BigEndian, &noe)
			offset2Chunk := content[16:atomHeader.Size]
			offset2ChunkCount := len(offset2Chunk) / 4
			printWithIndent(
				fmt.Sprintf("v: %d, flags: %v, noe: %v, o2chCount: %v", version, flags, noe, offset2ChunkCount),
				dopInfoIndent)

			var offset2ChunkArr []int32 = make([]int32, offset2ChunkCount)
			binary.Read(bytes.NewReader(offset2Chunk), binary.BigEndian, &offset2ChunkArr)
			for i := 0; i < offset2ChunkCount; i += 2 {
				if i < 1 { continue }
				printWithIndent(
					// fmt.Sprintf("offset: %v, chunk: %v", offset2ChunkArr[i], offset2ChunkArr[i+1]),
					fmt.Sprintf("offset: %v", offset2ChunkArr[i]), dopInfoIndent)
			}

			version := content[8:9]
			flags := content[9:12]
			var noe int32
			binary.Read(bytes.NewReader(content[12:16]), binary.BigEndian, &noe)
			cot := content[16:atomHeader.Size]
			printWithIndent(fmt.Sprintf("v: %d, flags: %v, noe: %v cot: %v", version, flags, noe, cot), dopInfoIndent)
		case moovHex, udtaHex, trakHex, mdiaHex, minfHex, stblHex: // atoms contains children
			printAtoms(content[8:skipSize], nextLevelIndent)
		}

		printAtoms(content[skipSize:], indent)
    }
}

func main() {
	args := os.Args
	if len(args) < 2 {
		panic("please provide video file path as an argument!")
	}
    path := os.Args[1]
    content, err := os.ReadFile(path)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Total bytes count: %v\n", len(content))
	printAtoms(content, 0)
}
