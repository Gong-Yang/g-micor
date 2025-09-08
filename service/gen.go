// 此文件仅用于代码生成，不参与实际编译
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
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

// 方法信息
type MethodInfo struct {
	Name       string
	ParamType  string
	ReturnType string
}

func main() {
	slog.Info("Starting code generation", "service", service)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, servicePath+service+"Service", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	// 扫描并收集方法信息
	methods, err := scanServiceMethods(pkgs, fset)
	if err != nil {
		log.Fatal(err)
	}

	if len(methods) == 0 {
		slog.Info("No service methods found")
		return
	}

	slog.Info("Found service methods", "count", len(methods))

	// 生成 contract_gen.go
	err = generateContractFile(service, methods)
	if err != nil {
		log.Fatal("Failed to generate contract file:", err)
	}

	// 生成 package_gen.go
	err = generatePackageFile(service, methods)
	if err != nil {
		log.Fatal("Failed to generate package file:", err)
	}

	slog.Info("Code generation completed successfully", "service", service)
}

// 扫描 Service 结构体的方法
func scanServiceMethods(pkgs map[string]*ast.Package, fset *token.FileSet) ([]MethodInfo, error) {
	var methods []MethodInfo

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					// 检查是否是方法
					if x.Recv != nil && len(x.Recv.List) > 0 {
						// 获取接收者类型
						recvType := getReceiverType(x.Recv.List[0])

						// 检查是否是 *Service 类型的方法
						if recvType == "*Service" {
							methodName := x.Name.Name

							// 跳过 Init 方法
							if methodName == "Init" {
								return true
							}

							// 验证方法签名
							paramType, returnType, err := validateMethodSignature(x, fset)
							if err != nil {
								slog.Warn("Method signature validation failed", "method", methodName, "error", err)
								return true
							}

							methods = append(methods, MethodInfo{
								Name:       methodName,
								ParamType:  paramType,
								ReturnType: returnType,
							})

							slog.Info("Found service method", "method", methodName, "param", paramType, "return", returnType)
						}
					}
				}
				return true
			})
		}
	}

	return methods, nil
}

// 获取接收者类型
func getReceiverType(recv *ast.Field) string {
	switch t := recv.Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// 验证方法签名
func validateMethodSignature(fn *ast.FuncDecl, fset *token.FileSet) (string, string, error) {
	// 检查参数：应该有 context.Context 和一个请求参数
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 2 {
		return "", "", fmt.Errorf("method must have exactly 2 parameters (context.Context and request)")
	}

	// 第一个参数应该是 context.Context
	firstParam := fn.Type.Params.List[0]
	firstParamType := getTypeString(firstParam.Type)
	if firstParamType != "context.Context" {
		return "", "", fmt.Errorf("first parameter must be context.Context, got %s", firstParamType)
	}

	// 第二个参数应该是指针类型
	secondParam := fn.Type.Params.List[1]
	paramType := getTypeString(secondParam.Type)
	if !strings.HasPrefix(paramType, "*") {
		return "", "", fmt.Errorf("second parameter must be a pointer type, got %s", paramType)
	}

	// 检查返回值：应该有两个返回值
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 {
		return "", "", fmt.Errorf("method must have exactly 2 return values")
	}

	// 第一个返回值应该是指针类型
	firstReturn := fn.Type.Results.List[0]
	returnType := getTypeString(firstReturn.Type)
	if !strings.HasPrefix(returnType, "*") {
		return "", "", fmt.Errorf("first return value must be a pointer type, got %s", returnType)
	}

	// 第二个返回值应该是 error
	secondReturn := fn.Type.Results.List[1]
	errorType := getTypeString(secondReturn.Type)
	if errorType != "error" {
		return "", "", fmt.Errorf("second return value must be error, got %s", errorType)
	}

	return paramType, returnType, nil
}

// 获取类型字符串
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}

// 去掉类型名的包名前缀
func removePackagePrefix(typeName string) string {
	// 如果包含包名前缀（如 *notify.SendEmailRequest），只保留类型名部分
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) == 2 {
			// 处理指针类型
			if strings.HasPrefix(parts[0], "*") {
				return "*" + parts[1]
			} else {
				return parts[1]
			}
		}
	}
	return typeName
}

