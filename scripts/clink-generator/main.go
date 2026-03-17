// clink-generator - 基于配置的 CLI 代码生成器
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// ==================== 配置结构 ====================

type GeneratorConfig struct {
	Version   string              `yaml:"version"`
	Global    GlobalConfig        `yaml:"global"`
	Endpoints map[string]Endpoint `yaml:"endpoints"`
}

type GlobalConfig struct {
	OutputFormat    string `yaml:"outputFormat"`
	DefaultPageSize int    `yaml:"defaultPageSize"`
	TimeFormat      string `yaml:"timeFormat"`
}

type Endpoint struct {
	Command     []string     `yaml:"command"`
	Description string       `yaml:"description"`
	Use         string       `yaml:"use,omitempty"`
	Args        []ArgConfig  `yaml:"args,omitempty"`
	Flags       []FlagConfig `yaml:"flags"`
}

type ArgConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

type FlagConfig struct {
	Param       string `yaml:"param"`
	Flag        string `yaml:"flag"`
	Shorthand   string `yaml:"shorthand,omitempty"`
	Type        string `yaml:"type,omitempty"`  // 覆盖自动推断的类型
	Description string `yaml:"description"`
	Default     string `yaml:"default,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
	Source      string `yaml:"source,omitempty"`
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
	Summary     string      `yaml:"summary"`
	OperationID string      `yaml:"operationId"`
	Parameters  []Parameter `yaml:"parameters,omitempty"`
	RequestBody *RequestBody `yaml:"requestBody,omitempty"`
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
}

// ==================== 生成器 ====================

type Generator struct {
	Config *GeneratorConfig
	Spec   *OpenAPISpec
}

func NewGenerator(configPath, specPath string) (*Generator, error) {
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

	return &Generator{Config: &config, Spec: &spec}, nil
}

func (g *Generator) Generate(outputDir string) error {
	commands := make(map[string][]EndpointInfo)
	
	for opID, endpoint := range g.Config.Endpoints {
		if len(endpoint.Command) == 0 {
			continue
		}
		op := g.findOperation(opID)
		if op == nil {
			fmt.Printf("⚠ Warning: operation %s not found\n", opID)
			continue
		}
		
		topCmd := endpoint.Command[0]
		info := EndpointInfo{
			OperationID: opID,
			Endpoint:    endpoint,
			Operation:   op,
		}
		commands[topCmd] = append(commands[topCmd], info)
	}

	if err := g.generateRoot(commands, filepath.Join(outputDir, "root_gen.go")); err != nil {
		return fmt.Errorf("generate root: %w", err)
	}

	for name, endpoints := range commands {
		filename := filepath.Join(outputDir, name+"_gen.go")
		if err := g.generateCommandFile(name, endpoints, filename); err != nil {
			return fmt.Errorf("generate %s: %w", name, err)
		}
	}

	if err := g.generateUtils(filepath.Join(outputDir, "utils_gen.go")); err != nil {
		return fmt.Errorf("generate utils: %w", err)
	}

	fmt.Printf("\n✓ Generated %d files\n", len(commands)+2)
	return nil
}

func (g *Generator) findOperation(opID string) *Operation {
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

type EndpointInfo struct {
	OperationID string
	Endpoint    Endpoint
	Operation   *Operation
}

func (g *Generator) generateRoot(commands map[string][]EndpointInfo, filename string) error {
	var cmdNames []string
	for name := range commands {
		cmdNames = append(cmdNames, name)
	}
	
	data := struct {
		Commands []string
	}{
		Commands: cmdNames,
	}
	
	return generateFromTemplate(filename, rootTemplate, data)
}

func (g *Generator) generateCommandFile(name string, endpoints []EndpointInfo, filename string) error {
	var subCommands []SubCommand
	
	for _, ep := range endpoints {
		cmdName := g.getSubcommandName(ep.Endpoint.Command)
		if cmdName == "" {
			cmdName = ep.OperationID
		}
		
		sc := SubCommand{
			Name:        cmdName,
			OperationID: ep.OperationID,
			Description: ep.Endpoint.Description,
			Use:         ep.Endpoint.Use,
			ParentCmd:   name,
		}
		
		for _, f := range ep.Endpoint.Flags {
			if f.Flag == "" {
				continue
			}
			
			flagType := g.getParamType(ep.Operation, f.Param)
			if f.Type != "" {
				flagType = f.Type  // 配置中指定的类型优先
			}
			varName := g.toVarName(f.Flag)
			cobraType := g.toCobraType(flagType)
			
			flag := FlagDef{
				VarName:     varName,
				Name:        f.Flag,
				Shorthand:   f.Shorthand,
				Type:        flagType,
				CobraType:   cobraType,
				Default:     g.formatDefault(f.Default, flagType),
				Description: f.Description,
				Required:    f.Required,
			}
			sc.Flags = append(sc.Flags, flag)
			sc.FlagVars = append(sc.FlagVars, FlagVar{Name: varName, Type: flagType})
		}
		
		for i, arg := range ep.Endpoint.Args {
			sc.Args = append(sc.Args, ArgDef{
				Name:  arg.Name,
				Type:  "string",
				Index: i,
			})
		}
		
		sc.APICall = g.buildAPICall(ep, cmdName)
		subCommands = append(subCommands, sc)
	}
	
	data := CommandFileData{
		Package:     "main",
		CommandName: name,
		SubCommands: subCommands,
	}
	
	return generateFromTemplate(filename, commandTemplate, data)
}

func (g *Generator) generateUtils(filename string) error {
	return generateFromTemplate(filename, utilsTemplate, map[string]string{
		"Package": "main",
	})
}

func (g *Generator) getSubcommandName(cmd []string) string {
	if len(cmd) <= 1 {
		return ""
	}
	return strings.Join(cmd[1:], "_")
}

func (g *Generator) toVarName(s string) string {
	// 避免使用 Go 关键字
	keywords := map[string]bool{
		"type": true, "map": true, "chan": true, "func": true,
		"var": true, "const": true, "package": true, "import": true,
	}
	
	parts := strings.Split(s, "-")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += strings.Title(parts[i])
	}
	
	// 如果是关键字，添加后缀
	if keywords[result] {
		result += "Val"
	}
	
	return result
}

func (g *Generator) getParamType(op *Operation, paramName string) string {
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

func (g *Generator) formatDefault(def, t string) string {
	if def == "" {
		if t == "string" {
			return `""`
		} else if t == "int" {
			return "0"
		} else if t == "bool" {
			return "false"
		}
		return ""
	}
	if t == "string" {
		return fmt.Sprintf(`"%s"`, def)
	}
	return def
}

func (g *Generator) buildAPICall(ep EndpointInfo, cmdName string) string {
	switch ep.OperationID {
	case "listCdrIbs":
		return fmt.Sprintf("api.GetInboundRecords(ctx, %sFlags.start, %sFlags.end, %sFlags.phone, %sFlags.agent, %sFlags.offset/%sFlags.limit+1, %sFlags.limit)", cmdName, cmdName, cmdName, cmdName, cmdName, cmdName, cmdName)
	case "listCdrObs":
		return fmt.Sprintf("api.GetOutboundRecords(ctx, %sFlags.start, %sFlags.end, %sFlags.phone, %sFlags.agent, %sFlags.offset/%sFlags.limit+1, %sFlags.limit)", cmdName, cmdName, cmdName, cmdName, cmdName, cmdName, cmdName)
	case "listAgentStatus":
		return fmt.Sprintf("api.GetAgentStatus(ctx, %sFlags.agent)", cmdName)
	case "webcall":
		return fmt.Sprintf("api.Webcall(ctx, %sFlags.phone, %sFlags.clid, %sFlags.ivr, nil)", cmdName, cmdName, cmdName)
	case "callout":
		return fmt.Sprintf("api.MakeCall(ctx, %sFlags.phone, %sFlags.agent, %sFlags.clid)", cmdName, cmdName, cmdName)
	case "online":
		return fmt.Sprintf("api.Online(ctx, %sFlags.agent, %sFlags.queue, %sFlags.tel, %sFlags.bindType)", cmdName, cmdName, cmdName, cmdName)
	case "offline":
		return fmt.Sprintf("api.Offline(ctx, %sFlags.agent)", cmdName)
	case "pause":
		return fmt.Sprintf("api.Pause(ctx, %sFlags.agent, %sFlags.pauseType, %sFlags.reason)", cmdName, cmdName, cmdName)
	case "unpause":
		return fmt.Sprintf("api.Unpause(ctx, %sFlags.agent)", cmdName)
	case "unlink":
		return fmt.Sprintf("api.Hangup(ctx, %sFlags.agent)", cmdName)
	case "hold":
		return fmt.Sprintf("api.Hold(ctx, %sFlags.agent)", cmdName)
	case "unhold":
		return fmt.Sprintf("api.Unhold(ctx, %sFlags.agent)", cmdName)
	case "transfer":
		return fmt.Sprintf("api.Transfer(ctx, %sFlags.agent, %sFlags.transferType, %sFlags.target)", cmdName, cmdName, cmdName)
	case "getQueueStatus":
		return fmt.Sprintf("api.GetQueueStatus(ctx, %sFlags.queue)", cmdName)
	case "listQueues":
		return fmt.Sprintf("api.ListQueues(ctx, %sFlags.offset, %sFlags.limit)", cmdName, cmdName)
	default:
		return fmt.Sprintf("api.%s(ctx)", g.toPascalCase(ep.OperationID))
	}
}

func (g *Generator) toCobraType(t string) string {
	switch t {
	case "int":
		return "Int"
	case "bool":
		return "Bool"
	default:
		return "String"
	}
}

func (g *Generator) toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}

// ==================== 模板数据 ====================

type CommandFileData struct {
	Package     string
	CommandName string
	SubCommands []SubCommand
}

type SubCommand struct {
	Name        string
	OperationID string
	Description string
	Use         string
	ParentCmd   string
	Flags       []FlagDef
	FlagVars    []FlagVar
	Args        []ArgDef
	APICall     string
}

type FlagDef struct {
	VarName     string
	Name        string
	Shorthand   string
	Type        string
	CobraType   string
	Default     string
	Description string
	Required    bool
}

type FlagVar struct {
	Name string
	Type string
}

type ArgDef struct {
	Name  string
	Type  string
	Index int
}

// ==================== 模板 ====================

const rootTemplate = `// Code generated by clink-generator; DO NOT EDIT.

package main

import "github.com/spf13/cobra"

func init() {
{{range .Commands}}	rootCmd.AddCommand({{.}}Cmd)
{{end}}}

{{range .Commands}}var {{.}}Cmd = &cobra.Command{
	Use:   "{{.}}",
	Short: "{{.}} related commands",
}
{{end}}
`

const commandTemplate = `// Code generated by clink-generator; DO NOT EDIT.
package main

import (
	"context"
	"github.com/raymondtc/clink-cli/pkg/response"
	"github.com/spf13/cobra"
)
{{range .SubCommands}}
{{$cmdName := .Name}}
var {{.Name}}Flags struct {
{{range .FlagVars}}	{{.Name}} {{.Type}}
{{end}}}

var {{.Name}}Cmd = &cobra.Command{
	Use:   "{{if .Use}}{{.Use}}{{else}}{{.Name}}{{end}}",
	Short: "{{.Description}}",
{{if .Args}}	Args:  cobra.ExactArgs({{len .Args}}),
{{end}}	RunE:  run{{.Name}},
}

func init() {
	{{.ParentCmd}}Cmd.AddCommand({{.Name}}Cmd)
{{range .Flags}}{{if .Shorthand}}	{{$cmdName}}Cmd.Flags().{{.CobraType}}VarP(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", "{{.Shorthand}}", {{.Default}}, "{{.Description}}"){{else}}	{{$cmdName}}Cmd.Flags().{{.CobraType}}Var(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", {{.Default}}, "{{.Description}}"){{end}}{{if .Required}}
	{{$cmdName}}Cmd.MarkFlagRequired("{{.Name}}"){{end}}
{{end}}}

func run{{.Name}}(cmd *cobra.Command, args []string) error {
	api, err := createAPI()
	if err != nil {
		return err
	}
	ctx := context.Background()
{{range .Args}}	{{.Name}} := args[{{.Index}}]
{{end}}	_, err = {{.APICall}}
	return response.Wrap("{{.Name}}", err)
}
{{end}}
`

const utilsTemplate = `// Code generated by clink-generator; DO NOT EDIT.
package main

import "github.com/raymondtc/clink-cli/pkg/renderer"

func renderOutput(data interface{}) error {
	r := renderer.New(renderer.Format(outputFormat))
	return r.Render(data)
}
`

func generateFromTemplate(filename, tmplStr string, data interface{}) error {
	tmpl, err := template.New(filepath.Base(filename)).Parse(tmplStr)
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

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config.yaml> <openapi.yaml> [output-dir]\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	specPath := os.Args[2]
	outputDir := "cmd/clink"
	if len(os.Args) >= 4 {
		outputDir = os.Args[3]
	}

	fmt.Println("Clink CLI Generator")
	fmt.Println("===================")
	fmt.Printf("Config: %s\nSpec:   %s\nOutput: %s\n\n", configPath, specPath, outputDir)

	gen, err := NewGenerator(configPath, specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := gen.Generate(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Done!")
}
