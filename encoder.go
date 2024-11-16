package go_xml

import (
	"io"
	"strings"
)

type Encoder struct {
	w           io.Writer
	selfClosing map[string]bool
	indent      string
	depth       int
}

func NewEncoder(w io.Writer, selfClosingTags []string, indent string) *Encoder {
	selfClosing := make(map[string]bool)
	for _, tag := range selfClosingTags {
		selfClosing[tag] = true
	}
	return &Encoder{
		w:           w,
		selfClosing: selfClosing,
		indent:      indent,
		depth:       0,
	}
}
func (e *Encoder) writeIndent() error {
	if e.indent != "" {
		if e.depth > 0 {
			_, err := e.w.Write([]byte("\n" + strings.Repeat(e.indent, e.depth)))
			return err
		}

		_, err := e.w.Write([]byte(strings.Repeat(e.indent, e.depth)))
		return err
	}
	return nil
}

func (e *Encoder) VisitElement(node *ElementNode) error {
	if err := e.writeIndent(); err != nil {
		return err
	}

	_, err := e.w.Write([]byte("<" + node.Name))
	if err != nil {
		return err
	}

	for _, attr := range node.Attributes {
		if attr.Name == "xmlns" {
			_, err := e.w.Write([]byte(` xmlns="` + escapeString(attr.Value) + `"`))
			if err != nil {
				return err
			}
			break
		}
	}

	for _, attr := range node.Attributes {
		if attr.Name != "xmlns" {
			_, err := e.w.Write([]byte(` ` + attr.Name + `="` + escapeString(attr.Value) + `"`))
			if err != nil {
				return err
			}
		}
	}

	isSelfClosing := node.SelfClose || (e.selfClosing[node.Name] && len(node.Children) == 0)
	if isSelfClosing {
		_, err := e.w.Write([]byte("/>"))
		if err != nil {
			return err
		}
		releaseElementNode(node)
		return nil
	}

	_, err = e.w.Write([]byte(">"))
	if err != nil {
		return err
	}

	e.depth++
	for _, child := range node.Children {
		err := child.Accept(e)
		if err != nil {
			return err
		}
	}
	e.depth--

	if e.indent != "" && len(node.Children) > 0 {
		if err := e.writeIndent(); err != nil {
			return err
		}
	}

	_, err = e.w.Write([]byte("</" + node.Name + ">"))
	if err != nil {
		return err
	}

	releaseElementNode(node)
	return nil
}

func (e *Encoder) VisitText(node *TextNode) error {
	_, err := e.w.Write([]byte(escapeString(node.Text)))
	if err != nil {
		return err
	}

	releaseTextNode(node)
	return nil
}
func escapeString(s string) string {
	var buf strings.Builder
	for _, c := range s {
		switch c {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(c)
		}
	}
	return buf.String()
}
