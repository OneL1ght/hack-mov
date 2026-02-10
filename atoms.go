package main

type Atom interface {}

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

// ============================================================================
// Атомы верхнего уровня внутри moov
// ============================================================================

// MovieAtom - контейнерный атом, описывающий структуру фильма
type MovieAtom struct {
	AtomHeader
	Children []Atom // Дочерние атомы: mvhd, trak*, udta, mvex и др.
}

// MovieHeaderAtom - заголовок фильма (обязательный внутри moov)
type MovieHeaderAtom struct {
	AtomHeader
	FullBox
	CreationTime      uint32   // 4 байта (версия 0) или 8 байт (версия 1)
	ModificationTime  uint32   // 4 байта (версия 0) или 8 байт (версия 1)
	TimeScale         uint32   // 4 байта - временная шкала в тиках/секунду
	Duration          uint32   // 4 байта (версия 0) или 8 байт (версия 1)
	PreferredRate     uint32   // 4 байта - 16.16 фиксированная точка (1.0 = 0x00010000)
	PreferredVolume   uint16   // 2 байта - 8.8 фиксированная точка (1.0 = 0x0100)
	// Reserved          [10]byte // 10 зарезервированных байт (не "Reserved1" как отдельное поле)
	Matrix            [9]int32 // 36 байт - матрица преобразования 3x3
	PreviewTime       uint32   // 4 байта
	PreviewDuration   uint32   // 4 байта
	PosterTime        uint32   // 4 байта
	SelectionTime     uint32   // 4 байта
	SelectionDuration uint32   // 4 байта
	CurrentTime       uint32   // 4 байта
	NextTrackID       uint32   // 4 байта - следующий доступный ID трека
}

// TrackAtom - контейнер для одного трека (аудио, видео, текст и т.д.)
type TrackAtom struct {
	AtomHeader
	Children []Atom // Дочерние атомы: tkhd, edts, mdia, udta и др.
}

