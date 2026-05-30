package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
)

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
	baseDir := `C:\Users\Yousef-Laptop\Desktop\tidy-icon-assets\app-icon-final`
	outPath := `C:\Users\Yousef-Laptop\tidy\installer\tidy.ico`

	// Read the high-res source
	srcPath := baseDir + `\tidy-app-icon-256.png`
	f, err := os.Open(srcPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	src, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	sizes := []int{16, 24, 32, 48, 64, 128, 256}

	var entries []iconDirEntry
	var images [][]byte

	for _, size := range sizes {
	// Scale down using nearest-neighbor (standard library)
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sx := x * src.Bounds().Dx() / size
			sy := y * src.Bounds().Dy() / size
			dst.Set(x, y, src.At(src.Bounds().Min.X+sx, src.Bounds().Min.Y+sy))
		}
	}

		var buf bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		encoder.Encode(&buf, dst)

		w, h := byte(size), byte(size)
		if size == 256 {
			w, h = 0, 0
		}

		entries = append(entries, iconDirEntry{
			Width:      w,
			Height:     h,
			Planes:     1,
			BitCount:   32,
			BytesInRes: uint32(buf.Len()),
		})
		images = append(images, buf.Bytes())
	}

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()

	binary.Write(out, binary.LittleEndian, uint16(0))
	binary.Write(out, binary.LittleEndian, uint16(1))
	binary.Write(out, binary.LittleEndian, uint16(len(entries)))

	headerSize := 6 + len(entries)*16
	offset := headerSize
	for i := range entries {
		entries[i].ImageOffset = uint32(offset)
		offset += len(images[i])
	}

	for _, e := range entries {
		binary.Write(out, binary.LittleEndian, e)
	}
	for _, img := range images {
		out.Write(img)
	}

	fmt.Printf("Created %s with %d sizes (scaled from 256px source)\n", outPath, len(entries))
}
