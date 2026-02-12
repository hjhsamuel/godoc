package gogen

// LIMITED IMPLEMENT
// doc generation only used package, import, type, const and var

const (
	PackageDecl  = "package_clause"
	ImportDecl   = "import_declaration"
	FunctionDecl = "function_declaration"
	TypeDecl     = "type_declaration"
	ConstDecl    = "const_declaration"
	VarDecl      = "var_declaration"
	MethodDecl   = "method_declaration"
)

type Token struct {
	Start uint
	End   uint
}

type PackageInfo struct {
	Content *Token // complete code
	Name    *Token // nickname of the package
	Package *Token // real name of the package, COULD BE NIL
}

type ImportInfo struct {
	Content  *Token         // complete code
	Packages []*PackageInfo // imported packages
}
