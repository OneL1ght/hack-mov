package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"

	"os"
)

const fccSize uint = 4
const minAtomSize int = 8
const ftypHex uint32 = 0x66747970
const wideHex uint32 = 0x77696465
const mdatHex uint32 = 0x6d646174
const moovHex uint32 = 0x6d6f6f76
const mvhdHex uint32 = 0x6d766864
const trakHex uint32 = 0x7472616b
const tkhdHex uint32 = 0x746b6864
const edtsHex uint32 = 0x65647473
const mdiaHex uint32 = 0x6d646961
const udtaHex uint32 = 0x75647461
const _swrHex uint32 = 0xa9737772
const minfHex uint32 = 0x6d696e66
const stblHex uint32 = 0x7374626c
const stsdHex uint32 = 0x73747364
const stcoHex uint32 = 0x7374636f // chunk offsets
const stscHex uint32 = 0x73747363 // sample to chunk
const mp4aHex uint32 = 0x6d703461
const raw_Hex uint32 = 0x72617720

type fourCC = [fccSize]byte

type ImgDims struct {
	Width  int32
	Height int32
	Chan   int32
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

func getStco(data []byte) (*Stco, error) {
	var res *Stco
	header, err := readAtomHeader(data[:8])
	if err != nil {
		return res, err
	}

	res = &Stco{
		Size: header.Size,
		Type: header.Type,
		Version: data[8],
		ChunkOffsets: make([]int32, len(data[16:header.Size]) / 4),
	}

	err = binary.Read(bytes.NewReader(data[9:12]), binary.BigEndian, &res.Flags)
	if err != nil {
		return res, err
	}

	err = binary.Read(bytes.NewReader(data[12:16]), binary.BigEndian, &res.NOE)
	if err != nil {
		return res, err
	}

	err = binary.Read(bytes.NewReader(data[16:header.Size]), binary.BigEndian, &res.ChunkOffsets)
	if err != nil {
		return res, err
	}

	return res, nil
}

func getStsc(data []byte) (*Stsc, error) {
	var res *Stsc
	header, err := readAtomHeader(data[:8])
	if err != nil {
		return res, err
	} else if uint32(len(data)) < header.Size {
		return res, fmt.Errorf("invalid length of data, lower than header size")
	}

	sample2ChunkSize := len(data[16:header.Size]) / 12
	res = &Stsc{
		Size: header.Size,
		Type: header.Type,
		Version: data[8],
		Sample2Chunk: make([]SampleChunkRow, sample2ChunkSize),
	}

	err = binary.Read(bytes.NewReader(data[9:12]), binary.BigEndian, &res.Flags)
	if err != nil {
		return res, err
	}

	err = binary.Read(bytes.NewReader(data[12:16]), binary.BigEndian, &res.NOE)
	if err != nil {
		return res, err
	}
	if res.NOE != uint32(sample2ChunkSize) {
		return res, fmt.Errorf(
			"number of entries not equal calculated sample2chunk table size, noe(%v) != table size(%v)",
			res.NOE, sample2ChunkSize)
	}

	err = binary.Read(bytes.NewReader(data[16:header.Size]), binary.BigEndian, &res.Sample2Chunk)
	if err != nil {
		return res, err
	}

	return res, nil
}

func printWithIndent(txt string, indent int) {
	spaces := strings.Repeat(" ", indent)
	fmt.Printf("%s%s\n", spaces, txt)
}

func readAtomHeader(content []byte) (AtomHeader, error) {
	if len(content) < 8 {
		return AtomHeader{}, errors.New(
			"got too few bytes for extract atom header: " + strconv.Itoa(len(content)))
	}
	var size, typeInt uint32
	defaultSizeBytes := copyBytes(&content, 4)
	binary.Read(bytes.NewReader(defaultSizeBytes), binary.BigEndian, &size)
	withoutSize := content[4:]
	typeBytes   := copyBytes(&withoutSize, 4)
	binary.Read(bytes.NewReader(typeBytes), binary.BigEndian, &typeInt)
	return AtomHeader{size, typeInt}, nil
}

func writeImg(data []byte, path string, dims ImgDims) error {
	imgSizeB := dims.Width * dims.Height * dims.Chan
	if int32(len(data)) != imgSizeB {
		 return errors.New("got data of invalid length!")
	}
	img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{int(dims.Width), int(dims.Height)}})
	for r := range dims.Height {
		for c := range dims.Width {
			pos := r * dims.Width * dims.Chan + c * dims.Chan
			px  := data[pos:pos+dims.Chan]
			img.SetRGBA(int(c), int(r), color.RGBA{px[0], px[1], px[2], 255})
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
		atomHeader, err := readAtomHeader(content)
		if err != nil {
			printWithIndent(fmt.Sprintf("%v", err), indent)
			return
		}

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
			stsc, err := getStsc(content[:atomHeader.Size])
			if err != nil {
				printWithIndent(fmt.Sprintf("%v", err), indent)
				return
			}
			for _, row := range stsc.Sample2Chunk {
				printWithIndent(
					fmt.Sprintf("first: %v, S/Chunk: %v, SdescID: %v", row.First, row.SpC, row.Id),
					dopInfoIndent)
			}
		case stcoHex:
			stco, err := getStco(content[:atomHeader.Size])
			if err != nil {
				printWithIndent(fmt.Sprintf("%v", err), indent)
				return
			}
			for i := 0; i < len(stco.ChunkOffsets); i += 2 {
				if i < 1 { continue }
				printWithIndent(fmt.Sprintf("offset: %v", stco.ChunkOffsets[i]), dopInfoIndent)
			}
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
