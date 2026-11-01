package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"os"
)

// EncodePNG 把图像写为 PNG 文件。
func EncodePNG(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

// EncodeICO 把文档渲染成多尺寸 ICO 文件。
// 每个目录项直接内嵌 PNG 位图（Windows Vista+ 原生支持，XP SP2+ 亦可显示）。
func EncodeICO(path string, doc *Document, sizes []int) error {
	type entry struct {
		size int
		data []byte
	}
	var entries []entry
	for _, s := range sizes {
		img := Render(doc, s, s)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		entries = append(entries, entry{size: s, data: buf.Bytes()})
	}

	dirSize := 6 + 16*len(entries)
	out := make([]byte, dirSize)
	binary.LittleEndian.PutUint16(out[0:2], 0) // reserved
	binary.LittleEndian.PutUint16(out[2:4], 1) // type: 1 = icon
	binary.LittleEndian.PutUint16(out[4:6], uint16(len(entries)))

	offset := dirSize
	for i, e := range entries {
		p := 6 + 16*i
		b := byte(e.size)
		if e.size >= 256 {
			b = 0 // 256 在单字节字段中以 0 表示
		}
		out[p] = b                                      // width
		out[p+1] = b                                    // height
		out[p+2] = 0                                    // color count
		out[p+3] = 0                                    // reserved
		binary.LittleEndian.PutUint16(out[p+4:p+6], 1)  // planes
		binary.LittleEndian.PutUint16(out[p+6:p+8], 32) // bit count
		binary.LittleEndian.PutUint32(out[p+8:p+12], uint32(len(e.data)))
		binary.LittleEndian.PutUint32(out[p+12:p+16], uint32(offset))
		offset += len(e.data)
	}

	var buf bytes.Buffer
	buf.Write(out)
	for _, e := range entries {
		buf.Write(e.data)
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
