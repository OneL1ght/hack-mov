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

type MovieHeaderAtom struct {
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
	Layer            int16
	AlternateGroup   int16
	Volume           [2]int8 // fixed point int16
	_                int16
	MatrixStructure  [36]byte
	TrackWidth       [2]int16 // fixed point int32
	TrackHeight      [2]int16 // fixed point int32
}

type Edts struct {
	AtomHeader
	Elst
}

type Elst struct {
	AtomHeader
	FullBox
	NOE           uint32
	EditListTable []int32 // An array of 32-bit values, grouped into entries containing 3 values each.
}

type Stco struct {
	AtomHeader
	FullBox
	NOE          uint32
	ChunkOffsets []int32 // offsets list in chunks order [o1, o2..], meand [ch1, ch2..]
}

type Stsc struct {
	Size         uint32
	Type         uint32
	Version      byte
	Flags        [3]byte
	NOE          uint32
	Sample2Chunk []SampleChunkRow
}

type SampleChunkRow struct {
	First int32
	SpC   int32 // sample per chunk
	Id    int32
}
