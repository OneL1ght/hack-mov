package main

// Базовый заголовок любого атома (8 байт)
type AtomHeader struct {
	Size uint32   // Размер всего атома в байтах (включая заголовок)
	Type uint32   // Тип атома в ASCII ('moov', 'mvhd', 'trak' и т.д.)
}

// FullBox - расширенный формат атома (начинается с version + flags)
type FullBox struct {
	Version uint8    // Версия формата (0 или 1)
	Flags   [3]byte  // Флаги (3 байта)
}

type Mvhd struct {
	AtomHeader
	FullBox
	CreationTime      uint32   // (версия 0) или 8 байт (версия 1)
	ModificationTime  uint32   // (версия 0) или 8 байт (версия 1)
	TimeScale         uint32   // временная шкала в тиках/секунду
	Duration          uint32   // (версия 0) или 8 байт (версия 1)
	PreferredRate     uint32   // 16.16 фиксированная точка (1.0 = 0x00010000) 4 * 9
	PreferredVolume   uint16   // 8.8 фиксированная точка (1.0 = 0x0100)       2
	Matrix            [9]int32 // матрица преобразования 3x3                   36
	PreviewTime       uint32
	PreviewDuration   uint32
	PosterTime        uint32
	SelectionTime     uint32
	SelectionDuration uint32
	CurrentTime       uint32
	NextTrackID       uint32   // - следующий доступный ID трека               4 * 7
}

type Ftyp struct {
	AtomHeader
	MajorBrand       uint32 // A 32-bit unsigned integer that represents a file format code.
	MinorVersion     uint32 // A 32-bit field that indicates the file format specification version.
	CompatibleBrands uint32 // A series of unsigned 32-bit integers listing compatible file formats.
}

type Mdat struct {
	AtomHeader
	ExtendedSize int64  // A 64-bit integer that specifies the number of bytes in this media data atom.
	Data         []byte // content
}

type Tkhd struct {
	AtomHeader
	FullBox
	CreationTime     int32
	ModificationTime int32
	TrackID          int32
	_                int32
	Duration         int32
	_                [8]byte
	Layer            int16 // A 16-bit integer that indicates this track’s spatial priority in its movie.
	AlternateGroup   int16 //  identifies a collection of movie tracks that contain alternate data for one another
	Volume           int16 // fixed point int16
	_                int16
	MatrixStructure  [36]byte
	TrackWidth       int32 // fixed point int32, the width of this track in pixels
	TrackHeight      int32 // fixed point int32, the height of this track in pixels
}

type Edts struct {
	AtomHeader
	Elst
}

// An atom that maps from a time in a movie to a time in a media,
// and ultimately to media data.
type Elst struct {
	AtomHeader
	FullBox
	NOE           uint32
	EditListTable []int32 // grouped into entries containing 3 values each [trak duration, media time, media rate]
}

type Stco struct {
	AtomHeader
	FullBox
	NOE          uint32
	ChunkOffsets []int32 // offsets list in chunks order [o1, o2..], means [ch1, ch2..]
}

// stores duration information for a media’s samples, providing
// a mapping from a time in a media to the corresponding data sample
type Stts struct {
	AtomHeader
	FullBox
	NOE int32
	Time2Sample []int32 // table with 2 int32 values in raw [['sample count', 'sample duration'],...]
}

// stores chunk information for the samples in a media
type Stsc struct {
	AtomHeader
	FullBox
	NOE          uint32
	Sample2Chunk []SampleChunkRow
}

type SampleChunkRow struct {
	First int32
	SpC   int32 // sample per chunk
	Id    int32
}

// to specify the size of each sample in the media
type Stsz struct {
	AtomHeader
	FullBox
	SampleSize uint32
	NOE        uint32
	SampleSizeTable []uint32 // ['sizeInBytesSample1', 'sizeInBytesSample2', ...]
}

type Mdhd struct {
	AtomHeader
	FullBox // One byte that specifies the version of this header atom, Three bytes of space for media header flags.
	CreationTime int32   // specifies the creation date for the media atom.
	Modification int32   // specifies the last modification date for the media atom.
	TimeScale    int32   // scale A time value that indicates the time scale for this media.
	Duration     int32   // The duration of this media in units of its time scale.
	Language     int16   // specifies the language code for this media.
	Quality      int16   // specifies the media’s playback quality.
}

type Stsd struct {
	AtomHeader
	FullBox // specification of the version of this sample description atom, A 3-byte space for sample description flags.
	NOE         int32   // containing the number of sample descriptions that follow.
	SampleTable []SampleDescRow // An array of sample descriptions
}

type SampleDescRow struct {
	Size         int32
	DataFmt      int32
	_            [6]byte
	DataRefIndex int16
	Description  []byte
}

type Hdlr struct {
	AtomHeader
	FullBox // specification of the version of this handler information, space for handler information flags.
	ComponentType    uint32  // fourcc that identifies the type of the handler.
	ComponentSubtype uint32  // fourcc that identifies the type of the media handler or data handler.
	_                [4]byte // Reserved.
	_                [4]byte // Reserved.
	_                [4]byte // Reserved.
	ComponentName    []byte  // A counted string that specifies the name of the component — that is, the media handler used when this media was created.
}

type Vmhd struct {
	AtomHeader
	FullBox
	GraphicsMode int16
	Opcolor      [3]int16 // rgb for transfer mode orientation
}
