package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path"
	"slices"
	"strings"
	"unicode"

	"os"
)

var parentsAtoms []uint32 = []uint32{moovNum, udtaNum, trakNum, mdiaNum, minfNum, stblNum}

const fccSize uint = 4
const minAtomSize int = 8
const ftypNum uint32 = 0x66747970
const wideNum uint32 = 0x77696465
const mdatNum uint32 = 0x6d646174
const moovNum uint32 = 0x6d6f6f76
const mvhdNum uint32 = 0x6d766864
const trakNum uint32 = 0x7472616b
const tkhdNum uint32 = 0x746b6864
const edtsNum uint32 = 0x65647473
const mdiaNum uint32 = 0x6d646961
const udtaNum uint32 = 0x75647461
const _swrNum uint32 = 0xa9737772
const minfNum uint32 = 0x6d696e66
const stblNum uint32 = 0x7374626c
const stsdNum uint32 = 0x73747364
const stcoNum uint32 = 0x7374636f // chunk offsets
const stscNum uint32 = 0x73747363 // sample to chunk
const mp4aNum uint32 = 0x6d703461
const raw_Num uint32 = 0x72617720

const maxListPrint = 5

type fourCC = [fccSize]byte

type ImgDims struct {
	Width  int32
	Height int32
	Chan   int32
}

