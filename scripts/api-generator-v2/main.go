// api-generator-v2 - 新版 API 代码生成器
// 基于 config/generator.v2.yaml 生成 pkg/api/auto_generated.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/raymondtc/clink-cli/pkg/codegen"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config.v2.yaml> <openapi.yaml> [output.go]\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	specPath := os.Args[2]
	outputFile := "pkg/api/auto_generated.go"
	if len(os.Args) >= 4 {
		outputFile = os.Args[3]
	}

	fmt.Println("API Generator v2")
	fmt.Println("================")
	fmt.Printf("Config: %s\nSpec:   %s\nOutput: %s\n\n", configPath, specPath, outputFile)

	gen, err := NewAPIGenerator(configPath, specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := gen.Generate(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Done!")
}

// APIGenerator 生成 API 方法
type APIGenerator struct {
	Config *codegen.GeneratorConfig
	Spec   *OpenAPISpec
}

// OpenAPISpec 表示 OpenAPI 规范
type OpenAPISpec struct {
	Paths map[string]PathItem `yaml:"paths"`
}

// PathItem 表示路径项
type PathItem struct {
	Get  *Operation `yaml:"get"`
	Post *Operation `yaml:"post"`
}

// Operation 表示操作
type Operation struct {
	Summary     string      `yaml:"summary"`
	OperationID string      `yaml:"operationId"`
	Parameters  []Parameter `yaml:"parameters,omitempty"`
	RequestBody *RequestBody `yaml:"requestBody,omitempty"`
}

// Parameter 表示参数
type Parameter struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
	Schema   Schema `yaml:"schema"`
}

// RequestBody 表示请求体
type RequestBody struct {
	Content map[string]MediaType `yaml:"content"`
}

// MediaType 表示媒体类型
type MediaType struct {
	Schema Schema `yaml:"schema"`
}

// Schema 表示模式
type Schema struct {
	Type       string            `yaml:"type"`
	Format     string            `yaml:"format,omitempty"`
	Properties map[string]Schema `yaml:"properties,omitempty"`
}

// NewAPIGenerator 创建新的 API 生成器
func NewAPIGenerator(configPath, specPath string) (*APIGenerator, error) {
	// 读取配置
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config codegen.GeneratorConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 读取 OpenAPI 规范
	specData, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	return &APIGenerator{Config: &config, Spec: &spec}, nil
}

// Generate 生成 API 文件
func (g *APIGenerator) Generate(filename string) error {
	methods := []APIMethod{}

	for opID, endpoint := range g.Config.Endpoints {
		op := g.findOperation(endpoint.OperationID)
		if op == nil {
			fmt.Printf("⚠ Warning: operation %s not found\n", endpoint.OperationID)
			continue
		}

		method := g.buildMethod(opID, endpoint, op)
		methods = append(methods, method)
	}

	data := struct {
		Package string
		Methods []APIMethod
	}{
		Package: "api",
		Methods: methods,
	}

	return generateFromTemplate(filename, apiTemplate, data)
}

// findOperation 查找操作
func (g *APIGenerator) findOperation(opID string) *Operation {
	for _, pathItem := range g.Spec.Paths {
		if pathItem.Get != nil && pathItem.Get.OperationID == opID {
			return pathItem.Get
		}
		if pathItem.Post != nil && pathItem.Post.OperationID == opID {
			return pathItem.Post
		}
	}
	return nil
}

