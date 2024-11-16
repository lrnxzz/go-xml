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

type Example struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name,attr"`
	Note string `xml:"note"`
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

func TestSimpleSerialization(t *testing.T) {
	ex := Example{ID: 1, Name: "Test", Note: ""}
	opts := &MarshalOptions{
		SelfClosingTags: []string{"note"},
		Indent:          "  ",      // Adding indentation
		XMLHeader:       true,      // Include XML header
		RootTag:         "Example", // Setting RootTag (optional, defaults to struct name)
	}
	outputBytes, err := Marshal(ex, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	output := string(outputBytes)

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<Example id="1" name="Test">
  <note/>
</Example>`
	if normalizeXML(output) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, output)
	}
}

func TestNestedSerialization(t *testing.T) {
	type NestedExample struct {
		ID              int       `xml:"id,attr"`
		ParentID        int       `xml:"parent_id,attr"`
		QtdDevolucoes   int       `xml:"QtdDevolucoes,attr"`
		ValorDevolucoes float64   `xml:"ValorDevolucoes,attr"`
		Children        []Example `xml:"children>child"`
	}

	ex := NestedExample{
		ID:              1,
		ParentID:        0,
		QtdDevolucoes:   17,
		ValorDevolucoes: 454.47,
		Children: []Example{
			{ID: 2, Name: "Child 1", Note: ""},
			{ID: 3, Name: "Child 2", Note: ""},
		},
	}
	opts := &MarshalOptions{
		SelfClosingTags: []string{"note"},
		Indent:          "  ",
		XMLHeader:       true,
		Namespace:       "http://example.com/schema",
		RootTag:         "NestedExample",
	}
	outputBytes, err := Marshal(ex, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}
	output := string(outputBytes)

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<NestedExample xmlns="http://example.com/schema" id="1" parent_id="0" QtdDevolucoes="17" ValorDevolucoes="454.47">
  <children>
    <child id="2" name="Child 1">
      <note/>
    </child>
    <child id="3" name="Child 2">
      <note/>
    </child>
  </children>
</NestedExample>`
	if normalizeXML(output) != normalizeXML(expected) {
		t.Fatalf("Expected: %s, Got: %s", expected, output)
	}
}

func TestSimplifiedSerialization(t *testing.T) {
	type Transacao struct {
		QtdTransacoes          int     `xml:"QtdTransacoes"`
		ValorTransacoes        float64 `xml:"ValorTransacoes"`
		DetalhamentoTransacoes int     `xml:"DetalhamentoTransacoes"`
	}

	type Devolucoes struct {
		QtdDevolucoes   int     `xml:"QtdDevolucoes,attr"`
		ValorDevolucoes float64 `xml:"ValorDevolucoes,attr"`
	}

	type APIX001 struct {
		DtArquivo  string      `xml:"DtArquivo,attr"`
		Ano        string      `xml:"Ano,attr"`
		ISPB       string      `xml:"ISPB,attr"`
		Transacoes []Transacao `xml:"Transacoes>Transacao"`
		Devolucoes Devolucoes  `xml:"Devolucoes"`
	}

	apix := APIX001{
		DtArquivo: "2021-11-30",
		Ano:       "2021",
		ISPB:      "12345678",
		Transacoes: []Transacao{
			{
				QtdTransacoes:          100,
				ValorTransacoes:        5000,
				DetalhamentoTransacoes: 1,
			},
			{
				QtdTransacoes:          50,
				ValorTransacoes:        1234.56,
				DetalhamentoTransacoes: 2,
			},
		},
		Devolucoes: Devolucoes{
			QtdDevolucoes:   5,
			ValorDevolucoes: 250,
		},
	}

	opts := &MarshalOptions{
		SelfClosingTags: []string{"Devolucoes"},
		Indent:          "    ", // Four spaces for indentation
		XMLHeader:       true,
		RootTag:         "APIX001",
	}

	outputBytes, err := Marshal(apix, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}

	expectedXML := `<?xml version="1.0" encoding="UTF-8"?>
<APIX001 DtArquivo="2021-11-30" Ano="2021" ISPB="12345678">
    <Transacoes>
        <Transacao>
            <QtdTransacoes>100</QtdTransacoes>
            <ValorTransacoes>5000.00</ValorTransacoes>
            <DetalhamentoTransacoes>1</DetalhamentoTransacoes>
        </Transacao>
        <Transacao>
            <QtdTransacoes>50</QtdTransacoes>
            <ValorTransacoes>1234.56</ValorTransacoes>
            <DetalhamentoTransacoes>2</DetalhamentoTransacoes>
        </Transacao>
    </Transacoes>
    <Devolucoes QtdDevolucoes="5" ValorDevolucoes="250.00"/>
</APIX001>`

	if normalizeXML(string(outputBytes)) != normalizeXML(expectedXML) {
		t.Fatalf("Expected: %s, Got: %s", expectedXML, string(outputBytes))
	}
}

func TestCompressionSerialization(t *testing.T) {
	type Message struct {
		Header string `xml:"Header"`
		Body   string `xml:"Body"`
		Footer string `xml:"Footer"`
	}

	msg := Message{
		Header: "Test Header",
		Body:   "This is a test message body.",
		Footer: "Test Footer",
	}

	opts := &MarshalOptions{
		Compress:  true,
		XMLHeader: true,
		Indent:    "  ",
	}

	compressedData, err := Marshal(msg, opts)
	if err != nil {
		t.Fatalf("Compression serialization error: %v", err)
	}

	if isReadable(compressedData) {
		t.Fatalf("Compressed data should not be readable as plain text")
	}

	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		t.Fatalf("Error creating gzip reader: %v", err)
	}
	defer reader.Close()

	decompressedData, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Error reading decompressed data: %v", err)
	}

	expectedXML := `<?xml version="1.0" encoding="UTF-8"?>
<Message>
  <Header>Test Header</Header>
  <Body>This is a test message body.</Body>
  <Footer>Test Footer</Footer>
</Message>`

	if normalizeXML(string(decompressedData)) != normalizeXML(expectedXML) {
		t.Fatalf("Expected: %s, Got: %s", expectedXML, string(decompressedData))
	}

	originalData, err := Marshal(msg, &MarshalOptions{
		XMLHeader: true,
		Indent:    "  ",
	})
	if err != nil {
		t.Fatalf("Error serializing without compression: %v", err)
	}

	if len(compressedData) >= len(originalData) {
		t.Logf("Compressed data is not smaller than original data. Compressed size: %d bytes, Original size: %d bytes", len(compressedData), len(originalData))
	} else {
		t.Logf("Compressed data size: %d bytes, Original data size: %d bytes", len(compressedData), len(originalData))
	}
}

