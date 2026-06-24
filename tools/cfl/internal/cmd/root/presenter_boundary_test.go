package root

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestCFLPresenterBoundaryEnforcement(t *testing.T) {
	t.Parallel()

	cmdRoot := filepath.Clean("..")
	fset := token.NewFileSet()
	var violations []string

	err := filepath.WalkDir(cmdRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel := filepath.ToSlash(path)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			if imp.Path.Value == `"github.com/open-cli-collective/atlassian-go/view"` && !allowedViewImport(rel) {
				violations = append(violations, rel+": direct shared/view import outside root/init exception")
			}
		}
		imports := boundaryImports(file.Imports)

		file, err = parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			pos := fset.Position(call.Pos())
			if violation := presenterBoundaryViolation(fset, rel, pos.Line, call, imports); violation != "" {
				violations = append(violations, violation)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("unexpected cfl presenter-boundary violations:\n%s", strings.Join(violations, "\n"))
	}
}

type importNames struct {
	fmt map[string]bool
	io  map[string]bool
	log map[string]bool
	os  map[string]bool
}

func presenterBoundaryViolation(fset *token.FileSet, rel string, line int, call *ast.CallExpr, imports importNames) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	receiver := exprString(fset, sel.X)
	methodName := sel.Sel.Name
	location := rel + ":" + itoa(line)

	if receiver == "v" && legacyViewHelper(methodName) && !allowedInitException(rel) {
		return boundaryMessage(location, "legacy shared/view helper v."+methodName+" outside init exception")
	}
	if receiver == "view" && methodName == "ValidateFormat" {
		return boundaryMessage(location, "view.ValidateFormat is not allowed in cfl command output paths")
	}
	if methodName == "View" && !allowedInitException(rel) {
		return boundaryMessage(location, "opts.View() is only allowed for root/init transitional exceptions")
	}
	if imports.fmt[receiver] && fmtOutputCall(methodName) && len(call.Args) > 0 {
		target := exprString(fset, call.Args[0])
		if outputWriteTarget(target, imports) && !allowedPromptWrite(fset, rel, call) && !allowedInitException(rel) {
			return boundaryMessage(location, "command-local "+methodName+" write to "+target+" is not presenter-owned")
		}
	}
	if imports.fmt[receiver] && fmtBareOutputCall(methodName) && !allowedInitException(rel) {
		return boundaryMessage(location, "command-local fmt."+methodName+" writes to process stdout/stderr outside presenter boundary")
	}
	if imports.io[receiver] && methodName == "WriteString" && len(call.Args) > 0 {
		target := exprString(fset, call.Args[0])
		if outputWriteTarget(target, imports) && !allowedInitException(rel) {
			return boundaryMessage(location, "command-local io.WriteString write to "+target+" is not presenter-owned")
		}
	}
	if outputWriteTarget(receiver, imports) && methodName == "Write" && !allowedInitException(rel) {
		return boundaryMessage(location, "command-local Write call on "+receiver+" is not presenter-owned")
	}
	if imports.log[receiver] && logOutputCall(methodName) && !allowedInitException(rel) {
		return boundaryMessage(location, "command-local log."+methodName+" output is not presenter-owned")
	}

	return ""
}

func boundaryMessage(location, problem string) string {
	return location + ": " + problem + "; use present.Emit or a tools/cfl/internal/present presenter"
}

func allowedViewImport(rel string) bool {
	return rel == "../root/root.go" || strings.HasPrefix(rel, "../init/")
}

func allowedInitException(rel string) bool {
	return strings.HasPrefix(rel, "../init/")
}

func allowedPromptWrite(fset *token.FileSet, rel string, call *ast.CallExpr) bool {
	if len(call.Args) == 0 || exprString(fset, call.Args[0]) != "opts.Stderr" {
		return false
	}
	if rel == "../configcmd/clear.go" {
		return len(call.Args) == 2 && exprString(fset, call.Args[1]) == `promptText + " [y/N]: "`
	}
	if isDeleteCommandFile(rel) {
		if len(call.Args) < 2 {
			return false
		}
		arg := exprString(fset, call.Args[1])
		return strings.Contains(arg, "About to delete") || strings.Contains(arg, "Are you sure? [y/N]:")
	}
	return false
}

func isDeleteCommandFile(rel string) bool {
	switch rel {
	case "../space/delete.go", "../page/delete.go", "../attachment/delete.go":
		return true
	}
	return false
}

func legacyViewHelper(name string) bool {
	switch name {
	case "Table", "Success", "RenderKeyValue", "RenderKeyValues", "Info", "Warning", "Error", "Println", "Render":
		return true
	}
	return false
}

func fmtOutputCall(name string) bool {
	switch name {
	case "Fprint", "Fprintf", "Fprintln":
		return true
	}
	return false
}

func outputWriteTarget(target string, imports importNames) bool {
	switch target {
	case "opts.Stdout", "opts.Stderr", "v.Out":
		return true
	}
	for osName := range imports.os {
		if target == osName+".Stdout" || target == osName+".Stderr" {
			return true
		}
	}
	return false
}

func fmtBareOutputCall(name string) bool {
	switch name {
	case "Print", "Printf", "Println":
		return true
	}
	return false
}

func logOutputCall(name string) bool {
	switch name {
	case "Print", "Printf", "Println", "Fatal", "Fatalf", "Fatalln", "Panic", "Panicf", "Panicln":
		return true
	}
	return false
}

func boundaryImports(imports []*ast.ImportSpec) importNames {
	names := importNames{
		fmt: map[string]bool{},
		io:  map[string]bool{},
		log: map[string]bool{},
		os:  map[string]bool{},
	}
	for _, imp := range imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := importName(imp, path)
		if name == "" {
			continue
		}
		switch path {
		case "fmt":
			names.fmt[name] = true
		case "io":
			names.io[name] = true
		case "log":
			names.log[name] = true
		case "os":
			names.os[name] = true
		}
	}
	return names
}

func importName(imp *ast.ImportSpec, path string) string {
	if imp.Name != nil {
		if imp.Name.Name == "_" || imp.Name.Name == "." {
			return ""
		}
		return imp.Name.Name
	}
	idx := strings.LastIndex(path, "/")
	if idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, fset, expr)
	return buf.String()
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = byte('0' + v%10)
		v /= 10
	}
	return string(digits[i:])
}
