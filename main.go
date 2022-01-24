package main

import (
	"fmt"
	"log"
	"os"
)

const LONGWORD_SIZE = 4

const RESIDENT_LIBRARIES_START = 4

func longWordSlice(stream []byte, offset uint) []byte {
	return stream[offset : offset+LONGWORD_SIZE]
}

func longWordValue(stream []byte, offset uint) uint32 {
	slice := longWordSlice(stream, offset)
	return uint32(slice[3]) + uint32(slice[2])<<8 +
		uint32(slice[1])<<16 + uint32(slice[0])<<24
}

func main() {
	fmt.Println("Amiga Inspect")
	content, err := os.ReadFile("space")
	if err != nil {
		log.Fatal(err)
	}
	processFile(content)
}

func processFile(content []byte) {
	if !checkHunkHeader(content) {
		fmt.Println("Incorrect header for Amiga Executable")
		return
	}
	fmt.Println("* Header check OK")

	residentLibEnd := residentLibraries(content)
	hTableSize, nextOffset := hunkTableSize(content, residentLibEnd)
	fmt.Printf("* Hunk table size: %d\n", hTableSize)
	firstHunk, nextOffset := firstHunkNumber(content, nextOffset)
	fmt.Printf("* First hunk: %d\n", firstHunk)
	lastHunk, nextOffset := lastHunkNumber(content, nextOffset)
	fmt.Printf("* Last hunk: %d\n", lastHunk)
	totalHunks := lastHunk - firstHunk + 1
	fmt.Printf("* Total number of hunks in file: %d\n", totalHunks)
	hSizes, nextOffset := hunkSizes(content, nextOffset, totalHunks)
	for i, hSize := range hSizes {
		fmt.Printf("* Size for hunk %d: %d\n", i, hSize)
	}
}

// checkHunkHeader checks if the byte stream begins with the
// AmigaOS "magic cookie" for Hunk executable files (0x000003f3).
func checkHunkHeader(content []byte) bool {
	return (content[0] == 0x00) && (content[1] == 0x00) &&
		(content[2] == 0x03) && (content[3] == 0xf3)
}

// residentLibraries scans the list of names of resident libraries
// that should be loaded with the program, and returns the offset
// to the first byte after the table.
// TODO: right now it assumes there are no resident libraries in the list
func residentLibraries(content []byte) uint {
	if longWordValue(content, RESIDENT_LIBRARIES_START) != 0 {
		fmt.Println("Calls to resident libraries found")
		os.Exit(0)
	}
	fmt.Println("* No calls to resident libraries found")
	return RESIDENT_LIBRARIES_START + LONGWORD_SIZE
}

// hunkTableSize reads and returns the Hunk table size needed by a
// loader when loading the program. This includes the hunks included
// in the file but also hunks loaded from resident libraries.
// The second return value is the offset of the next field in the format.
func hunkTableSize(content []byte, offset uint) (uint32, uint) {
	return longWordValue(content, offset), offset + LONGWORD_SIZE
}

// firstHunkNumber retrieves the number of the first hunk in the hunk
// table that should be loaded. If no resident libraries are referenced,
// this should always be zero.
func firstHunkNumber(content []byte, offset uint) (uint32, uint) {
	return longWordValue(content, offset), offset + LONGWORD_SIZE
}

// lastHunkNumber retrieves the number of the last hunk in the hunk
// table that should be loaded.
func lastHunkNumber(content []byte, offset uint) (uint32, uint) {
	return longWordValue(content, offset), offset + LONGWORD_SIZE
}

// hunkSizes retrieves the sizes of hunks in the hunk table.
func hunkSizes(content []byte, offset uint, hunks uint32) ([]uint32, uint) {
	var result []uint32
	currentOffset := offset
	for i := 0; i < int(hunks); i++ {
		result = append(result, longWordValue(content, currentOffset))
		currentOffset += LONGWORD_SIZE
	}
	return result, currentOffset
}
