package go_xml

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
	"unicode"
)

func normalizeXML(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "  ", "")
	return s
}

func isReadable(data []byte) bool {
	threshold := 0.8
	printableCount := 0
	totalCount := len(data)
	for _, b := range data {
		if unicode.IsPrint(rune(b)) || unicode.IsSpace(rune(b)) {
			printableCount++
		}
	}
	return float64(printableCount)/float64(totalCount) >= threshold
}

func TestBasicSerialization(t *testing.T) {
	type SimpleStruct struct {
		ID   int    `xml:"id,attr"`
		Name string `xml:"name"`
	}
	example := SimpleStruct{ID: 1, Name: "TestName"}
	opts := &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
		RootTag:   "CustomRootTag",
	}
	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<CustomRootTag id="1">
  <name>TestName</name>
</CustomRootTag>`
	if normalizeXML(string(outputBytes)) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, string(outputBytes))
	}
}

func TestNestedStructSerialization(t *testing.T) {
	type ChildStruct struct {
		ID   int    `xml:"id,attr"`
		Data string `xml:"data"`
	}
	type ParentStruct struct {
		ID       int           `xml:"id,attr"`
		Title    string        `xml:"title"`
		Child    *ChildStruct  `xml:"child,omitempty"`
		Children []ChildStruct `xml:"children>child"`
	}
	example := ParentStruct{
		ID:    3,
		Title: "Parent",
		Child: &ChildStruct{
			ID:   4,
			Data: "ChildData",
		},
		Children: []ChildStruct{
			{ID: 5, Data: "Child1"},
			{ID: 6, Data: "Child2"},
		},
	}
	opts := &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
	}
	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<ParentStruct id="3">
  <title>Parent</title>
  <child id="4">
    <data>ChildData</data>
  </child>
  <children>
    <child id="5">
      <data>Child1</data>
    </child>
    <child id="6">
      <data>Child2</data>
    </child>
  </children>
</ParentStruct>`
	if normalizeXML(string(outputBytes)) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, string(outputBytes))
	}
}

func TestSelfClosingTagsSerialization(t *testing.T) {
	type ElementWithEmptyFields struct {
		ID          int    `xml:"id,attr"`
		Content     string `xml:"content"`
		Description string `xml:"description"`
		Note        string `xml:"note"`
	}

	// Case with empty fields that should be self-closed
	example1 := ElementWithEmptyFields{
		ID:          7,
		Content:     "",
		Description: "",
		Note:        "",
	}
	opts1 := &MarshalOptions{
		SelfClosingTags: []string{"content", "description", "note"},
		Indent:          "  ",
		XMLHeader:       true,
	}
	outputBytes1, err := Marshal(example1, opts1)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected1 := `<?xml version="1.0" encoding="UTF-8"?>
<ElementWithEmptyFields id="7">
  <content/>
  <description/>
  <note/>
</ElementWithEmptyFields>`
	if normalizeXML(string(outputBytes1)) != normalizeXML(expected1) {
		t.Fatalf("Expected: %s, Got: %s", expected1, string(outputBytes1))
	}

	// Case where some fields have content
	example2 := ElementWithEmptyFields{
		ID:          8,
		Content:     "Has content",
		Description: "",
		Note:        "Also has content",
	}
	opts2 := opts1
	outputBytes2, err := Marshal(example2, opts2)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected2 := `<?xml version="1.0" encoding="UTF-8"?>
<ElementWithEmptyFields id="8">
  <content>Has content</content>
  <description/>
  <note>Also has content</note>
</ElementWithEmptyFields>`
	if normalizeXML(string(outputBytes2)) != normalizeXML(expected2) {
		t.Fatalf("Expected: %s, Got: %s", expected2, string(outputBytes2))
	}
}

