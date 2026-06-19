package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: icon-generator <output.ico>\n")
		os.Exit(1)
	}

	icoPath := os.Args[1]

	// Icon sizes to generate
	sizes := []int{16, 32, 48, 64, 128, 256}

	// Generate images at each size (simple purple gradient logo)
	images := make([]image.Image, len(sizes))
	for i, size := range sizes {
		img := generateIcon(size)
		images[i] = img
	}

	// Create ICO file
	if err := encodeICO(icoPath, images); err != nil {
		log.Fatalf("Failed to create ICO: %v", err)
	}

	fmt.Printf("✓ Created %s with %d sizes\n", icoPath, len(sizes))
}

// generateIcon creates a simple purple gradient icon matching the brand.
func generateIcon(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Transparent background
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
		}
	}

	// Draw a simple arrow/bolt shape in purple (#863bff)
	// This matches the favicon design: a stylized lightning bolt
	midX := size / 2
	midY := size / 2
	scale := float64(size) / 48.0

	// Purple color from the favicon
	purple := color.RGBA{134, 59, 255, 255}

	// Draw simplified bolt shape (filled diamond/arrow)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x) - float64(midX)
			fy := float64(y) - float64(midY)

			// Create a simple arrow/bolt shape
			// Check if point is inside the shape
			if inBolt(fx, fy, 12*scale) {
				img.SetRGBA(x, y, purple)
			}
		}
	}

	return img
}

// inBolt checks if a point is within the bolt shape.
func inBolt(x, y, radius float64) bool {
	// Simplified: create an arrow pointing up-right
	// Use multiple triangles to form the bolt

	// Top triangle
	if y < 0 && x < radius && y > -radius && x-y < radius {
		return true
	}
	// Middle body
	if y >= -radius && y < radius && x > -radius*0.4 && x < radius*0.8 {
		return true
	}
	// Bottom barb
	if y >= radius && y < radius*1.5 && x > radius*0.3 && x < radius*1.2 {
		return true
	}

	return false
}

// encodeICO creates a Windows ICO file from the given images.
// Uses PNG encoding for better compression (modern Windows supports this).
func encodeICO(filename string, images []image.Image) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode all images as PNG
	imageData := make([][]byte, len(images))
	for i, img := range images {
		buf := new(bytes.Buffer)
		if err := png.Encode(buf, img); err != nil {
			return fmt.Errorf("failed to encode PNG for size %dx%d: %w", img.Bounds().Dx(), img.Bounds().Dy(), err)
		}
		imageData[i] = buf.Bytes()
	}

	// ICO header (6 bytes)
	// Reserved (2 bytes): must be 0
	// Type (2 bytes): 1 for ICO
	// ImageCount (2 bytes): number of images
	binary.Write(file, binary.LittleEndian, uint16(0)) // Reserved
	binary.Write(file, binary.LittleEndian, uint16(1)) // Type: ICO
	binary.Write(file, binary.LittleEndian, uint16(len(images))) // Image count

	// Calculate offsets for image data
	// Directory starts at offset 6
	// Directory is 16 bytes per image
	dataStartOffset := 6 + 16*int64(len(images))
	offsets := make([]int32, len(images))
	offset := dataStartOffset

	for i := range images {
		offsets[i] = int32(offset)
		offset += int64(len(imageData[i]))
	}

	// Write directory entries
	for i, img := range images {
		bounds := img.Bounds()
		width := uint8(bounds.Dx())
		height := uint8(bounds.Dy())
		if width == 0 {
			width = 0 // 0 means 256
		}
		if height == 0 {
			height = 0 // 0 means 256
		}

		binary.Write(file, binary.LittleEndian, width)     // Width
		binary.Write(file, binary.LittleEndian, height)    // Height
		binary.Write(file, binary.LittleEndian, uint8(0))  // Color count (0 = no palette)
		binary.Write(file, binary.LittleEndian, uint8(0))  // Reserved
		binary.Write(file, binary.LittleEndian, uint16(1)) // Color planes
		binary.Write(file, binary.LittleEndian, uint16(32)) // Bits per pixel
		binary.Write(file, binary.LittleEndian, uint32(len(imageData[i]))) // Image size
		binary.Write(file, binary.LittleEndian, offsets[i]) // Offset to image data
	}

	// Write image data
	for _, data := range imageData {
		if _, err := file.Write(data); err != nil {
			return err
		}
	}

	return nil
}