func (d ImgDims) totalValues() int32 {
	return d.Width * d.Height * d.Chan
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
	var fullBox FullBox

	header, err := readAtomHeader(data[:8])
	if err != nil {
		return res, err
	}

	err = binary.Read(bytes.NewReader(data[8:12]), binary.BigEndian, &fullBox)
	if err != nil {
		return res, err
	}

	res = &Stco{
		AtomHeader: header,
		FullBox: fullBox,
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
	var header AtomHeader
	err := binary.Read(bytes.NewReader(content), binary.BigEndian, &header)
	return header, err
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

func printAtoms(content []byte, indent int, ainfo bool) {
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

		if ainfo {
			switch atomHeader.Type {
			case ftypNum:
				ftyp, err := getFtyp(content[:skipSize])
				if err != nil {
					panic(err)
				}
				ftypTxt := fmt.Sprintf("type: %s, mb: %s, cmb: %s, mv: %d",
					ftyp.Type, ftyp.MajorBrand, ftyp.CompatibleBrands, ftyp.MinorVersion)
				printWithIndent(ftypTxt, dopInfoIndent)
			case mdatNum:
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
			case mvhdNum:
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
			case _swrNum:
				sgi := content[8:skipSize]
				sgiTxt := underscoreSpecials(string(sgi))
				printWithIndent(fmt.Sprintf("software generated info: %s", sgiTxt), dopInfoIndent)
			case stsdNum:
				var noe int32
				binary.Read(bytes.NewReader(content[12:16]), binary.BigEndian, &noe)
				printWithIndent(fmt.Sprintf("noe: %v", noe), dopInfoIndent)
				sdt := content[16:atomHeader.Size]
				rowSize := 16
				for i := 0; i < len(sdt); {
					start := i * rowSize
					row := sdt[start:start+rowSize]

					var sampleSize, dfmt int32
					var dri int16
					binary.Read(bytes.NewReader(row[:4]), binary.BigEndian, &sampleSize)
					binary.Read(bytes.NewReader(row[4:8]), binary.BigEndian, &dfmt)
					binary.Read(bytes.NewReader(row[14:]), binary.BigEndian, &dri)
					printWithIndent(
						fmt.Sprintf("s: %v, dfmt: %#x, dfmt(str): %v, dri: %v",
							sampleSize, dfmt, uint32ToString(uint32(dfmt)), dri),
						dopInfoIndent)
					tail := underscoreSpecials(string(row[16:sampleSize]))
					printWithIndent(fmt.Sprintf("tail txt: %s", tail), dopInfoIndent)

					i += int(sampleSize)
				}
			case stscNum:
				stsc, err := getStsc(content[:atomHeader.Size])
				if err != nil {
					printWithIndent(fmt.Sprintf("%v", err), indent)
					return
				}
				count := len(stsc.Sample2Chunk)
				if count > 0 {
					for i := range min(maxListPrint, count) {
						row := stsc.Sample2Chunk[i]
						printWithIndent(
							fmt.Sprintf("first: %v, S/Chunk: %v, SdescID: %v", row.First, row.SpC, row.Id),
							dopInfoIndent)
					}
					if count > maxListPrint {
						printWithIndent(fmt.Sprintf("... and + %v lines", count - maxListPrint), dopInfoIndent)
					}
				}
			case stcoNum:
				stco, err := getStco(content[:atomHeader.Size])
				if err != nil {
					printWithIndent(fmt.Sprintf("%v", err), indent)
					return
				}

				count := len(stco.ChunkOffsets)
				if count > 0 {
					for i := range min(maxListPrint, count) {
						printWithIndent(fmt.Sprintf("offset: %v", stco.ChunkOffsets[i]), dopInfoIndent)
					}
					if count > maxListPrint {
						printWithIndent(fmt.Sprintf("... and + %v lines", count - maxListPrint), dopInfoIndent)
					}
				}
			}
		}

		if slices.Contains(parentsAtoms, atomHeader.Type) {
			printAtoms(content[8:skipSize], nextLevelIndent, ainfo)
		}

		printAtoms(content[skipSize:], indent, ainfo)
    }
}

func findAtomsData(data []byte, t uint32, atom *[][]byte) {
    if len(data) >= minAtomSize {
		header, err := readAtomHeader(data)
		if err != nil {
			fmt.Println("cannot read atom header")
			return
		}

		skipSize := uint64(header.Size)
		typeInt  := header.Type
		if typeInt == t {
			*atom = append(*atom, data[:skipSize])
		} else if header.Type == stsdNum {
			findAtomsData(data[16:header.Size], t, atom)
		} else if slices.Contains(parentsAtoms, header.Type) {
			findAtomsData(data[8:skipSize], t, atom)
		}

		findAtomsData(data[skipSize:], t, atom)
	}
}

func getUintOfFourCCStr(s string) (uint32, error) {
	if len(s) != 4 {
		return 0, fmt.Errorf("fourcc string must be contains 4 sybmols")
	}
	b := make([]byte, 0)
	for _, ch := range s {
		b = append(b, byte(ch))
	}
	var res uint32
	err := binary.Read(bytes.NewReader(b), binary.BigEndian, &res)
	if err != nil {
		return 0, err
	}
	return res, nil
}

func uint32ToString(src uint32) string {
	tb := make([]byte, 4)
	binary.BigEndian.PutUint32(tb, src)
	return string(tb)
}

func containsAtom(data []byte, atomNum uint32) bool {
	tmp := make([][]byte, 0)
	findAtomsData(data, atomNum, &tmp)
	return len(tmp) != 0
}

func exportFrames(data []byte, dir string, ) error {
	if !containsAtom(data, trakNum) {
		return errors.New("there are no trak atoms in passed data!")
	}

	var err error
	traksData := make([][]byte, 0)
	findAtomsData(data, trakNum, &traksData)

	var videoStco *Stco
	if len(traksData) != 0 {
		for _, ad := range traksData {
			if containsAtom(ad, raw_Num) {
				tmp := make([][]byte, 0)
				findAtomsData(ad, stcoNum, &tmp)
				if len(tmp) != 1 {
					return errors.New("got invalid result on stco searchin")
				}
				videoStco, err = getStco(tmp[0])
				if err != nil { return err }
			}
		}
	} else {
		fmt.Println("atom", uint32ToString(trakNum), "was not found!")
	}

	if videoStco == nil {
		return errors.New("video stco was not found!")
	}

	if err = os.Mkdir(dir, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	for i, offset := range videoStco.ChunkOffsets {
		imgDims := ImgDims{1920, 1080, 3}
		end := offset + imgDims.totalValues()
		fmt.Printf("saving frame of offset: %v, end: %v\n", offset, end)
		imgPath := path.Join(dir, fmt.Sprintf("img%v.png", i))
		err = writeImg(data[offset:end], imgPath, imgDims)
		if err != nil {
			return fmt.Errorf("couldn't save img due to error: %v", err)
		}
	}
	
	return nil
}

func underscoreSpecials(s string) string {
	f := func(r rune) rune {
		if unicode.IsSpace(r) && r != '\n' && r != '\r' {
			return '_'
		}
		if unicode.IsDigit(r) || unicode.IsLetter(r) || unicode.IsPunct(r) {
			return r
		}
		return '_'
	}

	return strings.Map(f, s)
}

func main() {
	flag.Usage = func() {
		fmt.Printf(`
hack-mov [options] mode

There are 2 modes of this script:
    explore: prints all atoms, and additional info if -ainfo specified
    export:  exports all video frames into created directory in calling directory

Example:
  hack-mov -f video.mov -ainfo explore

Options:
`)
		flag.PrintDefaults()
	}

	filePath := flag.String("f", "", "video file path [explore, export]")
	ainfo    := flag.Bool("ainfo", false, "prints additional info for atoms [explore]")
	flag.Parse()
	tail := flag.Args()
	if len(tail) < 1 {
		panic("no mode was provided")
	}
	mode := tail[0]

	content, err := os.ReadFile(*filePath)
	if err != nil {
		panic(err)
	}

	switch mode {
	case "explore":
		if filePath == nil {
			panic("no file path was provided!")
		}
		printAtoms(content, 0, *ainfo)
	case "export":
		imgsDir := "./imgs-" + path.Base(*filePath)
		exportFrames(content, imgsDir)
	default:
		panic("got wrong mode!")
	}
}
