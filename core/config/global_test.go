package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGlobalSettingsDefaults(t *testing.T) {
	dir := t.TempDir()
	g, err := LoadGlobalSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if g.OutputEncoding != EncodingDefault {
		t.Errorf("OutputEncoding = %q, want %q", g.OutputEncoding, EncodingDefault)
	}
	if g.InterfaceLanguage != LangAuto {
		t.Errorf("InterfaceLanguage = %q, want %q", g.InterfaceLanguage, LangAuto)
	}
}

func TestSaveLoadGlobalSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := &GlobalSettings{OutputEncoding: EncodingGB18030, InterfaceLanguage: "zh_cn"}
	if err := SaveGlobalSettings(dir, in); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(GlobalSettingsPath(dir)); err != nil {
		t.Fatalf("settings file not written: %v", err)
	}
	out, err := LoadGlobalSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Errorf("round trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestSaveGlobalSettingsRejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := SaveGlobalSettings(dir, &GlobalSettings{OutputEncoding: "utf-16"}); err == nil {
		t.Error("expected error for invalid encoding")
	}
	if err := SaveGlobalSettings(dir, &GlobalSettings{OutputEncoding: EncodingUTF8, InterfaceLanguage: "fr"}); err == nil {
		t.Error("expected error for invalid language")
	}
}

func TestLoadGlobalSettingsMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "global_settings.yml")
	if err := os.WriteFile(path, []byte("output_encoding: [broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGlobalSettings(dir); err == nil {
		t.Error("expected error for malformed yaml")
	}
}

func TestSerializeGlobalSettingsHasComments(t *testing.T) {
	text, err := SerializeGlobalSettings(DefaultGlobalSettings())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "output_encoding") || !strings.Contains(text, "utf8_bom | utf8 | gb18030") {
		t.Errorf("comment missing in output:\n%s", text)
	}
}
