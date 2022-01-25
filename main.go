package main

//
// A tool to get information from AmigaOS Hunk files, especially executables.
// The format is described in "The AmigaDOS Manual", Chapter 10
//

import (
	"fmt"
	"log"
	"os"
)

// constants

const LONGWORD_SIZE = 4

const (
	HunkUnit    = 0x000003E7
	HunkName    = 0x000003E8
	HunkCode    = 0x000003E9
	HunkData    = 0x000003EA
	HunkBSS     = 0x000003EB
	HunkReloc32 = 0x000003EC
	HunkReloc16 = 0x000003ED
	HunkEnd     = 0x000003F2
)

var hunkTypeMap = map[uint32]string{
	HunkUnit:    "Start of program unit",
	HunkName:    "Name block",
	HunkCode:    "Code block",
	HunkData:    "Initialized data block",
	HunkBSS:     "Uninitialized data block",
	HunkReloc32: "32-bit relocation information",
	HunkReloc16: "16-bit relocation information",
	HunkEnd:     "End block of a hunk",
}

// longWordsPerLine determine the number of long words to show in each line of
// a raw code or data dump
var longWordsPerLine = 4

// longWordSlice returns a slice containing the next long word
// in the stream at offset.
func longWordSlice(stream []byte, offset uint) []byte {
	return stream[offset : offset+LONGWORD_SIZE]
}

// longWordValue returns the next long word in the stream
// at offset as an unsigned 32-bit value, assuming Big Endian
// byte ordering.
func longWordValue(stream []byte, offset uint) uint32 {
	slice := longWordSlice(stream, offset)
	return uint32(slice[3]) + uint32(slice[2])<<8 +
		uint32(slice[1])<<16 + uint32(slice[0])<<24
}

// Buffer keeps a byte stream and current position in the stream.
type Buffer struct {
	stream []byte
	offset uint
}

func createBuffer(stream []byte) Buffer {
	var b Buffer
	b.stream = stream
	b.offset = 0
	return b
}

func (b *Buffer) nextLongWord() uint32 {
	b.offset += LONGWORD_SIZE
	return longWordValue(b.stream, b.offset-LONGWORD_SIZE)
}

func (b *Buffer) nextLongWordAsSlice() []byte {
	b.offset += LONGWORD_SIZE
	return longWordSlice(b.stream, b.offset-LONGWORD_SIZE)
}

func (b *Buffer) advancePointer(offset uint) {
	b.offset += offset
}

func printLongWordSlice(longWord []byte) string {
	return fmt.Sprintf("%02x %02x %02x %02x", longWord[0], longWord[1],
		longWord[2], longWord[3])
}

