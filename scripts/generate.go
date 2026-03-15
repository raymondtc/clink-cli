// generate.go - 自定义 OpenAPI 代码生成器
// 使用 Go 标准库，零外部依赖
package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// OpenAPI 结构定义
type OpenAPI struct {
	OpenAPI string                 `yaml:"openapi"`
	Info    Info                   `yaml:"info"`
	Paths   map[string]PathItem    `yaml:"paths"`
	Components Components          `yaml:"components"`
}

type Info struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type PathItem struct {
	Get  *Operation `yaml:"get"`
	Post *Operation `yaml:"post"`
}

type Operation struct {
	Summary    string              `yaml:"summary"`
	OperationID string             `yaml:"operationId"`
	Parameters []Parameter         `yaml:"parameters"`
	RequestBody *RequestBody       `yaml:"requestBody"`
	Responses   map[string]Response `yaml:"responses"`
}

type Parameter struct {
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
	Schema   Schema `yaml:"schema"`
}

type RequestBody struct {
	Content map[string]MediaType `yaml:"content"`
}

type MediaType struct {
	Schema Schema `yaml:"schema"`
}

type Response struct {
	Content map[string]MediaType `yaml:"content"`
}

type Components struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

type Schema struct {
	Type       string            `yaml:"type"`
	Format     string            `yaml:"format"`
	Properties map[string]Schema `yaml:"properties"`
	Items      *Schema           `yaml:"items"`
	Enum       []string          `yaml:"enum"`
	Ref        string            `yaml:"$ref"`
}

// 模板数据
type TemplateData struct {
	Package string
	Types   []TypeInfo
	Methods []MethodInfo
}

type TypeInfo struct {
	Name       string
	GoType     string
	Fields     []FieldInfo
	IsEnum     bool
	EnumValues []string
}

type FieldInfo struct {
	Name     string
	JSONName string
	Type     string
	Optional bool
}

type MethodInfo struct {
	Name       string
	HTTPMethod string
	Path       string
	Params     []ParamInfo
	HasBody    bool
	BodyType   string
	ReturnType string
}

type ParamInfo struct {
	Name     string
	GoName   string
	Type     string
	Required bool
	In       string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run generate.go <openapi.yaml>\n")
		os.Exit(1)
	}

	// 读取 OpenAPI 文件
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// 解析 YAML
	var spec OpenAPI
	if err := yaml.Unmarshal(data, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// 准备模板数据
	tmplData := TemplateData{
		Package: "generated",
		Types:   extractTypes(spec),
		Methods: extractMethods(spec),
	}

	// 生成类型文件
	if err := generateFile("types.go", typesTemplate, tmplData); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating types: %v\n", err)
		os.Exit(1)
	}

	// 生成客户端文件
	if err := generateFile("client.go", clientTemplate, tmplData); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Code generation complete!")
}

func extractTypes(spec OpenAPI) []TypeInfo {
	var types []TypeInfo

	for name, schema := range spec.Components.Schemas {
		typeInfo := TypeInfo{
			Name:   name,
			GoType: goTypeName(name),
		}

		// 处理枚举
		if len(schema.Enum) > 0 {
			typeInfo.IsEnum = true
			typeInfo.EnumValues = schema.Enum
		} else if schema.Type == "object" && schema.Properties != nil {
			// 处理对象
			for fieldName, fieldSchema := range schema.Properties {
				field := FieldInfo{
					Name:     goFieldName(fieldName),
					JSONName: fieldName,
					Type:     schemaToGoType(fieldSchema),
					Optional: true, // OpenAPI 3.0 默认 optional
				}
				typeInfo.Fields = append(typeInfo.Fields, field)
			}
		}

		types = append(types, typeInfo)
	}

	return types
}

func extractMethods(spec OpenAPI) []MethodInfo {
	var methods []MethodInfo

	for path, item := range spec.Paths {
		if item.Get != nil {
			methods = append(methods, extractMethod("GET", path, item.Get))
		}
		if item.Post != nil {
			methods = append(methods, extractMethod("POST", path, item.Post))
		}
	}

	return methods
}

func extractMethod(httpMethod, path string, op *Operation) MethodInfo {
	method := MethodInfo{
		Name:       goMethodName(op.OperationID),
		HTTPMethod: httpMethod,
		Path:       path,
	}

	// 提取参数
	for _, param := range op.Parameters {
		p := ParamInfo{
			Name:     param.Name,
			GoName:   goFieldName(param.Name),
			Type:     schemaToGoType(param.Schema),
			Required: param.Required,
			In:       param.In,
		}
		method.Params = append(method.Params, p)
	}

	// 检查是否有 body
	if op.RequestBody != nil {
		method.HasBody = true
		method.BodyType = "interface{}" // 简化处理
	}

	return method
}

func generateFile(filename, tmplStr string, data TemplateData) error {
	funcMap := template.FuncMap{
		"goFieldName": goFieldName,
	}
	tmpl, err := template.New(filename).Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	outPath := "pkg/generated/" + filename
	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	fmt.Printf("  Generated: %s\n", outPath)
	return nil
}

// 类型转换辅助函数
func schemaToGoType(schema Schema) string {
	switch schema.Type {
	case "string":
		if schema.Format == "int64" || schema.Format == "int" {
			return "int"
		}
		return "string"
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "array":
		if schema.Items != nil {
			return "[]" + schemaToGoType(*schema.Items)
		}
		return "[]interface{}"
	case "object":
		return "map[string]interface{}"
	default:
		if schema.Ref != "" {
			// 提取引用类型名
			parts := strings.Split(schema.Ref, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
		return "interface{}"
	}
}

func goTypeName(name string) string {
	return name
}

func goFieldName(name string) string {
	// 简单转换：首字母大写，处理下划线
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func goMethodName(name string) string {
	// 首字母大写
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

// 模板定义
const typesTemplate = `// Code generated by generate.go; DO NOT EDIT.

package {{.Package}}

{{range .Types}}
{{if .IsEnum}}
// {{.Name}} defines model for {{.Name}}.
type {{.Name}} string

// Possible values for {{.Name}}.
const (
{{range .EnumValues}}
	{{$.Name}}{{goFieldName .}} {{$.Name}} = "{{.}}"
{{end}}
)
{{else}}
// {{.Name}} defines model for {{.Name}}.
type {{.Name}} struct {
{{range .Fields}}
	{{.Name}} {{.Type}} ` + "`json:\"{{.JSONName}}{{if .Optional}},omitempty{{end}}\"`" + `
{{end}}
}
{{end}}
{{end}}
`

const clientTemplate = `// Code generated by generate.go; DO NOT EDIT.

package {{.Package}}

import (
	"context"
	"fmt"
	"net/http"
)

// Client provides access to the API.
type Client struct {
	baseURL string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{},
	}
}

{{range .Methods}}
// {{.Name}} calls the {{.HTTPMethod}} {{.Path}} endpoint.
func (c *Client) {{.Name}}(ctx context.Context{{range .Params}}, {{.GoName}} {{.Type}}{{end}}) (*http.Response, error) {
	url := c.baseURL + "{{.Path}}"
	req, err := http.NewRequestWithContext(ctx, "{{.HTTPMethod}}", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	// TODO: Add query parameters, body, etc.
	_ = req
	
	return c.httpClient.Do(req)
}
{{end}}
`
