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
	"math"
	"path"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unsafe"

	"os"
)

var parentsAtoms []uint32 = []uint32{moovNum, udtaNum, trakNum, mdiaNum, minfNum, stblNum, dinfNum, metaNum}
var videoComprNums []uint32 = []uint32{
	getUintOfFourCCStr("cvid"), //Cinepak
	getUintOfFourCCStr("jpeg"), //JPEG
	getUintOfFourCCStr("smc "), //Graphics
	getUintOfFourCCStr("rle "), //Animation
	getUintOfFourCCStr("rpza"), //Apple video
	getUintOfFourCCStr("kpcd"), //Kodak Photo CD
	getUintOfFourCCStr("png "), //Portable Network Graphics
	getUintOfFourCCStr("mjpa"), //Motion-JPEG (format A)
	getUintOfFourCCStr("mjpb"), //Motion-JPEG (format B)
	getUintOfFourCCStr("SVQ1"), //Sorenson video, version 1
	getUintOfFourCCStr("SVQ3"), //Sorenson video 3
	getUintOfFourCCStr("mp4v"), //MPEG-4 video
	getUintOfFourCCStr("avc1"), //H.264 video
	getUintOfFourCCStr("dvc "), //NTSC DV-25 video
	getUintOfFourCCStr("dvcp"), //PAL DV-25 video
	getUintOfFourCCStr("gif "), //CompuServe Graphics Interchange Format
	getUintOfFourCCStr("h263"), //H.263 video
	getUintOfFourCCStr("tiff"), //Tagged Image File Format
	getUintOfFourCCStr("raw "), //Uncompressed RGB
	getUintOfFourCCStr("2vuY"), //Uncompressed Y´CbCr, 8-bit-per-component 4:2:2
	getUintOfFourCCStr("yuv2"), //Uncompressed Y´CbCr, 8-bit-per-component 4:2:2
	getUintOfFourCCStr("v308"), //Uncompressed Y´CbCr, 8-bit-per-component 4:4:4
	getUintOfFourCCStr("v408"), //Uncompressed Y´CbCr, 8-bit-per-component 4:4:4:4
	getUintOfFourCCStr("v216"), //Uncompressed Y´CbCr, 10, 12, 14, or 16-bit-per-component 4:2:2
	getUintOfFourCCStr("v410"), //Uncompressed Y´CbCr, 10-bit-per-component 4:4:4
	getUintOfFourCCStr("v210"), //Uncompressed Y´CbCr, 10-bit-per-component 4:2:2
}
var drefInfoNums []uint32 = []uint32{
	getUintOfFourCCStr("alis"), //information about the file it refers to, including its full path name.
	getUintOfFourCCStr("rsrc"), //information documents existing content containing 'rsrc' data references; don’t use it for new development
	getUintOfFourCCStr("url "), //C string that specifies a URL. There may be additional data after the C string.
}

const atomHeaderSize uint32 = 8
const fullBoxSize uint32 = 4
const ftypSize uint32 = 20
const videoSampleDescSize int32 = 86

const russLangCode int16 = 32
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
const mdhdNum uint32 = 0x6d646864
const dinfNum uint32 = 0x64696e66
const hdlrNum uint32 = 0x68646c72
const stszNum uint32 = 0x7374737a
const sttsNum uint32 = 0x73747473
const vmhdNum uint32 = 0x766d6864
const colrNum uint32 = 0x636f6c72
const paspNum uint32 = 0x70617370
const fielNum uint32 = 0x6669656c
const drefNum uint32 = 0x64726566
const metaNum uint32 = 0x6d657461

const maxListPrint = 5

type UintX interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

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

func readQTMatrix(data []byte) ([9]float32, error) {
	const elemSize int = 4

	res := [9]float32{}
	if len(data) != 9 * elemSize {
		return res, fmt.Errorf("invalid data length to read transform matrix")
	}

	for i := range len(res) {
		var val uint32
		pos := i * elemSize
		binary.Read(bytes.NewReader(data[pos:pos+elemSize]), binary.BigEndian, &val)
		if (i+1) % 3 == 0 {
			res[i] = float32(fromFixedPoint(val, 2, 30))
		} else {
			res[i] = float32(fromFixedPoint(val, 16, 16))
		}
	}
	return res, nil
}

// converts fixed point 'src' represented with 'ic'.'fc' bits ratio to float
// 'ic' - integer bits count
// 'fc' - fractional bits count
func fromFixedPoint[T UintX](src T, ic, fc int) float64 {
	intact := float64(src >> ic)
	mask := T(1)
	for range fc-1 {
		mask <<= 1
		mask++
	}
	fract := float64(src & mask) / math.Pow(2, float64(fc))
	return intact + fract
}

