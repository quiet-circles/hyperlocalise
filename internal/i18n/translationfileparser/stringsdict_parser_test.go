package translationfileparser

import (
	"strings"
	"testing"
)

func TestAppleStringsdictParserParsesPluralCategories(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>files_count</key>
  <dict>
    <key>NSStringLocalizedFormatKey</key>
    <string>%#@files@</string>
    <key>files</key>
    <dict>
      <key>NSStringFormatSpecTypeKey</key>
      <string>NSStringPluralRuleType</string>
      <key>NSStringFormatValueTypeKey</key>
      <string>d</string>
      <key>zero</key>
      <string>No files</string>
      <key>one</key>
      <string>%d file</string>
      <key>other</key>
      <string>%d files</string>
    </dict>
  </dict>
</dict>
</plist>`)

	got, err := (AppleStringsdictParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse stringsdict: %v", err)
	}

	if got["files_count.NSStringLocalizedFormatKey"] != "%#@files@" {
		t.Fatalf("unexpected format key: %q", got["files_count.NSStringLocalizedFormatKey"])
	}
	if got["files_count.files.one"] != "%d file" {
		t.Fatalf("unexpected one value: %q", got["files_count.files.one"])
	}
	if got["files_count.files.other"] != "%d files" {
		t.Fatalf("unexpected other value: %q", got["files_count.files.other"])
	}
}

func TestMarshalAppleStringsdictPreservesTemplateAndPlaceholders(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>files_count</key>
  <dict>
    <key>NSStringLocalizedFormatKey</key>
    <string>%#@files@</string>
    <key>files</key>
    <dict>
      <key>one</key>
      <string>%d file</string>
      <key>other</key>
      <string>%d files</string>
    </dict>
  </dict>
</dict>
</plist>`)

	out, err := MarshalAppleStringsdict(template, map[string]string{
		"files_count.NSStringLocalizedFormatKey": "%#@files@",
		"files_count.files.one":                  "%d fichier",
		"files_count.files.other":                "%d fichiers",
	})
	if err != nil {
		t.Fatalf("marshal stringsdict: %v", err)
	}

	rendered := string(out)
	if !strings.Contains(rendered, "<plist version=\"1.0\">") {
		t.Fatalf("expected plist structure preserved, got %q", rendered)
	}
	if !strings.Contains(rendered, "<string>%#@files@</string>") {
		t.Fatalf("expected format placeholder preserved, got %q", rendered)
	}
	if !strings.Contains(rendered, "<string>%d fichier</string>") {
		t.Fatalf("expected one category translation, got %q", rendered)
	}
	if !strings.Contains(rendered, "<string>%d fichiers</string>") {
		t.Fatalf("expected other category translation, got %q", rendered)
	}
}
