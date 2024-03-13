package parse

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"godoc/models"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultModFile = "go.mod"
)

type GoParser struct {
	projPath   string
	moduleName string
}

func (g *GoParser) init() error {
	modPath := filepath.Join(g.projPath, DefaultModFile)
	content, err := os.ReadFile(modPath)
	if err != nil {
		return err
	}

	f, err := modfile.Parse(DefaultModFile, content, nil)
	if err != nil {
		return err
	}

	g.moduleName = f.Module.Mod.Path

	return nil
}

func (g *GoParser) ModuleName() string {
	return g.moduleName
}

func (g *GoParser) Parse(relPath, name string) ([]*models.Config, error) {
	tmpRes := make(map[string]*models.Config)
	_, err := g.parseFile("", relPath, name, tmpRes)
	if err != nil {
		return nil, err
	}

	var res []*models.Config

	for _, v := range tmpRes {
		res = append(res, v)
	}
	return res, nil
}

func (g *GoParser) parseFile(prefix, path, name string, tmpRes map[string]*models.Config) (bool, error) {
	if _, ok := tmpRes[filepath.Join(prefix, name)]; ok {
		return true, nil
	}

	codePath := filepath.Join(g.projPath, path)
	fs := token.NewFileSet()

	af, err := g.load(fs, codePath)
	if err != nil {
		return false, err
	}
	if af.Scope == nil {
		return false, errors.New("has no scope params")
	}

	// get target object
	obj := af.Scope.Lookup(name)
	if obj == nil {
		return false, nil
	}

	pkgs, err := g.loadPackage(fs, af, filepath.Dir(codePath))
	if err != nil {
		return true, err
	}
	info := pkgs[0].TypesInfo

	pkgMap, err := g.parseImport(af.Imports, pkgs[0].Imports)
	if err != nil {
		return true, err
	}

	confMap := make(map[string]*models.Config)
	c := g.parseObj(prefix, confMap, info, obj)
	if c == nil {
		return true, nil
	}

	if c != nil && c.Type == "" {
		tmpRes[c.Name] = c
	}

	for k, v := range confMap {
		if v == nil {
			if index := strings.Index(k, "."); index != -1 {
				// outer
				pkgName := k[:index]
				confName := k[index+1:]
				if importFiles, ok := pkgMap[pkgName]; !ok || importFiles == nil {
					continue
				} else {
					for _, importFile := range importFiles {
						matched, err := g.parseFile(filepath.Dir(importFile), importFile, confName, tmpRes)
						if err != nil || !matched {
							continue
						}
						break
					}
				}
			} else {
				// inner
				for _, singleFile := range pkgs[0].GoFiles {
					if filepath.Base(singleFile) == filepath.Base(path) {
						continue
					}
					relPath, err := filepath.Rel(g.projPath, singleFile)
					if err != nil {
						continue
					}
					matched, err := g.parseFile(prefix, relPath, k, tmpRes)
					if err != nil || !matched {
						continue
					}
					break
				}
			}
		} else if v.Type != "" {
			if _, ok := tmpRes[k]; !ok {
				if g.isBasic(v.Type) {
					constMap, err := g.getConst(info, af.Decls)
					if err != nil {
						return true, err
					}
					if constConf, ok := constMap[k]; ok {
						tmpRes[k] = constConf
					}
				} else {
					tmpRes[v.Name] = v
				}
			}
		} else {
			if _, ok := tmpRes[v.Name]; !ok {
				tmpRes[v.Name] = v
			}
		}
	}
	return true, nil
}

