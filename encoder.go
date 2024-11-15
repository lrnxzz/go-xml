package go_xml

import (
	"io"
	"strings"
)

type Encoder struct {
	w           io.Writer
	selfClosing map[string]bool
}

func NewEncoder(w io.Writer, selfClosingTags []string) *Encoder {
	selfClosing := make(map[string]bool)
	for _, tag := range selfClosingTags {
		selfClosing[tag] = true
	}
	return &Encoder{
		w:           w,
		selfClosing: selfClosing,
	}
}

func (e *Encoder) VisitElement(node *ElementNode) error {
	_, err := e.w.Write([]byte("<" + node.Name))
	if err != nil {
		return err
	}

	for _, attr := range node.Attributes {
		_, err := e.w.Write([]byte(` ` + attr.Name + `="` + escapeString(attr.Value) + `"`))
		if err != nil {
			return err
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

	for _, child := range node.Children {
		err := child.Accept(e)
		if err != nil {
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
