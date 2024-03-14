package md_file

import (
	"bytes"
	"fmt"
	"godoc/models"
	"os"
	"strings"
)

type Markdown struct {
}

func (m *Markdown) Print(path, title string, val []*models.Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.WriteString(m.title(title, 1)); err != nil {
		return err
	}

	for _, v := range val {
		if _, err = f.WriteString(m.title(v.Name, 2)); err != nil {
			return err
		}
		if v.Type != "" {
			_, err = f.WriteString(fmt.Sprintf("Type: %s\n\n", v.Type))
			if err != nil {
				return err
			}
		}
		if _, err = f.WriteString(m.table(v.Fields)); err != nil {
			return err
		}
		if _, err = f.WriteString("\n\n\n"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Markdown) title(v string, level int) string {
	return fmt.Sprintf("%s %s\n", strings.Repeat("#", level), v)
}

func (m *Markdown) table(fields []*models.Field) string {
	if len(fields) == 0 {
		return ""
	}

	buf := bytes.NewBuffer(nil)
	if fields[0].Value != "" {
		buf.WriteString("| Name | Value | Comment |\n")
		buf.WriteString("| ----- | ----- | ----- |\n")
		for _, field := range fields {
			buf.WriteString(fmt.Sprintf("| %s | %s | %s |\n", field.Name, field.Value,
				m.replaceEOL(field.Comment)))
		}
	} else {
		buf.WriteString("| Name | Type | Tag | Comment |\n")
		buf.WriteString("| ----- | ----- | ----- | ----- |\n")
		for _, field := range fields {
			buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", field.Name, m.formatSpChar(field.Type),
				field.Tag, m.replaceEOL(field.Comment)))
		}
	}
	return buf.String()
}

func (m *Markdown) formatSpChar(v string) string {
	buf := bytes.NewBuffer(nil)
	for _, c := range v {
		switch c {
		case '[', ']', '(', ')', '*', '.':
			buf.WriteByte('\\')
		}
		buf.WriteRune(c)
	}
	return buf.String()
}

func (m *Markdown) replaceEOL(v string) string {
	return strings.Replace(v, "\n", " <br>", -1)
}

func New() *Markdown {
	return &Markdown{}
}
