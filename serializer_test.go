package go_xml

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
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

	tests := []struct {
		name     string
		input    SimpleStruct
		opts     *MarshalOptions
		expected string
	}{
		{
			name:  "Basic case",
			input: SimpleStruct{ID: 1, Name: "TestName"},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
				RootTag:   "CustomRootTag",
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<CustomRootTag id="1">
  <name>TestName</name>
</CustomRootTag>`,
		},
		{
			name:  "Empty name",
			input: SimpleStruct{ID: 2, Name: ""},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
				RootTag:   "CustomRootTag",
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<CustomRootTag id="2">
  <name></name>
</CustomRootTag>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
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

	tests := []struct {
		name     string
		input    ParentStruct
		opts     *MarshalOptions
		expected string
	}{
		{
			name: "With child and children",
			input: ParentStruct{
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
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
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
</ParentStruct>`,
		},
		{
			name: "Without child",
			input: ParentStruct{
				ID:    3,
				Title: "Parent",
				Children: []ChildStruct{
					{ID: 5, Data: "Child1"},
					{ID: 6, Data: "Child2"},
				},
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<ParentStruct id="3">
  <title>Parent</title>
  <children>
    <child id="5">
      <data>Child1</data>
    </child>
    <child id="6">
      <data>Child2</data>
    </child>
  </children>
</ParentStruct>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
	}
}

func TestSelfClosingTagsSerialization(t *testing.T) {
	type ElementWithEmptyFields struct {
		ID          int    `xml:"id,attr"`
		Content     string `xml:"content"`
		Description string `xml:"description"`
		Note        string `xml:"note"`
	}

	tests := []struct {
		name     string
		input    ElementWithEmptyFields
		opts     *MarshalOptions
		expected string
	}{
		{
			name: "All fields empty",
			input: ElementWithEmptyFields{
				ID:          7,
				Content:     "",
				Description: "",
				Note:        "",
			},
			opts: &MarshalOptions{
				SelfClosingTags: []string{"content", "description", "note"},
				Indent:          "  ",
				XMLHeader:       true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<ElementWithEmptyFields id="7">
  <content/>
  <description/>
  <note/>
</ElementWithEmptyFields>`,
		},
		{
			name: "Some fields with content",
			input: ElementWithEmptyFields{
				ID:          8,
				Content:     "Has content",
				Description: "",
				Note:        "Also has content",
			},
			opts: &MarshalOptions{
				SelfClosingTags: []string{"content", "description", "note"},
				Indent:          "  ",
				XMLHeader:       true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<ElementWithEmptyFields id="8">
  <content>Has content</content>
  <description/>
  <note>Also has content</note>
</ElementWithEmptyFields>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
	}
}

func TestCompressionOptionWithLargerData(t *testing.T) {
	type Data struct {
		Text string `xml:"text"`
	}

	largeText := strings.Repeat("This is a test string for compression. ", 100)
	example := Data{Text: largeText}

	tests := []struct {
		name           string
		opts           *MarshalOptions
		verifyFunction func(t *testing.T, uncompressedData, compressedData []byte)
	}{
		{
			name: "Compression",
			opts: &MarshalOptions{
				Compress:  true,
				XMLHeader: true,
				Indent:    "  ",
			},
			verifyFunction: func(t *testing.T, uncompressedData, compressedData []byte) {
				t.Logf("Uncompressed data size: %d bytes", len(uncompressedData))
				t.Logf("Compressed data size: %d bytes", len(compressedData))

				if len(compressedData) >= len(uncompressedData) {
					t.Fatalf("Compressed data is not smaller than uncompressed data")
				}

				reader, err := gzip.NewReader(bytes.NewReader(compressedData))
				if err != nil {
					t.Fatalf("Gzip reader error: %v", err)
				}
				defer reader.Close()
				decompressedData, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("Decompression error: %v", err)
				}

				if !bytes.Equal(uncompressedData, decompressedData) {
					t.Fatalf("Decompressed data does not match uncompressed data")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uncompressedData, err := Marshal(example, &MarshalOptions{
				XMLHeader: true,
				Indent:    "  ",
			})
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}

			compressedData, err := Marshal(example, tt.opts)
			if err != nil {
				t.Fatalf("Compression error: %v", err)
			}

			tt.verifyFunction(t, uncompressedData, compressedData)
		})
	}
}

func TestSpecialCharacters(t *testing.T) {
	type SpecialCharStruct struct {
		Text string `xml:"text"`
	}

	tests := []struct {
		name     string
		input    SpecialCharStruct
		opts     *MarshalOptions
		expected string
	}{
		{
			name:  "Special characters",
			input: SpecialCharStruct{Text: "Special chars: & < > \" '"},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<SpecialCharStruct>
  <text>Special chars: &amp; &lt; &gt; &quot; &apos;</text>
</SpecialCharStruct>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
	}
}

func TestOmitEmptyFields(t *testing.T) {
	type OmitEmptyStruct struct {
		ID    int     `xml:"id,attr"`
		Name  string  `xml:"name,omitempty"`
		Value float64 `xml:"value,omitempty"`
		Note  *string `xml:"note,omitempty"`
	}

	tests := []struct {
		name     string
		input    OmitEmptyStruct
		opts     *MarshalOptions
		expected string
	}{
		{
			name:  "Omit empty fields",
			input: OmitEmptyStruct{ID: 8},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<OmitEmptyStruct id="8"></OmitEmptyStruct>`,
		},
		{
			name: "With non-empty fields",
			input: OmitEmptyStruct{
				ID:    9,
				Name:  "TestName",
				Value: 123.45,
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<OmitEmptyStruct id="9">
  <name>TestName</name>
  <value>123.45</value>
</OmitEmptyStruct>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
	}
}

func TestNamespaceSerialization(t *testing.T) {
	type NamespacedStruct struct {
		ID   int    `xml:"id,attr"`
		Name string `xml:"name"`
	}

	tests := []struct {
		name     string
		input    NamespacedStruct
		opts     *MarshalOptions
		expected string
	}{
		{
			name:  "With namespace",
			input: NamespacedStruct{ID: 9, Name: "Namespaced"},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
				Namespace: "http://example.com/schema",
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<NamespacedStruct xmlns="http://example.com/schema" id="9">
  <name>Namespaced</name>
</NamespacedStruct>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
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

	tests := []struct {
		name     string
		input    PointerStruct
		opts     *MarshalOptions
		expected string
	}{
		{
			name: "All fields set",
			input: PointerStruct{
				ID:     &id,
				Name:   &name,
				Active: &active,
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<PointerStruct id="10">
  <name>Pointer</name>
  <active>true</active>
</PointerStruct>`,
		},
		{
			name: "Some fields nil",
			input: PointerStruct{
				ID:   &id,
				Name: nil,
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<PointerStruct id="10"></PointerStruct>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
	}
}

func TestMixedContentSerialization(t *testing.T) {
	type MixedContent struct {
		ID     int      `xml:"id,attr"`
		Title  string   `xml:"title"`
		Values []string `xml:"values>value"`
		Note   string   `xml:"note,omitempty"`
	}

	tests := []struct {
		name     string
		input    MixedContent
		opts     *MarshalOptions
		expected string
	}{
		{
			name: "With note",
			input: MixedContent{
				ID:     11,
				Title:  "Mixed Content",
				Values: []string{"One", "Two", "Three"},
				Note:   "Note content",
			},
			opts: &MarshalOptions{
				Indent:    "    ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<MixedContent id="11">
    <title>Mixed Content</title>
    <values>
        <value>One</value>
        <value>Two</value>
        <value>Three</value>
    </values>
    <note>Note content</note>
</MixedContent>`,
		},
		{
			name: "Without note",
			input: MixedContent{
				ID:     11,
				Title:  "Mixed Content",
				Values: []string{"One", "Two", "Three"},
			},
			opts: &MarshalOptions{
				Indent:    "    ",
				XMLHeader: true,
			},
			expected: `<?xml version="1.0" encoding="UTF-8"?>
<MixedContent id="11">
    <title>Mixed Content</title>
    <values>
        <value>One</value>
        <value>Two</value>
        <value>Three</value>
    </values>
</MixedContent>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputBytes, err := Marshal(tt.input, tt.opts)
			if err != nil {
				t.Fatalf("Serialization error: %v", err)
			}
			if normalizeXML(string(outputBytes)) != normalizeXML(tt.expected) {
				t.Fatalf("Expected: %s, Got: %s", tt.expected, string(outputBytes))
			}
			if !isReadable(outputBytes) {
				t.Fatalf("Output is not readable")
			}
		})
	}
}

func BenchmarkPerformance(b *testing.B) {
	type SimpleStruct struct {
		ID   int    `xml:"id,attr"`
		Name string `xml:"name"`
	}

	type NestedStruct struct {
		ID       int      `xml:"id,attr"`
		Title    string   `xml:"title"`
		Children []string `xml:"children>child"`
	}

	tests := []struct {
		scenario   string
		data       interface{}
		opts       *MarshalOptions
		iterations int
	}{
		{
			scenario: "Simple struct - 1M serializations",
			data: SimpleStruct{
				ID:   1,
				Name: "TestName",
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			iterations: 1_000_000,
		},
		{
			scenario: "Nested struct - 1M serializations",
			data: NestedStruct{
				ID:    2,
				Title: "Parent",
				Children: []string{
					"Child1",
					"Child2",
					"Child3",
				},
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			iterations: 1_000_000,
		},
		{
			scenario: "Large data - 500k serializations",
			data: NestedStruct{
				ID:    3,
				Title: "Large Parent",
				Children: func() []string {
					largeList := make([]string, 1000)
					for i := range largeList {
						largeList[i] = fmt.Sprintf("Child%d", i+1)
					}
					return largeList
				}(),
			},
			opts: &MarshalOptions{
				Indent:    "  ",
				XMLHeader: true,
			},
			iterations: 500_000,
		},
	}

	fmt.Println("Scenario,Iterations,TotalTime(s),AvgTimePerIteration(ms)")

	for _, tt := range tests {
		b.Run(tt.scenario, func(b *testing.B) {
			start := time.Now()

			for i := 0; i < tt.iterations; i++ {
				_, err := Marshal(tt.data, tt.opts)
				if err != nil {
					b.Fatalf("Serialization error: %v", err)
				}
			}

			duration := time.Since(start).Seconds()
			avgTimePerIteration := (duration / float64(tt.iterations)) * 1000 // in milliseconds
			fmt.Printf("%s,%d,%.2f,%.4f\n", tt.scenario, tt.iterations, duration, avgTimePerIteration)
		})
	}
}
