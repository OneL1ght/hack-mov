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
	CreationTime      uint32   // 4 байта (версия 0) или 8 байт (версия 1)
	ModificationTime  uint32   // 4 байта (версия 0) или 8 байт (версия 1)
	TimeScale         uint32   // 4 байта - временная шкала в тиках/секунду
	Duration          uint32   // 4 байта (версия 0) или 8 байт (версия 1)
	PreferredRate     uint32   // 4 байта - 16.16 фиксированная точка (1.0 = 0x00010000)
	PreferredVolume   uint16   // 2 байта - 8.8 фиксированная точка (1.0 = 0x0100)
	Matrix            [9]int32 // 36 байт - матрица преобразования 3x3
	PreviewTime       uint32   // 4 байта
	PreviewDuration   uint32   // 4 байта
	PosterTime        uint32   // 4 байта
	SelectionTime     uint32   // 4 байта
	SelectionDuration uint32   // 4 байта
	CurrentTime       uint32   // 4 байта
	NextTrackID       uint32   // 4 байта - следующий доступный ID трека
}

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