func TestCompressionOptionWithLargerData(t *testing.T) {
	type Data struct {
		Text string `xml:"text"`
	}
	// Increase data size by repeating the text multiple times
	largeText := strings.Repeat("This is a test string for compression. ", 100)
	example := Data{Text: largeText}

	// Serialize without compression
	optsUncompressed := &MarshalOptions{
		XMLHeader: true,
		Indent:    "  ",
	}
	uncompressedData, err := Marshal(example, optsUncompressed)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}

	// Serialize with compression
	optsCompressed := &MarshalOptions{
		Compress:  true,
		XMLHeader: true,
		Indent:    "  ",
	}
	compressedData, err := Marshal(example, optsCompressed)
	if err != nil {
		t.Fatalf("Compression error: %v", err)
	}

	// Print sizes before and after compression
	t.Logf("Uncompressed data size: %d bytes", len(uncompressedData))
	t.Logf("Compressed data size: %d bytes", len(compressedData))

	// Verify that compressed data is smaller
	if len(compressedData) >= len(uncompressedData) {
		t.Fatalf("Compressed data is not smaller than uncompressed data")
	}

	// Decompress and verify the data
	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		t.Fatalf("Gzip reader error: %v", err)
	}
	defer reader.Close()
	decompressedData, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Decompression error: %v", err)
	}

	// Verify that the decompressed data matches the uncompressed data
	if !bytes.Equal(uncompressedData, decompressedData) {
		t.Fatalf("Decompressed data does not match uncompressed data")
	}
}

func TestSpecialCharacters(t *testing.T) {
	type SpecialCharStruct struct {
		Text string `xml:"text"`
	}
	example := SpecialCharStruct{Text: "Special chars: & < > \" '"}
	opts := &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
	}
	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<SpecialCharStruct>
  <text>Special chars: &amp; &lt; &gt; &quot; &apos;</text>
</SpecialCharStruct>`
	if normalizeXML(string(outputBytes)) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, string(outputBytes))
	}
}

func TestOmitEmptyFields(t *testing.T) {
	type OmitEmptyStruct struct {
		ID    int     `xml:"id,attr"`
		Name  string  `xml:"name,omitempty"`
		Value float64 `xml:"value,omitempty"`
		Note  *string `xml:"note,omitempty"`
	}
	example := OmitEmptyStruct{ID: 8}
	opts := &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
	}
	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<OmitEmptyStruct id="8"></OmitEmptyStruct>`
	if normalizeXML(string(outputBytes)) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, string(outputBytes))
	}
}

func TestNamespaceSerialization(t *testing.T) {
	type NamespacedStruct struct {
		ID   int    `xml:"id,attr"`
		Name string `xml:"name"`
	}
	example := NamespacedStruct{ID: 9, Name: "Namespaced"}
	opts := &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
		Namespace: "http://example.com/schema",
	}
	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<NamespacedStruct xmlns="http://example.com/schema" id="9">
  <name>Namespaced</name>
</NamespacedStruct>`
	if normalizeXML(string(outputBytes)) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, string(outputBytes))
	}
}

func TestPointerFieldsSerialization(t *testing.T) {
	type PointerStruct struct {
		ID     *int    `xml:"id,attr,omitempty"`
		Name   *string `xml:"name,omitempty"`
		Active *bool   `xml:"active,omitempty"`
	}
	id := 10
	name := "Pointer"
	active := true
	example := PointerStruct{ID: &id, Name: &name, Active: &active}
	opts := &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
	}
	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<PointerStruct id="10">
  <name>Pointer</name>
  <active>true</active>
</PointerStruct>`
	if normalizeXML(string(outputBytes)) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, string(outputBytes))
	}
}

func TestMixedContentSerialization(t *testing.T) {
	type MixedContent struct {
		ID     int      `xml:"id,attr"`
		Title  string   `xml:"title"`
		Values []string `xml:"values>value"`
		Note   string   `xml:"note,omitempty"`
	}
	example := MixedContent{
		ID:     11,
		Title:  "Mixed Content",
		Values: []string{"One", "Two", "Three"},
		Note:   "",
	}
	opts := &MarshalOptions{
		Indent:    "    ",
		XMLHeader: true,
	}
	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	expected := `<?xml version="1.0" encoding="UTF-8"?>
<MixedContent id="11">
    <title>Mixed Content</title>
    <values>
        <value>One</value>
        <value>Two</value>
        <value>Three</value>
    </values>
</MixedContent>`
	if normalizeXML(string(outputBytes)) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, string(outputBytes))
	}
}