func (g *GoParser) parseImport(astImports []*ast.ImportSpec, pkgImports map[string]*packages.Package) (map[string][]string, error) {
	pkgMap := make(map[string][]string)
	for _, importSpec := range astImports {
		var (
			pkgPath = strings.Trim(importSpec.Path.Value, `"`)
			pkgName string
		)
		if importSpec.Name != nil {
			pkgName = importSpec.Name.Name
		}
		v := pkgImports[pkgPath]
		if pkgName == "" {
			pkgName = v.Name
		}
		if strings.HasPrefix(pkgPath, g.moduleName) {
			for _, singleFile := range v.GoFiles {
				relPath, err := filepath.Rel(g.projPath, singleFile)
				if err != nil {
					return nil, err
				}
				pkgMap[pkgName] = append(pkgMap[pkgName], relPath)
			}
		} else {
			pkgMap[pkgName] = nil
		}
	}
	return pkgMap, nil
}

func (g *GoParser) getConst(info *types.Info, decls []ast.Decl) (map[string]*models.Config, error) {
	res := make(map[string]*models.Config)
	for _, decl := range decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
			for _, spec := range genDecl.Specs {
				v := spec.(*ast.ValueSpec)
				var comment string
				if v.Doc != nil {
					for _, line := range v.Doc.List {
						comment += strings.TrimSpace(strings.TrimPrefix(line.Text, "//"))
					}
				}
				if v.Comment != nil {
					for _, line := range v.Comment.List {
						comment += strings.TrimSpace(strings.TrimPrefix(line.Text, "//"))
					}
				}
				for _, n := range v.Names {
					obj := info.ObjectOf(n)
					if obj == nil {
						continue
					}

					c := obj.(*types.Const)

					var curType string
					switch t := c.Type().(type) {
					case *types.Named:
						curType = t.Obj().Name()
					default:
						curType = c.Type().String()
					}
					if _, ok := res[curType]; !ok {
						res[curType] = &models.Config{
							Name:   curType,
							Type:   c.Type().Underlying().String(),
							Fields: make([]*models.Field, 0),
						}
					}
					res[curType].Fields = append(res[curType].Fields, &models.Field{
						Name:    n.Name,
						Type:    curType,
						Comment: comment,
						Value:   c.Val().ExactString(),
					})
				}
			}
		}
	}
	return res, nil
}

func (g *GoParser) load(fs *token.FileSet, path string) (*ast.File, error) {
	return parser.ParseFile(fs, path, nil, parser.ParseComments)
}

func (g *GoParser) loadPackage(fs *token.FileSet, af *ast.File, pkgPath string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports |
			packages.NeedDeps | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo,
		Fset: fs,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			return af, nil
		},
		Dir: g.projPath,
	}
	return packages.Load(cfg, pkgPath)
}

func (g *GoParser) parseObj(prefix string, confMap map[string]*models.Config, info *types.Info, obj *ast.Object) *models.Config {
	if obj == nil {
		return nil
	}

	var (
		res = &models.Config{Name: filepath.Join(prefix, obj.Name)}
	)

	switch obj.Kind {
	case ast.Typ:
		// type
		spec := obj.Decl.(*ast.TypeSpec)

		switch spec.Type.(type) {
		//case *ast.Ident:
		//	//res.Type = t.Name
		//	res.Type = g.parseType(prefix, confMap, info, spec.Type)
		//	if _, ok := confMap[obj.Name]; !ok {
		//		confMap[obj.Name] = res
		//	}
		case *ast.StructType:
			resFields := g.parseStructField(prefix, confMap, info, spec.Type, "")
			res.Fields = append(res.Fields, resFields...)
		default:
			res.Type = g.parseType(prefix, confMap, info, spec.Type)
			if _, ok := confMap[obj.Name]; !ok {
				confMap[obj.Name] = res
			}
		}
	default:
		return nil
	}
	return res
}

