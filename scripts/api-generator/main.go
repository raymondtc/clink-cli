// api-generator - 生成 API 层代码
// 根据 config/generator.yaml 生成 pkg/api/generated_api.go 中的方法
package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// 配置结构（复用 cli-generator 的定义）
type GeneratorConfig struct {
	Version   string              `yaml:"version"`
	Endpoints map[string]Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Command     []string    `yaml:"command"`
	Description string      `yaml:"description"`
	Flags       []FlagConfig `yaml:"flags"`
}

type FlagConfig struct {
	Param    string `yaml:"param"`
	Flag     string `yaml:"flag"`
	Type     string `yaml:"type,omitempty"`
	Required bool   `yaml:"required,omitempty"`
}

// OpenAPI 结构
type OpenAPISpec struct {
	Paths map[string]PathItem `yaml:"paths"`
}

type PathItem struct {
	Get  *Operation `yaml:"get"`
	Post *Operation `yaml:"post"`
}

type Operation struct {
	OperationID string       `yaml:"operationId"`
	Parameters  []Parameter  `yaml:"parameters,omitempty"`
	RequestBody *RequestBody `yaml:"requestBody,omitempty"`
	Summary     string       `yaml:"summary"`
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

type Schema struct {
	Type       string            `yaml:"type"`
	Format     string            `yaml:"format,omitempty"`
	Properties map[string]Schema `yaml:"properties,omitempty"`
	Required   []string          `yaml:"required,omitempty"`
}

// APIGenerator 生成 API 方法
type APIGenerator struct {
	Config *GeneratorConfig
	Spec   *OpenAPISpec
}

func NewAPIGenerator(configPath, specPath string) (*APIGenerator, error) {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config GeneratorConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	specData, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}

	return &APIGenerator{
		Config: &config,
		Spec:   &spec,
	}, nil
}