// converts 'num' to uint representing fixed point 'ic'.'fc'
// 'ic' - integer bits count
// 'fc' - fractional bits count
func toFixedPoint[T UintX](num float64, ic, fc int) T {
	bitsCount := ic+fc
	if !slices.Contains([]int {8, 16, 32, 64}, bitsCount) {
		panic(fmt.Errorf(
			"provided fixed formula is not correct! sum of 'i' and 'fl' " +
			"should be equal to N of uint'N', but got: %d", bitsCount))
	}
	fractPart := num - float64(int64(num))
	minFract  := 1 / math.Pow(2, float64(fc))
	maxVal := int64(1) << bitsCount
	if num >= float64(maxVal) {
		panic(fmt.Errorf("overflow: %f cannot convert to number with %d bits", num, ic))
	}
	res := T(0)
	res |= T(num) << fc
	res |= T(fractPart / minFract)
	return res
}

func getMdat(data []byte) (Mdat, error) {
	const aHeaderFieldsSize = 16

	var mdat Mdat
	if len(data) < aHeaderFieldsSize {
		return mdat, fmt.Errorf(
			"too low amount of bytes received! " +
			"data length smaller than header size: %v < %v", len(data), aHeaderFieldsSize)
	}

	var size int32
	sizeBytes := copyBytes(&data, 4)
	err := binary.Read(bytes.NewReader(sizeBytes), binary.BigEndian, &size)
	if err != nil {
		return mdat, err
	}

	head, err := readAtomHeader(data)
	if err != nil {
		return mdat, err
	}

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
		AtomHeader: head,
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

func getEdts(data []byte) (*Edts, error) {
	header, err := readAtomHeader(data[:8])
	if err != nil {
		return nil, err
	}

	elstData := data[8:]
	elstHeader, err := readAtomHeader(elstData[:8])
	if err != nil {
		return nil, err
	}

	var elstFullBox FullBox
	err = binary.Read(bytes.NewReader(elstData[8:12]), binary.BigEndian, &elstFullBox)
	if err != nil {
		return nil, err
	}

	var elstNoe uint32
	err = binary.Read(bytes.NewReader(elstData[12:16]), binary.BigEndian, &elstNoe)
	if err != nil {
		return nil, err
	}

	elstEditList := make([]int32, 0)
	tableData := elstData[16:]
	for i := 0; i < len(tableData); i += 4 {
		var v int32
		err = binary.Read(bytes.NewReader(tableData[i:i+4]), binary.BigEndian, &v)
		if err != nil {
			return nil, fmt.Errorf("could not read value in EditList: %v", err)
		}
		elstEditList = append(elstEditList, v)
	}

	return &Edts{
		AtomHeader: header,
		Elst: Elst{
			AtomHeader: elstHeader,
			FullBox: elstFullBox,
			NOE: elstNoe,
			EditListTable: elstEditList,
		},
	}, nil
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
		AtomHeader: header,
		FullBox: FullBox{data[8], [3]byte{0,0,0}},
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

		if skipSize > uint64(len(content)) {
			panic(fmt.Sprint("skip size bigger than content length!", skipSize, ">", len(content)))
		}

		if ainfo {
			atomData := content[:skipSize]
			if slices.Contains(videoComprNums, atomHeader.Type) {
				var desc VideoSampleDescription
				err := binary.Read(bytes.NewReader(atomData[:videoSampleDescSize]), binary.BigEndian, &desc)
				if err != nil { panic(err) }

				comprnameb, err := GetBytesOf(desc.CompressorName)
				if err != nil { panic(err) }
				printWithIndent(fmt.Sprintf(
					"s: %v, dfmt: %v, dri: %v, w: %v, h: %v, vendor: %v, comp-r name: %v, colorId: %v, dsize: %v, ppi: %v %v, frameC: %v",
					desc.SampleDescriptionSize, uint32ToString(uint32(desc.DataFormat)), desc.DataReferenceIndex,
					desc.Width, desc.Height, uint32ToString(uint32(desc.Vendor)),
					string(comprnameb), desc.ColorTableID, desc.DataSize,
					fromFixedPoint(uint32(desc.HorizontalResolution), 16, 16), fromFixedPoint(uint32(desc.VerticalResolution), 16, 16),
					desc.FrameCount),
					dopInfoIndent)
				if int32(atomHeader.Size) > videoSampleDescSize {
					printAtoms(atomData[videoSampleDescSize:], indent+4, ainfo)
				}
			} else if slices.Contains(drefInfoNums, atomHeader.Type) {
				printWithIndent(fmt.Sprintf(
					"s: %v, version: %v, flags: %v, data: %v",
					atomHeader.Size, atomData[8], atomData[9:12], underscoreSpecials(string(atomData[12:]))),
					dopInfoIndent)
			}

			switch atomHeader.Type {
			case ftypNum:
				ftyp, err := getFtyp(atomData)
				if err != nil {
					panic(err)
				}
				ftypTxt := fmt.Sprintf("mb: %s, cmb: %s, mv: %d",
					uint32ToString(ftyp.MajorBrand), uint32ToString(ftyp.CompatibleBrands), ftyp.MinorVersion)
				printWithIndent(ftypTxt, dopInfoIndent)
			case mdatNum:
				mdat, err := getMdat(atomData)
				if err != nil {
					panic(err)
				}
				mdatTxt := fmt.Sprintf(
					"s: %d, es: %v, ds: %d", mdat.Size, mdat.ExtendedSize, len(mdat.Data))
				printWithIndent(mdatTxt, dopInfoIndent)
				if skipSize == 1 {
					skipSize = uint64(mdat.ExtendedSize)
				}
			case tkhdNum:
				var w, h, dur int32; var vol, layer, altGr int16
				binary.Read(bytes.NewReader(atomData[84:88]), binary.BigEndian, &w)
				binary.Read(bytes.NewReader(atomData[88:92]), binary.BigEndian, &h)
				binary.Read(bytes.NewReader(atomData[44:46]), binary.BigEndian, &vol)
				binary.Read(bytes.NewReader(atomData[28:32]), binary.BigEndian, &dur)
				binary.Read(bytes.NewReader(atomData[40:42]), binary.BigEndian, &layer)
				binary.Read(bytes.NewReader(atomData[42:44]), binary.BigEndian, &altGr)
				printWithIndent(fmt.Sprintf(
					"w: %v, h: %v, volume: %v, dur: %v, lay: %v, alt gr: %v",
					fromFixedPoint(uint32(w), 16, 16),
					fromFixedPoint(uint32(h), 16, 16),
					fromFixedPoint(uint16(vol), 8, 8), dur, layer, altGr), dopInfoIndent)
				matr, err := readQTMatrix(atomData[48:48+36])
				if err != nil { fmt.Println("warning: could not read matrix dur to err", err) }
				printWithIndent(fmt.Sprintf("%v", matr), dopInfoIndent)
			case hdlrNum:
				compType, err := GetBytesOf(content[12:16])
				if err != nil { return }
				compSubType, err := GetBytesOf(content[16:20])
				if err != nil { return }
				compName, err := GetBytesOf(content[32:skipSize])
				if err != nil { return }

				printWithIndent(
					fmt.Sprintf("compType: %s, compSubType: %s, compName: %s",
						compType, compSubType, underscoreSpecials(string(compName))),
					dopInfoIndent)
			case vmhdNum:
				var vmhd Vmhd
				binary.Read(bytes.NewReader(atomData), binary.BigEndian, &vmhd)
				printWithIndent(
					fmt.Sprintf("graphics mode: %v, opcolor: %v", vmhd.GraphicsMode, vmhd.Opcolor),
					dopInfoIndent)
			case mvhdNum:
				var mvhd Mvhd
				binary.Read(bytes.NewReader(atomData), binary.BigEndian, &mvhd)
				ts := mvhd.TimeScale
				if ts == 0 {
					ts = 1
				}
				durationSec := float32(mvhd.Duration) / float32(ts)
				matr, err := readQTMatrix(atomData[34:70])
				if err != nil { fmt.Printf("warning: could not read transform matrix due to err: %v", err) }
				printWithIndent(
					fmt.Sprintf("duration: %fs, timeScale: %d, matrix: %v", durationSec, mvhd.TimeScale, matr),
					dopInfoIndent)
			case mdhdNum:
				var mdhd Mdhd
				err := binary.Read(bytes.NewReader(content), binary.BigEndian, &mdhd)
				if err != nil { panic(err) }
				printWithIndent(
					fmt.Sprintf(
						"creat time: %v, mod date: %v, ts: %v, dur: %v, lang: %v, qual: %v",
						mdhd.CreationTime, mdhd.Modification, mdhd.TimeScale, mdhd.Duration, mdhd.Language, mdhd.Quality),
					dopInfoIndent)
			case _swrNum:
				sgi := content[8:skipSize]
				sgiTxt := underscoreSpecials(string(sgi))
				printWithIndent(fmt.Sprintf("software generated info: %s", sgiTxt), dopInfoIndent)
			case stsdNum:
				table := atomData[16:atomHeader.Size]
				if len(table) > 0 {
					printAtoms(table, nextLevelIndent, ainfo)
				}
			case fielNum:
				var fiel Fiel
				binary.Read(bytes.NewReader(atomData), binary.BigEndian, &fiel)
				printWithIndent(
					fmt.Sprintf("count: %v, ordering: %v", fiel.Count, fiel.Order),
					dopInfoIndent)
			case colrNum:
				var colr Colr
				binary.Read(bytes.NewReader(atomData), binary.BigEndian, &colr)
				printWithIndent(
					fmt.Sprintf(
						"param type: %v, pind: %v, tind: %v, mind: %v",
						uint32ToString(colr.ColorParamType), colr.PrimariesIndex, colr.TransferFunctionIndex, colr.MatrixIndex),
					dopInfoIndent)
			case paspNum:
				var pasp Pasp
				binary.Read(bytes.NewReader(atomData), binary.BigEndian, &pasp)
				printWithIndent(
					fmt.Sprintf("hspacing: %v, vspacing: %v", pasp.HSpacing, pasp.VSpacing),
					dopInfoIndent)
			case sttsNum:
				var noe uint32
				binary.Read(bytes.NewReader(atomData[12:16]), binary.BigEndian, &noe)
				printWithIndent(fmt.Sprintf("noe: %v", noe), dopInfoIndent)
				time2Sample := atomData[16:]
				count := len(time2Sample) / 8
				if count > 0 {
					for i := range min(maxListPrint, count) {
						var sampleCount, sampleDur int32
						binary.Read(bytes.NewReader(time2Sample[i*8:(i+1)*8]), binary.BigEndian, &sampleCount)
						binary.Read(bytes.NewReader(time2Sample[(i*8 + 4):((i+1)*8 + 4)]), binary.BigEndian, &sampleDur)
						printWithIndent(
							fmt.Sprintf("count: %v, dur: %v", sampleCount, sampleDur),
							dopInfoIndent)
					}
					if count > maxListPrint {
						printWithIndent(fmt.Sprintf("... and + %v lines", count - maxListPrint), dopInfoIndent)
					}
				} else {
					printWithIndent("no data (table is empyt)", dopInfoIndent)
				}
			case stscNum:
				stsc, err := getStsc(atomData)
				if err != nil {
					printWithIndent(fmt.Sprintf("%v", err), indent)
					return
				}
				printWithIndent(fmt.Sprintf("noe: %v", stsc.NOE), dopInfoIndent)
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
			case stszNum:
				var sampleSize, noe uint32
				binary.Read(bytes.NewReader(atomData[12:16]), binary.BigEndian, &sampleSize)
				binary.Read(bytes.NewReader(atomData[16:20]), binary.BigEndian, &noe)
				sizesData := atomData[20:skipSize]
				printWithIndent(fmt.Sprintf("sample size: %v, noe: %v", sampleSize, noe), dopInfoIndent)
				if len(sizesData) > 0 {
					for i := range min(maxListPrint, len(sizesData) / 4) {
						var sz uint32
						binary.Read(bytes.NewReader(sizesData[i*4:(i+1)*4]), binary.BigEndian, &sz)
						printWithIndent(fmt.Sprintf("%d sample size: %v", i, sz), dopInfoIndent)
					}
					if noe > maxListPrint {
						printWithIndent(fmt.Sprintf("... and + %v lines", noe - maxListPrint), dopInfoIndent)
					}
				} else {
					printWithIndent("sizes are equal (table is empty)", dopInfoIndent)
				}
			case stcoNum:
				stco, err := getStco(atomData)
				if err != nil {
					printWithIndent(fmt.Sprintf("%v", err), indent)
					return
				}

				printWithIndent(fmt.Sprintf("noe: %v", stco.NOE), dopInfoIndent)
				count := len(stco.ChunkOffsets)
				if count > 0 {
					for i := range min(maxListPrint, count) {
						printWithIndent(fmt.Sprintf("offset: %v", stco.ChunkOffsets[i]), dopInfoIndent)
					}
					if count > maxListPrint {
						printWithIndent(fmt.Sprintf("... and + %v lines", count - maxListPrint), dopInfoIndent)
					}
				}
			case edtsNum:
				edts, err := getEdts(atomData)
				if err != nil {
					printWithIndent(fmt.Sprintf("%v", err), indent)
					return
				}
				printWithIndent(fmt.Sprintf("noe: %v", edts.Elst.NOE), dopInfoIndent)
				table := edts.Elst.EditListTable
				count := len(table) / 3
				if count > 0 {
					for i := range min(maxListPrint, count) {
						tdur  := table[i*3]
						mtime := table[i*3 + 1]
						mrate := fromFixedPoint(uint32(table[i*3 + 2]), 16, 16)
						printWithIndent(
							fmt.Sprintf("trak dur: %v media time: %v, media rate: %v", tdur, mtime, mrate),
							dopInfoIndent)
					}
					if count > maxListPrint {
						printWithIndent(fmt.Sprintf("... and + %v lines", count - maxListPrint), dopInfoIndent)
					}
				} else { printWithIndent("Warning: there was no values!", dopInfoIndent)}
			case drefNum:
				printWithIndent(
					fmt.Sprintf("ver: %v, flags: %v, noe: %d",
						atomData[8], atomData[9:12], atomData[12:16]), dopInfoIndent)
				printAtoms(atomData[16:], nextLevelIndent, ainfo)
			}
		}

		if slices.Contains(parentsAtoms, atomHeader.Type) {
			printAtoms(content[atomHeaderSize:skipSize], nextLevelIndent, ainfo)
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

func getUintOfFourCCStr(s string) uint32 {
	if len(s) != 4 { panic("fourcc string must be contains 4 sybmols") }
	b := make([]byte, 0)
	for _, ch := range s {
		b = append(b, byte(ch))
	}
	var res uint32
	err := binary.Read(bytes.NewReader(b), binary.BigEndian, &res)
	if err != nil {
		panic(err)
	}
	return res
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
		if unicode.IsDigit(r) || unicode.IsLetter(r) || unicode.IsPunct(r) {
			return r
		}
		return '_'
	}

	return strings.Map(f, s)
}

func IsLittleEndianPlatform() bool {
    return binary.NativeEndian.Uint16([]byte{0x12, 0x34}) == 0x3412
}

func GetBytesOf[T any](s T) ([]byte, error) {
	res := make([]byte, 0)
	srcValue := reflect.ValueOf(s)
	kind := srcValue.Kind()

	if kind != reflect.Struct {
		res, err := GetBytesOfTypeValue(srcValue.Type(), srcValue)
		if err != nil { return nil, err }
		return res, nil
	}

	for field, val := range srcValue.Fields() {
		b, err := GetBytesOfTypeValue(field.Type, val)
		if err != nil { return nil, err }

		res = slices.Concat(res, b)
	}

	return res, nil
}

func GetBytesOfTypeValue(typ reflect.Type, val reflect.Value) ([]byte, error) {
	var b []byte
	var err error

	switch typ.Kind() {
	case reflect.Slice:
		if typ.AssignableTo(reflect.TypeFor[[]byte]())  {
			b = val.Bytes()
			break
		}
		for _, elem := range val.Seq2() {
			eb, err := GetBytesOfTypeValue(elem.Type(), elem)
			if err != nil { return nil, err }
			b = slices.Concat(b, eb)
		}
	case reflect.Array:
		for _, elem := range val.Seq2() {
			eb, err := GetBytesOfTypeValue(elem.Type(), elem)
			if err != nil { return nil, err }
			b = slices.Concat(b, eb)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		 reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		 reflect.Float32, reflect.Float64:
		b, err = GetBytesOfNumber(val)
		if IsLittleEndianPlatform() { slices.Reverse(b) }
		if err != nil { return nil, err }
	case reflect.Struct:
		b, err = GetBytesOfAtomStruct(val)
		if err != nil { return nil, err }
	case reflect.String:
		sl := val.Slice(0, val.Len())
		elemSize := 1
		b = unsafe.Slice((*byte)(sl.UnsafePointer()), sl.Len() * int(elemSize))
		if IsLittleEndianPlatform() {
			for i := range sl.Len() {
				slices.Reverse(b[int(elemSize) * i : int(elemSize) * (i+1)])
			}
		}
	default:
		return nil, errors.New(
			"got unsupported type for bytes extracting: " + typ.Kind().String())
	}

	return b, nil
}

func GetBytesOfNumber(val reflect.Value) ([]byte, error) {
	var b []byte
	var err error
	switch val.Type().Kind() {
	case reflect.Int:
		v := val.Int()
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Int8:
		v := int8(val.Int())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Int16:
		v := int16(val.Int())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Int32:
		v := int32(val.Int())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Int64:
		v := int64(val.Int())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Uint:
		v := val.Uint()
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Uint8:
		v := uint8(val.Uint())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Uint16:
		v := uint16(val.Uint())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Uint32:
		v := uint32(val.Uint())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Uint64:
		v := uint64(val.Uint())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Float32:
		v := float32(val.Float())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	case reflect.Float64:
		v := float64(val.Float())
		b = unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	}

	return b, err
}

func GetBytesOfAtomStruct(val reflect.Value) ([]byte, error) {
	var b []byte
	var err error
	switch val.Type().String() {
	case "main.AtomHeader":
		s := val.Interface().(AtomHeader)
		b, err = GetBytesOf(s)
		if err != nil {
			return nil, err
		}
	case "main.FullBox":
		s := val.Interface().(FullBox)
		b, err = GetBytesOf(s)
		if err != nil {
			return nil, err
		}
	case "main.SampleChunkRow":
		s := val.Interface().(SampleChunkRow)
		b, err = GetBytesOf(s)
		if err != nil {
			return nil, err
		}
	default:
		for field, val := range val.Fields() {
			cb, err := GetBytesOfTypeValue(field.Type, val)
			if err != nil { 
				return nil, errors.New(
					"got unsupported atom type for bytes extracting: " + val.Type().Kind().String())
			}
			b = slices.Concat(b, cb)
		}
	}

	return b, err
}

func writeDemo(file *os.File) error {
	brand := getUintOfFourCCStr("qt  ")
	ftyp, err := GetBytesOf(Ftyp{AtomHeader{ftypSize, ftypNum}, brand, 512, brand})
	if err != nil { return err }
	file.Write(ftyp)

	wide, err := GetBytesOf(AtomHeader{8, wideNum})
	if err != nil { return err }
	file.Write(wide)

	h := 720
	w := 1280
	fps := int32(30)
	totalFrames := int32(60)
	oneFrameSize := int32(w*h*3)
	timeScale := int32(1000)
	durationSec := int32(2)
	contentSize := oneFrameSize * totalFrames
	content := make([]byte, contentSize)
	for i := range len(content) {
		var v byte
		if i > len(content) / 2 {
			v = 100
		} else {
			v = 200
		}
		content[i] = v
	}

	mdat, err := GetBytesOf(Mdat{AtomHeader{0, mdatNum}, 0, content})
	if err != nil { return err }
	mdatSize, err := GetBytesOf(uint32(len(mdat)))
	if err != nil { return err }
	mdat = slices.Replace(mdat, 0, 4, mdatSize...)
	file.Write(mdat)

	tkhd, err := GetBytesOf(Tkhd{
		AtomHeader{0, tkhdNum},
		FullBox{0, [3]byte{0,0,0}},
		0,
		0,
		0,
		0,
		durationSec * timeScale,
		[8]byte{},
		0,
		0,
		0,
		0,
		[36]byte{0,1,0,0, 0,0,0,0, 0,0,0,0, 0,0,0,0, 0,1,0,0, 0,0,0,0, 0,0,0,0, 0,0,0,0, 64,0,0,0}, // as in example
		int32(int32(w) << 16),
		int32(int32(h) << 16),
	})
	if err != nil { return err }
	tkhdSize, err := GetBytesOf(uint32(len(tkhd)))
	if err != nil { return err }
	tkhd = slices.Replace(tkhd, 0, 4, tkhdSize...)

	hdlrNum := getUintOfFourCCStr("hdlr")
	compTypeNum := getUintOfFourCCStr("mhlr")
	url_Num := getUintOfFourCCStr("vide") // or 'soun', 'subt'
	compName, err := GetBytesOf("VideoHandler") // \012
	hdlr, err := GetBytesOf(Hdlr{
		AtomHeader{0, hdlrNum},
		FullBox{0, [3]byte{0,0,0}},
		compTypeNum,
		url_Num,
		[4]byte{}, [4]byte{}, [4]byte{},
		compName,
	})
	if err != nil { return err }
	hdlrSize, err := GetBytesOf(uint32(len(hdlr)))
	if err != nil { return err }
	hdlr = slices.Replace(hdlr, 0, 4, hdlrSize...)

	editListTable := []int32{
		totalFrames / fps * timeScale, // trak dur
		0,       // media time
		1 << 16, // media rate
	}
	elstNum := getUintOfFourCCStr("elst")
	elst := Elst{
		AtomHeader{atomHeaderSize + fullBoxSize + 4 + uint32(len(editListTable))*4, elstNum},
		FullBox{},
		1,
		editListTable,
	}
	edts, err := GetBytesOf(Edts{AtomHeader{0, edtsNum}, elst})
	if err != nil { return err }
	edtsSize, err := GetBytesOf(uint32(len(edts)))
	if err != nil { return err }
	edts = slices.Replace(edts, 0, 4, edtsSize...)

	trakTS := int32(30_000)
	mdhd, err := GetBytesOf(Mdhd{
		AtomHeader{0, mdhdNum},
		FullBox{0, [3]byte{0,0,0}},
		0,
		0,
		trakTS,
		durationSec * trakTS,
		32767, // from example video
		0,
	}) 
	if err != nil { return err }
	mdhd = slices.Concat(mdhd, make([]byte, 12)) // ffprobe said "overread" without it
	mdhdSize, err := GetBytesOf(uint32(len(mdhd)))
	if err != nil { return err }
	mdhd = slices.Replace(mdhd, 0, 4, mdhdSize...)

	meta := []byte{}
	fiel, err := GetBytesOf(Fiel{AtomHeader{10, fielNum}, 1, 0})
	if err != nil { return err }
	meta = slices.Concat(meta, fiel)

	colr, err := GetBytesOf(Colr{
		AtomHeader{0, colrNum},
		getUintOfFourCCStr("nclc"),
		1,
		1,
		0,
	})
	if err != nil { return err }
	colrSize, err := GetBytesOf(uint32(len(colr)))
	if err != nil { return err }
	colr = slices.Replace(colr, 0, 4, colrSize...)
	meta = slices.Concat(meta, colr)

	pasp, err := GetBytesOf(Pasp{
		AtomHeader{0, paspNum},
		1, 1,
	})
	if err != nil { return err }
	paspSize, err := GetBytesOf(uint32(len(pasp)))
	if err != nil { return err }
	pasp = slices.Replace(pasp, 0, 4, paspSize...)
	meta = slices.Concat(meta, pasp)
	videoSampleDesc, err := GetBytesOf(VideoSampleDescription{
		0,
		int32(raw_Num),
		[6]byte{},
		1,
		0,
		0,
		int32(getUintOfFourCCStr("UNKN")),
		0,
		0,
		int16(w), int16(h),
		72 << 16, 72 << 16,
		0,
		1, // frame count
		[32]byte{22,76,97,118,99,54,50,46,49,49,46,49,48,48,32,114,97,119,118,105,100,101,111,0,0,0,0,0,0,0,0,0}, // from example video 
		24,
		-1,
	})
	if err != nil { return err }

	videoSampleDesc = slices.Concat(videoSampleDesc, meta)
	videoSampleDescSize, err := GetBytesOf(uint32(len(videoSampleDesc)))
	if err != nil { return err }
	videoSampleDesc = slices.Replace(videoSampleDesc, 0, 4, videoSampleDescSize...)

	if err != nil { return err }
	stsd, err := GetBytesOf(Stsd{
		AtomHeader{0, stsdNum},
		FullBox{0, [3]byte{0,0,0}},
		1,
		videoSampleDesc,
	})
	if err != nil { return err }
	stsdSize, err := GetBytesOf(uint32(len(stsd)))
	if err != nil { return err }
	stsd = slices.Replace(stsd, 0, 4, stsdSize...)

	firstOffset := int32(len(ftyp) + len(wide) + int(atomHeaderSize))
	offsets := make([]int32, totalFrames)
	for i := range len(offsets) {
		offsets[i] = firstOffset + int32(i) * oneFrameSize
	}
	stco, err := GetBytesOf(Stco{
		AtomHeader{0, stcoNum},
		FullBox{0, [3]byte{0,0,0}},
		uint32(totalFrames),
		offsets,
	})
	if err != nil { return err }
	stcoSize, err := GetBytesOf(uint32(len(stco)))
	if err != nil { return err }
	stco = slices.Replace(stco, 0, 4, stcoSize...)

	sttsNum := getUintOfFourCCStr("stts")
	stts, err := GetBytesOf(Stts{
		AtomHeader{0, sttsNum},
		FullBox{0, [3]byte{0,0,0}},
		1,
		[]int32{totalFrames, trakTS / fps},
	})
	if err != nil { return err }
	sttsSize, err := GetBytesOf(uint32(len(stts)))
	if err != nil { return err }
	stts = slices.Replace(stts, 0, 4, sttsSize...)

	stscNum := getUintOfFourCCStr("stsc")
	stsc, err := GetBytesOf(Stsc{
		AtomHeader{0, stscNum},
		FullBox{0, [3]byte{0,0,0}},
		1,
		[]SampleChunkRow{ {1, 1, 1}, },
	})
	if err != nil { return err }
	stscSize, err := GetBytesOf(uint32(len(stsc)))
	if err != nil { return err }
	stsc = slices.Replace(stsc, 0, 4, stscSize...)

	stsz, err := GetBytesOf(Stsz{
		AtomHeader{0, stszNum},
		FullBox{0, [3]byte{0,0,0}},
		uint32(oneFrameSize),
		uint32(totalFrames),
		[]uint32 {}, // because all sample have same size
	})
	if err != nil { return err }
	stszSize, err := GetBytesOf(uint32(len(stsz)))
	if err != nil { return err }
	stsz = slices.Replace(stsz, 0, 4, stszSize...)

	stbl, err := GetBytesOf(struct {
		Head AtomHeader
		Stsd []byte
		Stts []byte
		Stsc []byte
		Stsz []byte
		Stco []byte
	}{
		AtomHeader{0, stblNum}, // atomHeaderSize + uint32(len(stsd) + ...)
		stsd,
		stts,
		stsc,
		stsz,
		stco,
	})
	if err != nil { return err }
	stblSize, err := GetBytesOf(uint32(len(stbl)))
	if err != nil { return err }
	stbl = slices.Replace(stbl, 0, 4, stblSize...)

	vmhdNum := getUintOfFourCCStr("vmhd")
	vmhd, err := GetBytesOf(Vmhd{
		AtomHeader{0, vmhdNum},
		FullBox{0, [3]byte{0,0,0}},
		0, [3]int16{0, 0, 0},
	})
	if err != nil { return err }
	vmhdSize, err := GetBytesOf(uint32(len(vmhd)))
	if err != nil { return err }
	vmhd = slices.Replace(vmhd, 0, 4, vmhdSize...)

	compTypeNum = getUintOfFourCCStr("dhlr")
	url_Num = getUintOfFourCCStr("url ")
	compName, err = GetBytesOf("DataHandler") // \011
	minfHdlr, err := GetBytesOf(Hdlr{
		AtomHeader{0, hdlrNum},
		FullBox{0, [3]byte{0,0,0}},
		compTypeNum,
		url_Num,
		[4]byte{}, [4]byte{}, [4]byte{},
		compName,
	})
	if err != nil { return err }
	minfHdlrSize, err := GetBytesOf(uint32(len(minfHdlr)))
	if err != nil { return err }
	minfHdlr = slices.Replace(minfHdlr, 0, 4, minfHdlrSize...)

	dref, err := GetBytesOf(Dref{
		AtomHeader{8 + 4 + 4 + 12, drefNum},
		FullBox{},
		1,
		[]DatRef{{
			AtomHeader{12, url_Num},
			FullBox{0, [3]byte{0,0,1}},
			[]byte{},
		}},
	})

	dinf, err := GetBytesOf(struct {
		Head AtomHeader
		Dref []byte
	} {
		AtomHeader{uint32(8 + len(dref)), dinfNum},
		dref,
	})

	minf, err := GetBytesOf(struct {
		Head AtomHeader
		Vmhd []byte
		Hdlr []byte
		Dinf []byte
		Stbl []byte
	}{
		AtomHeader{0, minfNum},
		vmhd,
		minfHdlr,
		dinf,
		stbl,
	})
	if err != nil { return err }
	minfSize, err := GetBytesOf(uint32(len(minf)))
	if err != nil { return err }
	minf = slices.Replace(minf, 0, 4, minfSize...)

	mdia, err := GetBytesOf(struct {
		Head AtomHeader
		Mdhd []byte
		Hdlr []byte
		Minf []byte
	}{
		AtomHeader{atomHeaderSize + uint32(len(mdhd) + len(minf) + len(hdlr)), mdiaNum},
		mdhd,
		hdlr,
		minf,
	})
	if err != nil { return err }

	trak, err := GetBytesOf(struct {
		Head AtomHeader
		Tkhd []byte
		Edts []byte
		Mdia []byte
	}{
		AtomHeader{atomHeaderSize + uint32(len(tkhd) + len(edts) + len(mdia)), trakNum},
		tkhd,
		edts,
		mdia,
	})
	if err != nil { return err }

	mvhd, err := GetBytesOf(Mvhd{
		AtomHeader{0, mvhdNum},
		FullBox{0, [3]byte{0,0,0}},
		0, // CreationTime  (версия 0) или 8 байт (версия 1)
		0, // ModificationTime  (версия 0) или 8 байт (версия 1)
		uint32(timeScale), // TimeScale  временная шкала в тиках/секунду
		uint32(durationSec * timeScale), // Duration  (версия 0) или 8 байт (версия 1)
		0x00010000, // PreferredRate  16.16 фиксированная точка (1.0 = 0x00010000)
		0x0100, // PreferredVolume  8.8 фиксированная точка (1.0 = 0x0100)
		[9]int32{0,0,1,0,0,0,1,0,0}, // Matrix  матрица преобразования 3x3
		0, // PreviewTime
		0, // PreviewDuration
		0, // PosterTime
		0, // SelectionTime
		0, // SelectionDuration
		0, // CurrentTime
		0, // NextTrackID  - следующий доступный ID трека
	})
	if err != nil { return err }
	mvhd = slices.Concat(mvhd, make([]byte, 10))
	mvhdSize, err := GetBytesOf(uint32(len(mvhd)))
	if err != nil { return err }
	mvhd = slices.Replace(mvhd, 0, 4, mvhdSize...)

	udta, err := GetBytesOf(struct {
		Head AtomHeader
		List []byte
	} {
		AtomHeader{0, udtaNum},
		[]byte{0,0,0,24,169,115,119,114,0,12,85,196,76,97,118,102,54,50,46,51,46,49,48,48}, // from example video
	})
	if err != nil { return err }
	udtaSize, err := GetBytesOf(uint32(len(udta)))
	if err != nil { return err }
	udta = slices.Replace(udta, 0, 4, udtaSize...)

	moov, err := GetBytesOf(struct {
		Head AtomHeader
		Mvhd []byte
		Trak []byte
		// Udta []byte
	} {
		AtomHeader{0, moovNum},
		mvhd,
		trak,
		// udta,
	})
	if err != nil { return err }
	moovSize, err := GetBytesOf(uint32(len(moov)))
	if err != nil { return err }
	moov = slices.Replace(moov, 0, 4, moovSize...)

	file.Write(moov)
	fmt.Println("writing end up successfully!")
	return nil
}

func panicIfErr(err error) { if err != nil { panic(err) } }

func main() {
	flag.Usage = func() {
		fmt.Printf(`
hack-mov [options] mode

There are 2 modes of this script:
    explore: prints all atoms, and additional info if -ainfo specified
    export:  exports all video frames into created directory in calling directory
	write:   write demo result.mov

Example:
  hack-mov -f video.mov -ainfo explore
  hack-mov write

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

	switch mode {
	case "explore":
		if filePath == nil {
			panic("no file path was provided!")
		}
		content, err := os.ReadFile(*filePath)
		if err != nil {
			panic(err)
		} else if len(content) == 0 {
			panic(fmt.Errorf("nothing to explore, file is empty"))
		}
		printAtoms(content, 0, *ainfo)
	case "export":
		content, err := os.ReadFile(*filePath)
		if err != nil {
			panic(err)
		}
		imgsDir := "./imgs-" + path.Base(*filePath)
		exportFrames(content, imgsDir)
	case "write":
		file, err := os.Create("result.mov")
		defer file.Close()
		err = writeDemo(file)
		panicIfErr(err)
	default:
		panic("got wrong mode!")
	}
}
