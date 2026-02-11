package output

import (
	"github.com/hjhsamuel/godoc/models"
	"github.com/hjhsamuel/godoc/output/md_file"
	"github.com/pkg/errors"
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