// 生成 contract_gen.go 文件
func generateContractFile(serviceName string, methods []MethodInfo) error {
	const contractTemplate = `package {{.ServiceName}}

import (
	"context"
	"github.com/Gong-Yang/g-micor/core/discover"
	"google.golang.org/grpc"
	"log/slog"
)

var Client {{.ServiceName | title}}Client = &{{.ServiceName}}RemoteClient{}

type {{.ServiceName}}RemoteClient struct {
	client {{.ServiceName | title}}Client
}

func (n *{{.ServiceName}}RemoteClient) init() error {
	c, err := discover.Grpc("{{.ServiceName}}")
	if err != nil {
		return err
	}
	client := New{{.ServiceName | title}}Client(c)
	n.client = client
	slog.Info("{{.ServiceName}} remote client init")
	return nil
}
{{range .Methods}}
func (n *{{$.ServiceName}}RemoteClient) {{.Name}}(ctx context.Context, in {{.ParamType | removePackagePrefix}}, opts ...grpc.CallOption) ({{.ReturnType | removePackagePrefix}}, error) {
	if n.client == nil {
		err := n.init()
		if err != nil {
			return nil, err
		}
	}
	return n.client.{{.Name}}(ctx, in, opts...)
}
{{end}}
`

	funcMap := template.FuncMap{
		"title":               strings.Title,
		"removePackagePrefix": removePackagePrefix,
	}

	tmpl, err := template.New("contract").Funcs(funcMap).Parse(contractTemplate)
	if err != nil {
		return err
	}

	data := struct {
		ServiceName string
		Methods     []MethodInfo
	}{
		ServiceName: serviceName,
		Methods:     methods,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return err
	}

	// 格式化代码
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	// 确保目录存在
	contractDir := filepath.Join(contractPath, serviceName)
	err = os.MkdirAll(contractDir, 0755)
	if err != nil {
		return err
	}

	// 写入文件
	filePath := filepath.Join(contractDir, "contract_gen.go")
	err = os.WriteFile(filePath, formatted, 0644)
	if err != nil {
		return err
	}

	slog.Info("Generated contract file", "path", filePath)
	return nil
}

// 生成 package_gen.go 文件
func generatePackageFile(serviceName string, methods []MethodInfo) error {
	const packageTemplate = `package {{.ServiceName}}Service

import (
	"context"
	"github.com/Gong-Yang/g-micor/contract/{{.ServiceName}}"
	"google.golang.org/grpc"
)

type {{.ServiceName}}LocalClient struct {
	server *Service
}
{{range .Methods}}
func (n *{{$.ServiceName}}LocalClient) {{.Name}}(ctx context.Context, in {{.ParamType}}, opts ...grpc.CallOption) ({{.ReturnType}}, error) {
	return n.server.{{.Name}}(ctx, in)
}
{{end}}
func (n *Service) Init(s grpc.ServiceRegistrar) string {
	{{.ServiceName}}.Client = &{{.ServiceName}}LocalClient{server: n} // 本地直接调
	{{.ServiceName}}.Register{{.ServiceName | title}}Server(s, n)           // 将服务注册
	return "{{.ServiceName}}"                              // 服务名称
}
`

	funcMap := template.FuncMap{
		"title": strings.Title,
	}

	tmpl, err := template.New("package").Funcs(funcMap).Parse(packageTemplate)
	if err != nil {
		return err
	}

	data := struct {
		ServiceName string
		Methods     []MethodInfo
	}{
		ServiceName: serviceName,
		Methods:     methods,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return err
	}

	// 格式化代码
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	// 确保目录存在
	serviceDir := filepath.Join(servicePath, serviceName+"Service")
	err = os.MkdirAll(serviceDir, 0755)
	if err != nil {
		return err
	}

	// 写入文件
	filePath := filepath.Join(serviceDir, "package_gen.go")
	err = os.WriteFile(filePath, formatted, 0644)
	if err != nil {
		return err
	}

	slog.Info("Generated package file", "path", filePath)
	return nil
}