// TrackHeaderAtom - заголовок трека (обязательный внутри trak)
// type TrackHeaderAtom struct {
// 	FullAtom
// 	CreationTime    uint64
// 	ModificationTime uint64
// 	TrackID         uint32
// 	Reserved1       uint32
// 	Duration        uint64 // Длительность трека в тиках временной шкалы
// 	Reserved2       [8]byte
// 	Layer           int16  // Порядок наложения (выше = ближе к зрителю)
// 	AlternateGroup  int16  // Группа альтернативных треков
// 	Volume          int16  // Громкость трека (для аудио)
// 	Reserved3       uint16
// 	Matrix          [9]int32 // Матрица преобразования
// 	Width           uint32   // Ширина кадра в фиксированной точке (16.16)
// 	Height          uint32   // Высота кадра в фиксированной точке (16.16)
// }
//
// // EditAtom - контейнер для списка редактирования трека (опциональный)
// type EditAtom struct {
// 	AtomHeader
// 	Children []Atom // Обычно содержит только elst
// }
//
// // EditListAtom - список редактирования (опциональный внутри edts)
// type EditListAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	Entries    []EditListEntry
// }
//
// type EditListEntry struct {
// 	SegmentDuration uint64 // Длительность сегмента в тиках временной шкалы фильма
// 	MediaTime       int64  // Время начала в медиа (в тиках временной шкалы медиа)
// 	MediaRate       int32  // Скорость воспроизведения (1.0 = 0x00010000)
// }
//
// // MediaAtom - контейнер для информации о медиа трека (обязательный внутри trak)
// type MediaAtom struct {
// 	AtomHeader
// 	Children []Atom // mdhd, hdlr, minf (все обязательные)
// }
//
// // MediaHeaderAtom - заголовок медиа (обязательный внутри mdia)
// type MediaHeaderAtom struct {
// 	FullAtom
// 	CreationTime    uint64
// 	ModificationTime uint64
// 	TimeScale       uint32 // Временная шкала медиа
// 	Duration        uint64 // Длительность медиа в тиках временной шкалы
// 	Language        uint16 // Язык (ISO-639-2/T в упакованном формате)
// 	Quality         uint16 // Качество (обычно 0)
// }
//
// // HandlerReferenceAtom - ссылка на обработчик (обязательный внутри mdia)
// type HandlerReferenceAtom struct {
// 	FullAtom
// 	Reserved1       [4]byte
// 	HandlerType     [4]byte // 'vide', 'soun', 'text', 'hint' и др.
// 	Reserved2       [12]byte
// 	Name            string  // Название обработчика (завершается нулём)
// }
//
// // MediaInformationAtom - контейнер для информации о медиа (обязательный внутри mdia)
// type MediaInformationAtom struct {
// 	AtomHeader
// 	Children []Atom // vmhd/smhd/hmhd/nmhd, dinf, stbl
// }
//
// // VideoMediaHeaderAtom - заголовок видео-медиа (для видео треков)
// type VideoMediaHeaderAtom struct {
// 	FullAtom
// 	GraphicsMode uint16 // Режим композитинга
// 	OpColor      [3]uint16 // Цвет операции (R, G, B)
// }
//
// // SoundMediaHeaderAtom - заголовок аудио-медиа (для аудио треков)
// type SoundMediaHeaderAtom struct {
// 	FullAtom
// 	Balance int16 // Баланс стерео (-1.0 левый, 0.0 центр, +1.0 правый)
// 	Reserved uint16
// }
//
// // HintMediaHeaderAtom - заголовок для треков подсказок
// type HintMediaHeaderAtom struct {
// 	FullAtom
// 	MaxPDUSize   uint16
// 	AvgPDUSize   uint16
// 	MaxBitrate   uint32
// 	AvgBitrate   uint32
// 	SlidingWindowSize uint32
// }
//
// // NullMediaHeaderAtom - заголовок для "пустых" треков
// type NullMediaHeaderAtom struct {
// 	FullAtom
// }
//
// // DataInformationAtom - контейнер для информации о расположении данных
// type DataInformationAtom struct {
// 	AtomHeader
// 	Children []Atom // Обычно содержит только dref
// }
//
// // DataReferenceAtom - таблица ссылок на данные
// type DataReferenceAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	// Entries []DataReferenceEntry (структура зависит от типа ссылки)
// }
//
// // SampleTableAtom - контейнер для таблиц семплов (обязательный внутри minf)
// type SampleTableAtom struct {
// 	AtomHeader
// 	Children []Atom // stsd, stts, stsc, stsz/stz2, stco/co64 и др.
// }
//
// // SampleDescriptionAtom - описание формата семплов
// type SampleDescriptionAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	// SampleDescriptions []SampleDescriptionEntry
// 	// Для видео: VisualSampleEntry содержит ширину, высоту, кодек и т.д.
// 	// Для аудио: AudioSampleEntry содержит частоту, количество каналов и т.д.
// }
//
// // TimeToSampleAtom - таблица временных меток семплов
// type TimeToSampleAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	Entries    []TimeToSampleEntry
// }
//
// type TimeToSampleEntry struct {
// 	SampleCount uint32 // Количество семплов с одинаковым дельта-временем
// 	SampleDelta uint32 // Дельта-время между семплами в тиках временной шкалы медиа
// }
//
// // SampleToChunkAtom - соответствие чанков и семплов
// type SampleToChunkAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	Entries    []SampleToChunkEntry
// }
//
// type SampleToChunkEntry struct {
// 	FirstChunk      uint32 // Номер первого чанка в группе
// 	SamplesPerChunk uint32 // Количество семплов в каждом чанке группы
// 	SampleDescriptionID uint32 // ID описания семпла из stsd
// }
//
// // SampleSizeAtom - размеры семплов
// type SampleSizeAtom struct {
// 	FullAtom
// 	SampleSize  uint32 // Если != 0, все семплы имеют одинаковый размер
// 	SampleCount uint32
// 	SampleSizes []uint32 // Если SampleSize == 0, содержит размер каждого семпла
// }
//
// // ChunkOffsetAtom - смещения чанков в файле (32-битная версия)
// type ChunkOffsetAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	ChunkOffsets []uint32 // Смещения чанков от начала файла
// }
//
// // ChunkLargeOffsetAtom - смещения чанков в файле (64-битная версия, 'co64')
// type ChunkLargeOffsetAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	ChunkOffsets []uint64 // Смещения чанков от начала файла
// }
//
// // SyncSampleAtom - таблица ключевых кадров (опциональный)
// type SyncSampleAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	SampleNumbers []uint32 // Номера семплов, являющихся ключевыми кадрами (I-frames)
// }
//
// // CompositionOffsetAtom - компенсация времени композиции (опциональный для B-frames)
// type CompositionOffsetAtom struct {
// 	FullAtom
// 	EntryCount uint32
// 	Entries    []CompositionOffsetEntry
// }
//
// type CompositionOffsetEntry struct {
// 	SampleCount uint32
// 	SampleOffset int32 // Смещение времени композиции в тиках
// }
//
// // ============================================================================
// // Дополнительные атомы верхнего уровня внутри moov
// // ============================================================================
//
// // UserDataAtom - контейнер для пользовательских метаданных
// type UserDataAtom struct {
// 	AtomHeader
// 	Children []Atom // meta, copyright, name и другие пользовательские атомы
// }
//
// // MetadataAtom - атом метаданных (современный формат)
// type MetadataAtom struct {
// 	FullAtom
// 	Children []Atom // hdlr, ilst/dinf и др.
// }
//
// // MovieExtendsAtom - расширения фильма для фрагментированных MP4 (fMP4)
// type MovieExtendsAtom struct {
// 	AtomHeader
// 	Children []Atom // trex*
// }
//
// // TrackExtendsAtom - значения по умолчанию для фрагментов трека
// type TrackExtendsAtom struct {
// 	FullAtom
// 	TrackID                     uint32
// 	DefaultSampleDescriptionIndex uint32
// 	DefaultSampleDuration       uint32
// 	DefaultSampleSize           uint32
// 	DefaultSampleFlags          uint32
// }
//
// // ============================================================================
// // Вспомогательные функции для работы с атомами
// // ============================================================================
//
// // Читает заголовок атома из бинарного потока
// func ReadAtomHeader(data []byte) (AtomHeader, error) {
// 	if len(data) < 8 {
// 		return AtomHeader{}, errors.New("ErrAtomTooSmall")
// 	}
// 	
// 	header := AtomHeader{
// 		Size: binary.BigEndian.Uint32(data[0:4]),
// 	}
// 	copy(header.Type[:], data[4:8])
// 	
// 	return header, nil
// }
//
// // Проверяет, является ли атом контейнером (может содержать дочерние атомы)
// func IsContainerAtom(atomType [4]byte) bool {
// 	containerTypes := [][4]byte{
// 		{'m', 'o', 'o', 'v'}, {'t', 'r', 'a', 'k'}, {'m', 'd', 'i', 'a'},
// 		{'m', 'i', 'n', 'f'}, {'s', 't', 'b', 'l'}, {'e', 'd', 't', 's'},
// 		{'u', 'd', 't', 'a'}, {'d', 'i', 'n', 'f'}, {'m', 'v', 'e', 'x'},
// 	}
// 	for _, ct := range containerTypes {
// 		if atomType == ct {
// 			return true
// 		}
// 	}
// 	return false
// }
