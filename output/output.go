package output

import (
	"github.com/pkg/errors"
	"godoc/models"
	"godoc/output/md_file"
)

type Output interface {
	Print(path, title string, val []*models.Config) error
}

func NewOutput(t models.OutputType) (Output, error) {
	var o Output
	switch t {
	case models.OutputType_Markdown:
		o = md_file.New()
	case models.OutputType_Json:
	default:
		return nil, errors.Errorf("type %s not supported", t)
	}
	return o, nil
}