// buildMethod 构建方法
func (g *APIGenerator) buildMethod(opID string, endpoint codegen.EndpointConfig, op *Operation) APIMethod {
	method := APIMethod{
		Name:        g.toPascalCase(opID),
		OperationID: endpoint.OperationID,
		Description: endpoint.Description,
		ResponseType: endpoint.Response.Type,
		HasTimeTransform: len(endpoint.Request.Transforms) > 0,
	}

	// 确定 HTTP 方法
	for _, pathItem := range g.Spec.Paths {
		if pathItem.Get != nil && pathItem.Get.OperationID == endpoint.OperationID {
			method.HTTPMethod = "GET"
			break
		}
		if pathItem.Post != nil && pathItem.Post.OperationID == endpoint.OperationID {
			method.HTTPMethod = "POST"
			break
		}
	}

	// 构建参数
	for _, field := range endpoint.Parameters.Fields {
		param := APIParam{
			Name:      field.Name,
			VarName:   g.toVarName(field.Flag, field.Name),
			Type:      g.mapType(field.Type),
			Required:  field.Required,
			FlagName:  field.Flag,
			FieldName: g.toPascalCase(field.Name),
			ZeroValue: g.zeroValue(field.Type),
		}
		method.Params = append(method.Params, param)
	}

	// 获取生成的类型名称 (oapi-codegen 生成的是 PascalCase)
	method.GeneratedParamsType = fmt.Sprintf("generated.%sParams", g.toPascalCase(endpoint.OperationID))
	if method.HTTPMethod == "POST" {
		method.GeneratedParamsType = fmt.Sprintf("generated.%sJSONRequestBody", g.toPascalCase(endpoint.OperationID))
	}

	// 设置客户端方法名
	method.ClientMethodName = g.toPascalCase(endpoint.OperationID) + "WithResponse"

	return method
}

// APIMethod 表示 API 方法
type APIMethod struct {
	Name                string
	OperationID         string
	Description         string
	HTTPMethod          string
	Params              []APIParam
	ResponseType        string
	GeneratedParamsType string
	ClientMethodName    string // 客户端方法名
	HasTimeTransform    bool
}

// APIParam 表示 API 参数
type APIParam struct {
	Name      string
	VarName   string
	Type      string
	Required  bool
	FlagName  string
	FieldName string // PascalCase 字段名
	ZeroValue string // 类型的零值
}

// toPascalCase 转换为 PascalCase (使用 codegen 包)
func (g *APIGenerator) toPascalCase(s string) string {
	return codegen.ToPascalCase(s)
}

// toVarName 生成变量名 (使用 codegen 包)
func (g *APIGenerator) toVarName(flag, name string) string {
	if flag != "" {
		return codegen.ToValidIdentifier(flag)
	}
	return codegen.ToValidIdentifier(name)
}

// mapType 映射类型
func (g *APIGenerator) mapType(t string) string {
	switch t {
	case "int":
		return "int"
	case "bool":
		return "bool"
	default:
		return "string"
	}
}

// zeroValue 返回类型的零值
func (g *APIGenerator) zeroValue(t string) string {
	switch t {
	case "int":
		return "0"
	case "bool":
		return "false"
	default:
		return `""`
	}
}

// generateFromTemplate 从模板生成代码
func generateFromTemplate(filename, tmplStr string, data interface{}) error {
	tmpl, err := template.New(filepath.Base(filename)).Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	fmt.Printf("  Generated: %s\n", filename)
	return nil
}

const apiTemplate = `// Code generated by api-generator-v2; DO NOT EDIT.

package {{.Package}}

import (
	"context"
	"fmt"

	"github.com/raymondtc/clink-cli/pkg/generated"
)
{{- range .Methods }}

// {{ .Name }} - {{ .Description }}
func (a *GeneratedAPI) {{ .Name }}(ctx context.Context{{- range .Params }}, {{ .VarName }} {{ .Type }}{{- end }}) (interface{}, error) {
	// 构建请求参数
	params := &{{ .GeneratedParamsType }}{}
	{{- range .Params }}
	{{- if .Required }}
	params.{{ .FieldName }} = {{ .VarName }}
	{{- else }}
	if {{ .VarName }} != {{ .ZeroValue }} {
		params.{{ .FieldName }} = &{{ .VarName }}
	}
	{{- end }}
	{{- end }}

	// 调用 API
	{{- if eq .HTTPMethod "GET" }}
	resp, err := a.client.{{ .ClientMethodName }}(ctx, params)
	{{- else }}
	resp, err := a.client.{{ .ClientMethodName }}(ctx, *params)
	{{- end }}
	if err != nil {
		return nil, fmt.Errorf("{{ .OperationID }}: %w", err)
	}

	// 处理响应
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode())
	}

	return resp.JSON200, nil
}
{{- end }}
`
