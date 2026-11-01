package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestEncodeOutputUTF8BOM(t *testing.T) {
	got, err := EncodeOutput("# 标题", EncUTF8BOM)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, utf8BOM) || !bytes.HasSuffix(got, []byte("# 标题")) {
		t.Errorf("unexpected output: %x", got)
	}
	// 未知取值同样回退 utf8_bom
	got2, err := EncodeOutput("x", "unknown-enc")
	if err != nil || !bytes.HasPrefix(got2, utf8BOM) {
		t.Errorf("unknown encoding should fall back to utf8_bom: %x err=%v", got2, err)
	}
}

func TestEncodeOutputUTF8NoBOM(t *testing.T) {
	got, err := EncodeOutput("# 标题", EncUTF8)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(got, utf8BOM) || string(got) != "# 标题" {
		t.Errorf("utf8 should be raw content: %x", got)
	}
}

func TestEncodeOutputGB18030(t *testing.T) {
	got, err := EncodeOutput("# 标题 for go", EncGB18030)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := simplifiedchinese.GB18030.NewDecoder().Bytes(got)
	if err != nil || string(dec) != "# 标题 for go" {
		t.Errorf("gb18030 round trip failed: %q err=%v", dec, err)
	}
}

func TestWriteOutputEncodingApplied(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "sub", "doc.md")
	path, err := WriteOutput(dir, "sub/doc.md", "# 树形 ├──", EncGB18030)
	if err != nil {
		t.Fatal(err)
	}
	if path != out {
		t.Errorf("path = %q, want %q", path, out)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(data, utf8BOM) {
		t.Error("gb18030 output must not carry a UTF-8 BOM")
	}
	dec, _ := simplifiedchinese.GB18030.NewDecoder().Bytes(data)
	if string(dec) != "# 树形 ├──" {
		t.Errorf("round trip mismatch: %q", dec)
	}
}
