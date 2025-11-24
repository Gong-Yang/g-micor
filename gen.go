// 此文件用于根据proto文件自动生成代码，不参与实际编译
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const (
	packageName = "agent" // 需要生成的包名\
)

// 方法信息
type MethodInfo struct {
	Name       string
	ParamType  string
	ReturnType string
}

// Proto文件中的Service信息
type ProtoServiceInfo struct {
	ServiceName string
	Methods     []MethodInfo
}

func main() {
	slog.Info("Starting auto code generation", "package", packageName)

	// 1. 执行protoc命令生成go文件
	err := generateProtoCode(packageName)
	if err != nil {
		log.Fatal("Failed to generate proto code:", err)
	}

	// 2. 扫描proto文件，找到service定义
	protoService, err := scanProtoService(packageName)
	if err != nil {
		log.Fatal("Failed to scan proto service:", err)
	}

	if protoService == nil {
		slog.Info("No service found in proto file")
		return
	}

	slog.Info("Found proto service", "service", protoService.ServiceName, "methods", len(protoService.Methods))

	// 3. 检查并补充Service结构体的方法
	err = ensureServiceMethods(packageName, protoService)
	if err != nil {
		log.Fatal("Failed to ensure service methods:", err)
	}

	// 4. 生成contract_gen.go文件
	err = generateContractFile(packageName, protoService)
	if err != nil {
		log.Fatal("Failed to generate contract file:", err)
	}

	// 5. 生成localAdapter_gen.go文件
	err = generatePackageFile(packageName, protoService)
	if err != nil {
		log.Fatal("Failed to generate package file:", err)
	}

	slog.Info("Code generation completed successfully", "package", packageName)
}

// 执行protoc命令生成go文件
func generateProtoCode(pkg string) error {
	protoFile := fmt.Sprintf("contract/%sC/%s.proto", pkg, pkg)

	// 检查proto文件是否存在
	if _, err := os.Stat(protoFile); os.IsNotExist(err) {
		return fmt.Errorf("proto file not found: %s", protoFile)
	}

	cmd := exec.Command("protoc",
		"--go_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=.",
		"--go-grpc_opt=paths=source_relative",
		protoFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("protoc command failed: %v, output: %s", err, string(output))
	}

	slog.Info("Proto code generated successfully", "package", pkg)
	return nil
}

