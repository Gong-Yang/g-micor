// 此文件仅用于代码生成，不参与实际编译
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// 生成package_gen.go、contract_gen.go
const (
	service = "notify" //需要生成的service

	servicePath  = "./service/"
	contractPath = "./contract/"
)

// FuncInfo 存储函数信息
type FuncInfo struct {
	Name    string
	ReqType string
	ResType string
	Doc     string
}

func main() {
	slog.Info("Starting code generation", "service", service)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, servicePath+service, nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("Parsed packages", "count", len(pkgs))

	// 收集所有带有 // export 的函数
	var exportedFuncs []FuncInfo
	for _, pkg := range pkgs {
		slog.Info("Processing package", "name", pkg.Name)
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					if fn.Doc != nil {
						for _, comment := range fn.Doc.List {
							if strings.Contains(comment.Text, "// export") {
								slog.Info("Found exported function", "name", fn.Name.Name)
								funcInfo := extractFuncInfo(fn)
								if funcInfo != nil {
									exportedFuncs = append(exportedFuncs, *funcInfo)
								}
								break
							}
						}
					}
				}
			}
		}
	}

	if len(exportedFuncs) == 0 {
		slog.Info("No exported functions found")
		return
	}

	slog.Info("Found exported functions", "count", len(exportedFuncs))

	// 生成文件
	generatePackageFile(exportedFuncs)
	generateContractFile(exportedFuncs)

	slog.Info("Code generation completed", "service", service, "functions", len(exportedFuncs))
}

// extractFuncInfo 从AST函数声明中提取函数信息
func extractFuncInfo(fn *ast.FuncDecl) *FuncInfo {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		slog.Warn("Function must have exactly one parameter", "func", fn.Name.Name)
		return nil
	}

	if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 {
		slog.Warn("Function must return exactly two values (res, err)", "func", fn.Name.Name)
		return nil
	}

	// 提取请求类型
	reqType := extractTypeString(fn.Type.Params.List[0].Type)
	if reqType == "" {
		slog.Warn("Cannot extract request type", "func", fn.Name.Name)
		return nil
	}

	// 提取响应类型
	resType := extractTypeString(fn.Type.Results.List[0].Type)
	if resType == "" {
		slog.Warn("Cannot extract response type", "func", fn.Name.Name)
		return nil
	}

	// 提取文档注释
	doc := ""
	if fn.Doc != nil {
		for _, comment := range fn.Doc.List {
			if !strings.Contains(comment.Text, "// export") {
				doc += strings.TrimPrefix(comment.Text, "//") + "\n"
			}
		}
		doc = strings.TrimSpace(doc)
	}

	return &FuncInfo{
		Name:    fn.Name.Name,
		ReqType: reqType,
		ResType: resType,
		Doc:     doc,
	}
}

// extractTypeString 从AST类型表达式中提取类型字符串
func extractTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		// 指针类型
		return "*" + extractTypeString(t.X)
	case *ast.SelectorExpr:
		// 包选择器，如 pkg.Type
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
	case *ast.Ident:
		// 简单标识符
		return t.Name
	}
	return ""
}

// generatePackageFile 生成 package_gen.go 文件
func generatePackageFile(funcs []FuncInfo) {
	filePath := filepath.Join(servicePath, service, "package_gen.go")

	tmpl := `package {{.Service}}

import (
	"github.com/Gong-Yang/g-micor/contract/{{.Service}}_contract"
)

func init() {
{{- range .Funcs}}
	{{$.Service}}_contract.{{.Name}}I = {{.Name}}
{{- end}}
}

type Service struct {
}

{{range .Funcs}}
func (s Service) {{.Name}}(req {{.ReqType}}, res {{.ResType}}) (err error) {
	res_, err := {{.Name}}(req)
	if err != nil {
		return err
	}
	*res = *res_
	return
}

{{end}}`

	data := struct {
		Service string
		Funcs   []FuncInfo
	}{
		Service: service,
		Funcs:   funcs,
	}

	if err := executeTemplate(tmpl, data, filePath); err != nil {
		log.Fatalf("Failed to generate package file: %v", err)
	}

	slog.Info("Generated package file", "path", filePath)
}

// generateContractFile 生成 contract_gen.go 文件
func generateContractFile(funcs []FuncInfo) {
	filePath := filepath.Join(contractPath, service+"_contract", "contract_gen.go")

	tmpl := `package {{.Service}}_contract

import (
	"github.com/Gong-Yang/g-micor/core/discover"
	"log/slog"
)

{{range .Funcs}}
var {{.Name}}I func(req {{trimPkg .ReqType}}) (res {{trimPkg .ResType}}, err error)

{{end}}
{{range .Funcs}}
{{if .Doc}}// {{.Doc}}{{end}}
func {{.Name}}(req {{trimPkg .ReqType}}) (res {{trimPkg .ResType}}, err error) {
	if {{.Name}}I != nil {
		return {{.Name}}I(req)
	}
	client, err := discover.Discover("{{$.Service}}")
	if err != nil {
		slog.Info("{{$.Service}} discover error", "err", err)
		return
	}
	res = new({{sp .ResType}})
	err = client.Call("{{$.Service}}.{{.Name}}", req, res)
	if err != nil { 
		return
	}
	return
}

{{end}}`

	// 创建模板函数
	funcMap := template.FuncMap{
		"trimPkg": func(s string) string {
			return "*" + strings.Split(s, ".")[1]
		},
		"sp": func(s string) string {
			return strings.Split(s, ".")[1]
		},
	}

	data := struct {
		Service string
		Funcs   []FuncInfo
	}{
		Service: service,
		Funcs:   funcs,
	}

	if err := executeTemplateWithFuncs(tmpl, funcMap, data, filePath); err != nil {
		log.Fatalf("Failed to generate contract file: %v", err)
	}

	slog.Info("Generated contract file", "path", filePath)
}

// executeTemplate 执行模板并写入文件
func executeTemplate(tmplStr string, data interface{}, filePath string) error {
	tmpl, err := template.New("gen").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// executeTemplateWithFuncs 执行带函数的模板并写入文件
func executeTemplateWithFuncs(tmplStr string, funcMap template.FuncMap, data interface{}, filePath string) error {
	tmpl, err := template.New("gen").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
