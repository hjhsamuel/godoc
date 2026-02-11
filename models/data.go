package models

type Config struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Fields []*Field `json:"fields"`
	Index  int      `json:"index"`
}

type Field struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Tag     string `json:"tag"`
	Comment string `json:"comment"`
	Value   string `json:"value"`
	Index   int    `json:"index"`
}