func (g *GoParser) parseStructField(prefix string, confMap map[string]*models.Config, info *types.Info, expr ast.Expr, name string) []*models.Field {
	var res []*models.Field
	switch t := expr.(type) {
	case *ast.StructType:
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				if field.Type == nil {
					continue
				}

				resField := &models.Field{}

				switch field.Type.(type) {
				case *ast.StructType:
					var childName string
					if name != "" {
						childName = fmt.Sprintf("%s.%s", name, field.Names[0].Name)
					} else {
						childName = field.Names[0].Name
					}
					childField := g.parseStructField(prefix, confMap, info, field.Type, childName)
					res = append(res, childField...)
					continue
				default:
					resField.Type = g.parseType(prefix, confMap, info, field.Type)
				}

				if field.Names != nil {
					if name != "" {
						resField.Name = fmt.Sprintf("%s.%s", name, field.Names[0].Name)
					} else {
						resField.Name = field.Names[0].Name
					}
				} else {
					continue
				}

				res = append(res, resField)
			}
		}
	default:
		return res
	}
	return res
}

func (g *GoParser) parseType(prefix string, confMap map[string]*models.Config, info *types.Info, expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		if confMap != nil && (t.Obj != nil || !g.isBasic(t.Name)) {
			if t.Obj != nil {
				if _, ok := confMap[t.Name]; !ok {
					c := g.parseObj(prefix, confMap, info, t.Obj)
					confMap[t.Name] = c
				}
			} else {
				if _, ok := confMap[t.Name]; !ok {
					confMap[t.Name] = nil
				}
			}
		}
		return t.Name
	case *ast.StarExpr:
		return "*" + g.parseType(prefix, confMap, info, t.X)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", g.parseType(prefix, confMap, info, t.Key), g.parseType(prefix, confMap, info, t.Value))
	case *ast.SelectorExpr:
		name := fmt.Sprintf("%s.%s", g.parseType(prefix, nil, info, t.X), g.parseType(prefix, nil, info, t.Sel))
		if confMap != nil {
			if _, ok := confMap[name]; !ok {
				confMap[name] = nil
			}
		}
		return name
	case *ast.ArrayType:
		return "[]" + g.parseType(prefix, confMap, info, t.Elt)
	case *ast.ChanType:
		var flag string
		switch t.Dir {
		case ast.SEND | ast.RECV:
			flag = "chan"
		case ast.SEND:
			flag = "chan<-"
		case ast.RECV:
			flag = "<-chan"
		}
		return flag + " " + g.parseType(prefix, confMap, info, t.Value)
	case *ast.FuncType:
		var params []string
		if t.Params != nil {
			for _, field := range t.Params.List {
				typ := g.parseType(prefix, confMap, info, field.Type)
				if len(field.Names) == 0 {
					params = append(params, typ)
				} else {
					var names []string
					for _, name := range field.Names {
						names = append(names, name.Name)
					}
					params = append(params, fmt.Sprintf("%s %s", strings.Join(names, ", "), typ))
				}
			}
		}
		var results []string
		if t.Results != nil {
			for _, field := range t.Results.List {
				typ := g.parseType(prefix, confMap, info, field.Type)
				if len(field.Names) == 0 {
					results = append(results, typ)
				} else {
					var names []string
					for _, name := range field.Names {
						names = append(names, name.Name)
					}
					results = append(results, fmt.Sprintf("%s %s", strings.Join(names, ", "), typ))
				}
			}
		}
		r := strings.Join(results, ", ")
		if len(results) > 1 {
			r = fmt.Sprintf("(%s)", r)
		}
		return fmt.Sprintf("func(%s) %s", strings.Join(params, ", "), r)
	case *ast.InterfaceType:
		return "interface{}"
	}
	return ""
}

func (g *GoParser) isBasic(name string) bool {
	res := true
	switch name {
	case "int", "int8", "int16", "int32", "int64":
	case "uint", "uint8", "uint16", "uint32", "uint64":
	case "float32", "float64":
	case "complex64", "complex128":
	case "uintptr":
	case "byte":
	case "rune":
	case "any":
	case "string":
	case "bool":
	default:
		res = false
	}
	return res
}

func NewGoParser(path string) (*GoParser, error) {
	g := &GoParser{
		projPath: path,
	}

	if err := g.init(); err != nil {
		return nil, err
	}

	return g, nil
}