func TestAdditionalScenarios(t *testing.T) {
	type Product struct {
		ID       int     `xml:"ID"`
		Name     string  `xml:"Name"`
		Price    float64 `xml:"Price"`
		Quantity int     `xml:"Quantity"`
		Note     string  `xml:"Note"`
	}

	product := Product{
		ID:       0,
		Name:     "",
		Price:    0.0,
		Quantity: 0,
		Note:     "",
	}

	opts := &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
		RootTag:   "Product",
	}

	outputBytes, err := Marshal(product, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}

	expectedXML := `<?xml version="1.0" encoding="UTF-8"?>
<Product>
  <ID>0</ID>
  <Name></Name>
  <Price>0.00</Price>
  <Quantity>0</Quantity>
  <Note></Note>
</Product>`

	if normalizeXML(string(outputBytes)) != normalizeXML(expectedXML) {
		t.Fatalf("Expected: %s, Got: %s", expectedXML, string(outputBytes))
	}

	type Data struct {
		Text string `xml:"Text"`
	}

	data := Data{
		Text: "Special characters: & < > \" '",
	}

	opts = &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
	}

	outputBytes, err = Marshal(data, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}

	expectedXML = `<?xml version="1.0" encoding="UTF-8"?>
<Data>
  <Text>Special characters: &amp; &lt; &gt; &quot; &apos;</Text>
</Data>`

	if normalizeXML(string(outputBytes)) != normalizeXML(expectedXML) {
		t.Fatalf("Expected: %s, Got: %s", expectedXML, string(outputBytes))
	}

	type Config struct {
		Enabled *bool  `xml:"Enabled"`
		Name    string `xml:"Name"`
	}

	enabled := true
	config := Config{
		Enabled: &enabled,
		Name:    "MyConfig",
	}

	opts = &MarshalOptions{
		Indent:    "  ",
		XMLHeader: true,
	}

	outputBytes, err = Marshal(config, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}

	expectedXML = `<?xml version="1.0" encoding="UTF-8"?>
<Config>
  <Enabled>true</Enabled>
  <Name>MyConfig</Name>
</Config>`

	if normalizeXML(string(outputBytes)) != normalizeXML(expectedXML) {
		t.Fatalf("Expected: %s, Got: %s", expectedXML, string(outputBytes))
	}

	type Tag struct {
		ID   int    `xml:"id,attr"`
		Name string `xml:"name,attr"`
	}

	type Item struct {
		ID   int    `xml:"id,attr"`
		Name string `xml:"name,attr"`
		Tags []Tag  `xml:"Tags>Tag"`
	}

	item := Item{
		ID:   1,
		Name: "Item1",
		Tags: []Tag{
			{ID: 101, Name: "Tag1"},
			{ID: 102, Name: "Tag2"},
		},
	}

	opts = &MarshalOptions{
		SelfClosingTags: []string{"Tag"},
		Indent:          "  ",
		XMLHeader:       true,
		RootTag:         "Item",
	}

	outputBytes, err = Marshal(item, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}

	expectedXML = `<?xml version="1.0" encoding="UTF-8"?>
<Item id="1" name="Item1">
  <Tags>
    <Tag id="101" name="Tag1"/>
    <Tag id="102" name="Tag2"/>
  </Tags>
</Item>`

	if normalizeXML(string(outputBytes)) != normalizeXML(expectedXML) {
		t.Fatalf("Expected: %s, Got: %s", expectedXML, string(outputBytes))
	}
}

func TestFullSelfClosingSerialization(t *testing.T) {
	type Details struct {
		Description string `xml:"description"`
		Remarks     string `xml:"remarks"`
	}

	type FullSelfClosingExample struct {
		ID      int      `xml:"id,attr"`
		Name    string   `xml:"name,attr"`
		Details *Details `xml:"details"`
	}

	example := FullSelfClosingExample{
		ID:   123,
		Name: "FullSelfClosing",
		Details: &Details{
			Description: "",
			Remarks:     "",
		},
	}

	opts := &MarshalOptions{
		SelfClosingTags: []string{"details", "description", "remarks"},
		Indent:          "  ",
		XMLHeader:       true,
		RootTag:         "FullSelfClosingExample",
	}

	outputBytes, err := Marshal(example, opts)
	if err != nil {
		t.Fatalf("Serialization error: %v", err)
	}

	expectedXML := `<?xml version="1.0" encoding="UTF-8"?>
<FullSelfClosingExample id="123" name="FullSelfClosing">
  <details>
    <description/>
    <remarks/>
  </details>
</FullSelfClosingExample>`

	if normalizeXML(string(outputBytes)) != normalizeXML(expectedXML) {
		t.Fatalf("Expected: %s, Got: %s", expectedXML, string(outputBytes))
	}
}
