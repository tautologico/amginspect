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

func checkHunkHeader(content []byte) bool {
	return (content[0] == 0x00) && (content[1] == 0x00) &&
		(content[2] == 0x03) && (content[3] == 0xf3)
}

func processFile(content []byte) {
	if !checkHunkHeader(content) {
		fmt.Println("Incorrect header for Amiga Executable")
		return
	}
	fmt.Println("* Header check OK")

	residentLibEnd := residentLibraries(content)
	fmt.Printf("* Hunk table size: %d\n", hunkTableSize(content, residentLibEnd))
}

func residentLibraries(content []byte) uint {
	if longWordValue(content, RESIDENT_LIBRARIES_START) != 0 {
		fmt.Println("Calls to resident libraries found")
		os.Exit(0)
	}
	fmt.Println("* No calls to resident libraries found")
	return RESIDENT_LIBRARIES_START + LONGWORD_SIZE
}

func hunkTableSize(content []byte, end uint) uint32 {
	return longWordValue(content, end)
}
