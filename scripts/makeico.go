package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"os"
)

// ICO directory entry
type iconDirEntry struct {
	Width        byte
	Height       byte
	ColorCount   byte
	Reserved     byte
	Planes       uint16
	BitCount     uint16
	BytesInRes   uint32
	ImageOffset  uint32
}

func main() {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	baseDir := `C:\Users\Yousef-Laptop\Desktop\tidy-icon-assets\app-icon-final`
	outPath := `C:\Users\Yousef-Laptop\tidy\installer\tidy.ico`

	var entries []iconDirEntry
	var images [][]byte

	for _, size := range sizes {
		pngPath := fmt.Sprintf("%s\\tidy-app-icon-%d.png", baseDir, size)
		data, err := os.ReadFile(pngPath)
		if err != nil {
			fmt.Printf("warning: could not read %s: %v\n", pngPath, err)
			continue
		}

		// Validate it's a valid PNG
		_, err = png.Decode(bytes.NewReader(data))
		if err != nil {
			fmt.Printf("warning: invalid PNG %s: %v\n", pngPath, err)
			continue
		}

		w, h := byte(size), byte(size)
		if size == 256 {
			w, h = 0, 0 // 256 is encoded as 0 in ICO
		}

		entries = append(entries, iconDirEntry{
			Width:      w,
			Height:     h,
			ColorCount: 0,
			Reserved:   0,
			Planes:     1,
			BitCount:   32,
			BytesInRes: uint32(len(data)),
		})
		images = append(images, data)
	}

	if len(entries) == 0 {
		fmt.Println("no valid images found")
		os.Exit(1)
	}

	f, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("failed to create %s: %v\n", outPath, err)
		os.Exit(1)
	}
	defer f.Close()

	// ICO header
	binary.Write(f, binary.LittleEndian, uint16(0)) // Reserved
	binary.Write(f, binary.LittleEndian, uint16(1)) // Type: icon
	binary.Write(f, binary.LittleEndian, uint16(len(entries)))

	// Calculate offsets
	headerSize := 6 + len(entries)*16
	offset := headerSize
	for i := range entries {
		entries[i].ImageOffset = uint32(offset)
		offset += len(images[i])
	}

	// Write directory
	for _, e := range entries {
		binary.Write(f, binary.LittleEndian, e)
	}

	// Write image data
	for _, img := range images {
		f.Write(img)
	}

	fmt.Printf("Created %s with %d images\n", outPath, len(entries))
}
