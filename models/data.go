package models

type Config struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Fields []*Field `json:"fields"`
}

type Field struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Tag     string `json:"tag"`
	Comment string `json:"comment"`
	Value   string `json:"value"`
}