// 扫描proto文件，解析service定义
func scanProtoService(pkg string) (*ProtoServiceInfo, error) {
	protoFile := fmt.Sprintf("contract/%sC/%s.proto", pkg, pkg)

	file, err := os.Open(protoFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var serviceName string
	var methods []MethodInfo
	inService := false

	// 正则表达式匹配service和rpc定义
	serviceRegex := regexp.MustCompile(`^\s*service\s+(\w+)\s*\{`)
	rpcRegex := regexp.MustCompile(`^\s*rpc\s+(\w+)\s*\(\s*(\w+)\s*\)\s*returns\s*\(\s*(\w+)\s*\)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过注释行
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		// 检查service定义
		if matches := serviceRegex.FindStringSubmatch(line); matches != nil {
			serviceName = matches[1]
			inService = true
			continue
		}

		// 检查service结束
		if inService && line == "}" {
			inService = false
			break
		}

		// 解析rpc方法
		if inService {
			if matches := rpcRegex.FindStringSubmatch(line); matches != nil {
				methodName := matches[1]
				paramType := "*" + pkg + "C" + "." + matches[2]
				returnType := "*" + pkg + "C" + "." + matches[3]

				methods = append(methods, MethodInfo{
					Name:       methodName,
					ParamType:  paramType,
					ReturnType: returnType,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if serviceName == "" {
		return nil, nil
	}

	return &ProtoServiceInfo{
		ServiceName: serviceName,
		Methods:     methods,
	}, nil
}

// 确保Service结构体包含所有必要的方法
func ensureServiceMethods(pkg string, protoService *ProtoServiceInfo) error {
	serviceDir := fmt.Sprintf("module/%s/endpoint", pkg)
	serviceFile := filepath.Join(serviceDir, "rpcServer.go")

	// 检查service.go文件是否存在
	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		// 如果不存在，创建基础的service.go文件
		err = createBaseServiceFile(serviceFile, pkg, protoService)
		if err != nil {
			return err
		}
		slog.Info("Created base service file", "file", serviceFile)
		return nil
	}

	// 解析现有的service.go文件
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, serviceFile, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// 收集现有方法
	existingMethods := make(map[string]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				if getReceiverType(fn.Recv.List[0]) == "*RPCServer" {
					existingMethods[fn.Name.Name] = true
				}
			}
		}
		return true
	})

	// 检查是否需要添加方法
	var missingMethods []MethodInfo
	for _, method := range protoService.Methods {
		if !existingMethods[method.Name] {
			missingMethods = append(missingMethods, method)
		}
	}

	// 如果有缺失的方法，添加它们
	if len(missingMethods) > 0 {
		err = addMissingMethods(serviceFile, pkg, missingMethods)
		if err != nil {
			return err
		}
		slog.Info("Added missing methods", "count", len(missingMethods), "file", serviceFile)
	}

	return nil
}

// 创建基础的service.go文件
func createBaseServiceFile(filePath, pkg string, protoService *ProtoServiceInfo) error {
	const baseServiceTemplate = `package endpoint

import (
	"context"
	"github.com/Gong-Yang/GGYYNet/contract/{{.Package}}C"
)

type RPCServer struct {
	{{.Package}}C.Unimplemented{{.ServiceName}}Server
}
{{range .Methods}}
func (s *RPCServer) {{.Name}}(ctx context.Context, req {{.ParamType}}) ({{.ReturnType}}, error) {
	panic("implement me")
}
{{end}}
`

	funcMap := template.FuncMap{}
	tmpl, err := template.New("baseService").Funcs(funcMap).Parse(baseServiceTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Package     string
		ServiceName string
		Methods     []MethodInfo
	}{
		Package:     pkg,
		ServiceName: protoService.ServiceName,
		Methods:     protoService.Methods,
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
	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(filePath, formatted, 0644)
}

// 向现有service.go文件添加缺失的方法
func addMissingMethods(filePath, pkg string, missingMethods []MethodInfo) error {
	// 读取现有文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 生成要添加的方法代码
	const methodTemplate = `
func (s *RPCServer) {{.Name}}(ctx context.Context, req {{.ParamType}}) ({{.ReturnType}}, error) {
	panic("implement me")
}
`

	var methodsCode strings.Builder
	for _, method := range missingMethods {
		tmpl, err := template.New("method").Parse(methodTemplate)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, method)
		if err != nil {
			return err
		}

		methodsCode.WriteString(buf.String())
	}

	// 将新方法添加到文件末尾
	newContent := string(content) + methodsCode.String()

	// 格式化代码
	formatted, err := format.Source([]byte(newContent))
	if err != nil {
		return err
	}

	// 写回文件
	return os.WriteFile(filePath, formatted, 0644)
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

// 去掉类型名的包名前缀
func removePackagePrefix(typeName string) string {
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) == 2 {
			if strings.HasPrefix(parts[0], "*") {
				return "*" + parts[1]
			} else {
				return parts[1]
			}
		}
	}
	return typeName
}

// 生成contract_gen.go文件
func generateContractFile(pkg string, protoService *ProtoServiceInfo) error {
	const contractTemplate = `package {{.Package}}C

import (
	"context"
	"github.com/Gong-Yang/g-micor/discover"
	"google.golang.org/grpc"
	"log/slog"
)

var ModuleName = "{{.Package}}"

var Client {{.ServiceName}}Client = &{{.Package}}RemoteClient{}

type {{.Package}}RemoteClient struct {
	client {{.ServiceName}}Client
}

func (s *{{.Package}}RemoteClient) init() error {
	c, err := discover.Grpc("{{.Package}}")
	if err != nil {
		return err
	}
	client := New{{.ServiceName}}Client(c)
	s.client = client
	slog.Info("{{.Package}} remote client init")
	return nil
}
{{range .Methods}}
func (s *{{$.Package}}RemoteClient) {{.Name}}(ctx context.Context, in {{.ParamType | removePackagePrefix}}, opts ...grpc.CallOption) ({{.ReturnType | removePackagePrefix}}, error) {
	if s.client == nil {
		err := s.init()
		if err != nil {
			return nil, err
		}
	}
	return s.client.{{.Name}}(ctx, in, opts...)
}
{{end}}
`

	funcMap := template.FuncMap{
		"removePackagePrefix": removePackagePrefix,
	}

	tmpl, err := template.New("contract").Funcs(funcMap).Parse(contractTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Package     string
		ServiceName string
		Methods     []MethodInfo
	}{
		Package:     pkg,
		ServiceName: protoService.ServiceName,
		Methods:     protoService.Methods,
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
	contractDir := fmt.Sprintf("contract/%sC", pkg)
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

// 生成localAdapter_gen.go文件
func generatePackageFile(pkg string, protoService *ProtoServiceInfo) error {
	const packageTemplate = `package endpoint

import (
	"context"
	"github.com/Gong-Yang/GGYYNet/contract/{{.Package}}C"
	"google.golang.org/grpc"
)

func InitRPC(register grpc.ServiceRegistrar) {
	s := &RPCServer{}
	{{.Package}}C.Client = &localAdapter{server: s}    // 本地直接调
	{{.Package}}C.Register{{.ServiceName}}Server(register, s)  // 将服务注册
	return
}

type localAdapter struct {
	server *RPCServer
}
{{range .Methods}}
func (s *localAdapter) {{.Name}}(ctx context.Context, in {{.ParamType}}, opts ...grpc.CallOption) ({{.ReturnType}}, error) {
	return s.server.{{.Name}}(ctx, in)
}
{{end}}
`

	tmpl, err := template.New("package").Parse(packageTemplate)
	if err != nil {
		return err
	}

	data := struct {
		Package     string
		ServiceName string
		Methods     []MethodInfo
	}{
		Package:     pkg,
		ServiceName: protoService.ServiceName,
		Methods:     protoService.Methods,
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
	serviceDir := fmt.Sprintf("module/%s/endpoint", pkg)
	err = os.MkdirAll(serviceDir, 0755)
	if err != nil {
		return err
	}

	// 写入文件
	filePath := filepath.Join(serviceDir, "localAdapter_gen.go")
	err = os.WriteFile(filePath, formatted, 0644)
	if err != nil {
		return err
	}

	slog.Info("Generated package file", "path", filePath)
	return nil
}