// Generate 生成 API 文件
func (g *APIGenerator) Generate(filename string) error {
	methods := []APIMethod{}

	for opID, endpoint := range g.Config.Endpoints {
		op := g.findOperation(opID)
		if op == nil {
			fmt.Printf("⚠ Warning: operation %s not found\n", opID)
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

func (g *APIGenerator) buildMethod(opID string, endpoint Endpoint, op *Operation) APIMethod {
	method := APIMethod{
		Name:        g.getAPIMethodName(opID),
		Description: op.Summary,
		OperationID: opID,
	}

	// 确定 HTTP 方法
	for _, pathItem := range g.Spec.Paths {
		if pathItem.Get != nil && pathItem.Get.OperationID == opID {
			method.HTTPMethod = "GET"
			break
		}
		if pathItem.Post != nil && pathItem.Post.OperationID == opID {
			method.HTTPMethod = "POST"
			break
		}
	}

	// 构建参数
	for _, flag := range endpoint.Flags {
		if flag.Param == "" {
			continue
		}

		paramType := g.getParamType(op, flag.Param)
		if flag.Type != "" {
			paramType = flag.Type
		}
		
		param := APIParam{
			Name:     flag.Param,
			VarName:  g.toVarName(flag.Param),
			Type:     paramType,
			Required: flag.Required,
		}
		method.Params = append(method.Params, param)
	}

	// 构建函数体
	method.Body = g.buildMethodBody(opID, method)

	return method
}

func (g *APIGenerator) toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}

func (g *APIGenerator) getAPIMethodName(opID string) string {
	// 映射 operationId 到 API 方法名
	switch opID {
	case "listCdrIbs":
		return "GetInboundRecords"
	case "listCdrObs":
		return "GetOutboundRecords"
	case "listAgentStatus":
		return "GetAgentStatus"
	case "webcall":
		return "Webcall"
	case "callout":
		return "MakeCall"
	case "online":
		return "Online"
	case "offline":
		return "Offline"
	case "pause":
		return "Pause"
	case "unpause":
		return "Unpause"
	case "unlink":
		return "Hangup"
	case "hold":
		return "Hold"
	case "unhold":
		return "Unhold"
	case "transfer":
		return "Transfer"
	case "getQueueStatus":
		return "GetQueueStatus"
	case "listQueues":
		return "ListQueues"
	default:
		return g.toPascalCase(opID)
	}
}

func (g *APIGenerator) toVarName(s string) string {
	parts := strings.Split(s, "_")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += strings.Title(parts[i])
	}
	return result
}

func (g *APIGenerator) getParamType(op *Operation, paramName string) string {
	// 从 OpenAPI 参数中查找
	for _, p := range op.Parameters {
		if p.Name == paramName {
			switch p.Schema.Type {
			case "integer":
				return "int"
			case "boolean":
				return "bool"
			default:
				return "string"
			}
		}
	}

	// 从 requestBody 中查找
	if op.RequestBody != nil {
		for _, media := range op.RequestBody.Content {
			if schema, ok := media.Schema.Properties[paramName]; ok {
				switch schema.Type {
				case "integer":
					return "int"
				case "boolean":
					return "bool"
				default:
					return "string"
				}
			}
		}
	}

	return "string"
}

func (g *APIGenerator) buildMethodBody(opID string, method APIMethod) string {
	var b strings.Builder

	// 构建参数检查
	for _, p := range method.Params {
		if p.Required {
			if p.Type == "string" {
				b.WriteString(fmt.Sprintf(`	if %s == "" {
		return nil, fmt.Errorf("%s is required")
	}
`, p.VarName, p.Name))
			} else if p.Type == "int" {
				b.WriteString(fmt.Sprintf(`	if %s == 0 {
		return nil, fmt.Errorf("%s is required")
	}
`, p.VarName, p.Name))
			}
		}
	}

	// 构建请求参数
	if method.HTTPMethod == "GET" {
		b.WriteString("\n\tparams := map[string]string{}\n")
		for _, p := range method.Params {
			if p.Type == "string" {
				b.WriteString(fmt.Sprintf(`	if %s != "" {
		params["%s"] = %s
	}
`, p.VarName, p.Name, p.VarName))
			} else if p.Type == "int" {
				b.WriteString(fmt.Sprintf(`	if %s != 0 {
		params["%s"] = strconv.Itoa(%s)
	}
`, p.VarName, p.Name, p.VarName))
			}
		}
		b.WriteString(fmt.Sprintf(`
	resp, err := a.client.Request(ctx, "GET", "/cc/%s", params, nil)
`, strings.ToLower(opID)))
	} else {
		b.WriteString("\n\tbody := map[string]interface{}{}\n")
		for _, p := range method.Params {
			b.WriteString(fmt.Sprintf(`	body["%s"] = %s
`, p.Name, p.VarName))
		}
		b.WriteString(fmt.Sprintf(`
	bodyJSON, _ := json.Marshal(body)
	resp, err := a.client.Request(ctx, "POST", "/cc/%s", nil, strings.NewReader(string(bodyJSON)))
`, strings.ToLower(opID)))
	}

	// 通用错误处理
	b.WriteString(`
	if err != nil {
		return nil, response.Wrap("` + opID + `", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("empty response")
	}
`)

	return b.String()
}

// APIMethod 表示一个 API 方法
type APIMethod struct {
	Name        string
	Description string
	OperationID string
	HTTPMethod  string
	Params      []APIParam
	Body        string
}

// APIParam 表示一个参数
type APIParam struct {
	Name     string
	VarName  string
	Type     string
	Required bool
}

// 模板
const apiTemplate = `// Code generated by api-generator; DO NOT EDIT.

package {{.Package}}

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/raymondtc/clink-cli/pkg/response"
)

{{range .Methods}}
// {{.Name}} - {{.Description}}
func (a *API) {{.Name}}(ctx context.Context{{range .Params}}, {{.VarName}} {{.Type}}{{end}}) (interface{}, error) {
{{.Body}}

	return resp, nil
}
{{end}}
`

func generateFromTemplate(filename, tmplStr string, data interface{}) error {
	tmpl, err := template.New("api").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config.yaml> <openapi.yaml> [output.go]\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	specPath := os.Args[2]
	outputFile := "pkg/api/auto_generated.go"
	if len(os.Args) >= 4 {
		outputFile = os.Args[3]
	}

	gen, err := NewAPIGenerator(configPath, specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := gen.Generate(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Generated API methods to %s\n", outputFile)
}
