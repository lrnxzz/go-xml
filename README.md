# go-xml

The `go-xml`  was created to be an XML encoder with features like self-closing tags, customizable indentation, and built-in gzip compression. It simplifies serialization by generating XML text directly, avoiding the complexity of tokenizers, and ensures efficiency through the visitator pattern for structured processing. Inspired by FastHttp, it uses object pooling to minimizes allocations, making it fast and scalable, even for large datasets or high-throughput workloads.

# Installation

To use go-xml, simply add it to your Go project:
```
go get github.com/lrnxzz/go-xml/v2
```

# Complete Usage Example
```golang
package main

import (
	"fmt"
	"github.com/lrnxzz/go-xml/v2"
)

type Address struct {
	Street  string `xml:"street"`
	City    string `xml:"city"`
	ZipCode string `xml:"zipcode"`
}

type Employee struct {
	ID      int      `xml:"id,attr"`
	Name    string   `xml:"name"`
	Address *Address `xml:"address,omitempty"`
	Tags    []string `xml:"tags>tag,omitempty"`
	Notes   string   `xml:"notes,omitempty"`
}

type Department struct {
	Name       string     `xml:"name"`
	Employees  []Employee `xml:"employees>employee"`
	Department string     `xml:"department,attr"`
}

func main() {
	department := Department{
		Name:       "Engineering",
		Department: "Software",
		Employees: []Employee{
			{
				ID:   1,
				Name: "Alice",
				Address: &Address{
					Street:  "123 Elm St",
					City:    "Techville",
					ZipCode: "54321",
				},
				Tags:  []string{"team-lead", "mentor"},
				Notes: "Works remotely.",
			},
			{
				ID:   2,
				Name: "Bob",
				Tags: []string{"backend"},
			},
		},
	}

	options := &go_xml.MarshalOptions{
		Indent:          "  ",
		XMLHeader:       true,
		Namespace:       "http://example.com/departments",
		SelfClosingTags: []string{"notes"},
	}

	output, err := go_xml.Marshal(department, options)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(string(output))
}
```

For more complex examples and compression usage you can see here: serializer_test.go

## Ouput
```xml
<?xml version="1.0" encoding="UTF-8"?>
<CustomDepartment xmlns="http://example.com/departments" department="Software">
  <name>Engineering</name>
  <employees>
    <employee id="1">
      <name>Alice</name>
      <address>
        <street>123 Elm St</street>
        <city>Techville</city>
        <zipcode>54321</zipcode>
      </address>
      <tags>
        <tag>team-lead</tag>
        <tag>mentor</tag>
      </tags>
      <notes>Works remotely.</notes>
    </employee>
    <employee id="2">
      <name>Bob</name>
      <tags>
        <tag>backend</tag>
      </tags>
      <notes/>
    </employee>
  </employees>
</CustomDepartment>
```
# Benchmark

| **Scenario**                                        | **Iterations** | **Total Time (s)** | **Average Time (ms)** | **Time per Iteration (ns/op)**   |
|-----------------------------------------------------|----------------|---------------------|-----------------------|-----------------------------------|
| Simple Serialization *(basic flat structures with a small number of fields)* - **1M Iterations**          | 1,000,000      | **1.49**           | **0.0015**           | **1,491,303,889**                |
| Nested Serialization *(deeply nested structures with multiple child nodes)* - **1M Iterations**          | 1,000,000      | **3.19**           | **0.0032**           | **3,189,267,685**                |
| Large Data Serialization *(handling high volumes of data in complex structures)* - **1M Iterations**             | 1,000,000      | **213.43**         | **0.2134**           | **213,437,842,312**              |