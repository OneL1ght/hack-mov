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
