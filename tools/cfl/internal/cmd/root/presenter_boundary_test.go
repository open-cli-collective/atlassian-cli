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
			if violation := presenterBoundaryViolation(fset, rel, pos.Line, call); violation != "" {
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

func presenterBoundaryViolation(fset *token.FileSet, rel string, line int, call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	receiver := exprString(fset, sel.X)
	name := sel.Sel.Name
	location := rel + ":" + itoa(line)

	if receiver == "v" && legacyViewHelper(name) && !allowedInitException(rel) {
		return location + ": legacy shared/view helper v." + name + " outside init exception"
	}
	if receiver == "view" && name == "ValidateFormat" {
		return location + ": view.ValidateFormat is not allowed in cfl command output paths"
	}
	if name == "View" && !allowedInitException(rel) {
		return location + ": opts.View() is only allowed for root/init transitional exceptions"
	}
	if receiver == "fmt" && fmtOutputCall(name) && len(call.Args) > 0 {
		target := exprString(fset, call.Args[0])
		if outputWriteTarget(target) && !allowedPromptWrite(fset, rel, call) && !allowedInitException(rel) {
			return location + ": command-local " + name + " write to " + target + " is not presenter-owned"
		}
	}
	if receiver == "fmt" && fmtBareOutputCall(name) && !allowedInitException(rel) {
		return location + ": command-local fmt." + name + " writes to process stdout/stderr outside presenter boundary"
	}

	return ""
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
	if rel == "../space/delete.go" || rel == "../page/delete.go" || rel == "../attachment/delete.go" {
		if len(call.Args) < 2 {
			return false
		}
		arg := exprString(fset, call.Args[1])
		return strings.Contains(arg, "About to delete") || strings.Contains(arg, "Are you sure? [y/N]:")
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

func outputWriteTarget(target string) bool {
	switch target {
	case "opts.Stdout", "opts.Stderr", "v.Out", "os.Stdout", "os.Stderr":
		return true
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