func main() {
	fmt.Println("Amiga Inspect")

	if len(os.Args) == 1 {
		fmt.Printf("usage: amginspect <file>")
		os.Exit(0)
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Opening file %s ...\n", os.Args[1])

	buffer := createBuffer(content)
	processFile(&buffer)
}

func processFile(buffer *Buffer) {
	if !checkHunkHeader(buffer) {
		fmt.Println("Incorrect header for Amiga Executable")
		return
	}
	fmt.Println("* Header check OK")
	residentLibraries(buffer)
	hTableSize := hunkTableSize(buffer)
	fmt.Printf("* Hunk table size: %d\n", hTableSize)
	firstHunk := firstHunkNumber(buffer)
	fmt.Printf("* First hunk: %d\n", firstHunk)
	lastHunk := lastHunkNumber(buffer)
	fmt.Printf("* Last hunk: %d\n", lastHunk)
	totalHunks := lastHunk - firstHunk + 1
	fmt.Printf("* Total number of hunks in file: %d\n", totalHunks)
	hSizes := hunkSizes(buffer, totalHunks)
	for i, hSize := range hSizes {
		fmt.Printf("* Memory size for hunk %d: %d\n", i, hSize)
	}

	fmt.Printf("========================================\n")
	for i := 0; i < int(totalHunks); i++ {
		fmt.Printf("* Dumping Hunk #%d\n", i)
		fmt.Printf("----------\n")
		for !dumpHunkBlock(buffer) {
			fmt.Printf("----------\n")
		}
		fmt.Printf("========================================\n")
	}
}

// checkHunkHeader checks if the byte stream begins with the
// AmigaOS "magic cookie" for Hunk executable files (0x000003f3).
func checkHunkHeader(b *Buffer) bool {
	header := b.nextLongWordAsSlice()
	return (header[0] == 0x00) && (header[1] == 0x00) &&
		(header[2] == 0x03) && (header[3] == 0xf3)
}

// residentLibraries scans the list of names of resident libraries
// that should be loaded with the program, and returns the offset
// to the first byte after the table.
// TODO: right now it assumes there are no resident libraries in the list
func residentLibraries(buffer *Buffer) {
	if buffer.nextLongWord() != 0 {
		fmt.Println("Calls to resident libraries found")
		os.Exit(0)
	}
	fmt.Println("* No calls to resident libraries found")
}

// hunkTableSize reads and returns the Hunk table size needed by a
// loader when loading the program. This includes the hunks included
// in the file but also hunks loaded from resident libraries.
// The second return value is the offset of the next field in the format.
func hunkTableSize(buffer *Buffer) uint32 {
	return buffer.nextLongWord()
}

// firstHunkNumber retrieves the number of the first hunk in the hunk
// table that should be loaded. If no resident libraries are referenced,
// this should always be zero.
func firstHunkNumber(buffer *Buffer) uint32 {
	return buffer.nextLongWord()
}

// lastHunkNumber retrieves the number of the last hunk in the hunk
// table that should be loaded.
func lastHunkNumber(buffer *Buffer) uint32 {
	return buffer.nextLongWord()
}

// hunkSizes retrieves the sizes of hunks in the hunk table.
func hunkSizes(buffer *Buffer, hunks uint32) []uint32 {
	var result []uint32

	for i := 0; i < int(hunks); i++ {
		result = append(result, buffer.nextLongWord())
	}
	return result
}

// dumpHunkBlock displays information about the hunk block
// starting at the current buffer pointer (and advances
// the pointer to the next block). Returns true if
// this is the last block in the hunk.
func dumpHunkBlock(buffer *Buffer) bool {
	hunkType := buffer.nextLongWord()
	fmt.Printf("* Hunk block type: %s\n", showHunkType(hunkType))

	switch hunkType {
	case HunkCode:
		dumpCodeBlock(buffer)
	case HunkReloc32:
		dumpReloc32Block(buffer)
	case HunkData:
		dumpDataBlock(buffer)
	case HunkBSS:
		dumpBSSBlock(buffer)
	}

	return hunkType == HunkEnd
}

func showHunkType(hunkType uint32) string {
	typeStr, ok := hunkTypeMap[hunkType]
	if !ok {
		return "Unknown hunk block type"
	}
	return typeStr
}

func dumpCodeBlock(buffer *Buffer) {
	hunkSize := buffer.nextLongWord()
	fmt.Printf("* Code block size: %d long words = %d bytes\n", hunkSize,
		hunkSize*LONGWORD_SIZE)
	fmt.Printf("** Code: \n")
	lwNumber := 0
	for i := 0; i < int(hunkSize); i++ {
		nextLW := buffer.nextLongWordAsSlice()
		fmt.Printf("%s ", printLongWordSlice(nextLW))
		lwNumber += 1
		if lwNumber == longWordsPerLine {
			fmt.Printf("\n")
			lwNumber = 0
		}
	}
	fmt.Printf("\n")
}

func dumpReloc32Block(buffer *Buffer) {
	i := 0
	n := buffer.nextLongWord()
	for n != 0 {
		fmt.Printf("** N%d: %d\n", i+1, n)
		fmt.Printf("** Hunk number %d: %d\n", i+1, buffer.nextLongWord())
		for offs := 0; offs < int(n); offs++ {
			fmt.Printf("** Offset %d: %s\n", offs, printLongWordSlice(buffer.nextLongWordAsSlice()))
		}
		i++
		n = buffer.nextLongWord()
	}
}

func dumpDataBlock(buffer *Buffer) {
	blockSize := buffer.nextLongWord()
	fmt.Printf("** Data block size: %d long words = %d bytes\n", blockSize,
		blockSize*LONGWORD_SIZE)
	buffer.advancePointer(uint(blockSize * LONGWORD_SIZE))
}

func dumpBSSBlock(buffer *Buffer) {
	blockSize := buffer.nextLongWord()
	fmt.Printf("** BSS block size: %d long words = %d bytes\n", blockSize,
		blockSize*LONGWORD_SIZE)
}
