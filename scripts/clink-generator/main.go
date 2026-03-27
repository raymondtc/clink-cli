package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

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
	Command        []string     `yaml:"command"`
	Description    string       `yaml:"description"`
	Use            string       `yaml:"use,omitempty"`
	ResultType     string       `yaml:"resultType,omitempty"`
	CustomTemplate string       `yaml:"customTemplate,omitempty"`
	Custom         CustomConfig `yaml:"custom,omitempty"`
	Args           []ArgConfig  `yaml:"args,omitempty"`
	Flags          []FlagConfig `yaml:"flags"`
}

type CustomConfig struct {
	TimeRange      bool   `yaml:"timeRange,omitempty"`
	Pagination     bool   `yaml:"pagination,omitempty"`
	Conditional    bool   `yaml:"conditional,omitempty"`
	ConditionField string `yaml:"conditionField,omitempty"`
	ConditionAPI   string `yaml:"conditionAPI,omitempty"`
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
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description"`
	Default     string `yaml:"default,omitempty"`
	DefaultFunc string `yaml:"defaultFunc,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
	Source      string `yaml:"source,omitempty"`
}

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
	singleCommands := make(map[string]EndpointInfo) // 单元素命令直接映射

	for opID, endpoint := range g.Config.Endpoints {
		if len(endpoint.Command) == 0 {
			continue
		}
		op := g.findOperation(opID)
		if op == nil {
			fmt.Printf("⚠ Warning: operation %s not found\n", opID)
			continue
		}

		info := EndpointInfo{
			OperationID: opID,
			Endpoint:    endpoint,
			Operation:   op,
		}

		if len(endpoint.Command) == 1 {
			// 单元素命令，直接添加到 root
			singleCommands[endpoint.Command[0]] = info
		} else {
			// 多级命令，按顶级命令分组
			topCmd := endpoint.Command[0]
			commands[topCmd] = append(commands[topCmd], info)
		}
	}

	// 生成根命令文件（包含单元素命令的注册）
	if err := g.generateRoot(commands, singleCommands, filepath.Join(outputDir, "root_gen.go")); err != nil {
		return fmt.Errorf("generate root: %w", err)
	}

	// 生成多级命令文件
	for name, endpoints := range commands {
		filename := filepath.Join(outputDir, name+"_gen.go")
		if err := g.generateCommandFile(name, endpoints, filename); err != nil {
			return fmt.Errorf("generate %s: %w", name, err)
		}
	}

	// 生成单元素命令文件
	for name, endpoint := range singleCommands {
		filename := filepath.Join(outputDir, name+"_gen.go")
		if err := g.generateSingleCommandFile(name, endpoint, filename); err != nil {
			return fmt.Errorf("generate %s: %w", name, err)
		}
	}

	fmt.Printf("\n✓ Generated %d files\n", len(commands)+len(singleCommands)+1)
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

func (g *Generator) generateRoot(commands map[string][]EndpointInfo, singleCommands map[string]EndpointInfo, filename string) error {
	var multiCmdNames []string
	for name := range commands {
		multiCmdNames = append(multiCmdNames, name)
	}
	var singleCmdNames []string
	for name := range singleCommands {
		singleCmdNames = append(singleCmdNames, name)
	}

	data := struct {
		Commands       []string
		SingleCommands []string
	}{
		Commands:       multiCmdNames,
		SingleCommands: singleCmdNames,
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
			Name:           cmdName,
			OperationID:    ep.OperationID,
			Description:    ep.Endpoint.Description,
			Use:            ep.Endpoint.Use,
			ParentCmd:      name,
			ResultType:     ep.Endpoint.ResultType,
			CustomTemplate: ep.Endpoint.CustomTemplate,
			Custom:         ep.Endpoint.Custom,
		}

		for _, f := range ep.Endpoint.Flags {
			if f.Flag == "" && f.Source != "arg" {
				continue
			}

			flagType := g.getParamType(ep.Operation, f.Param)
			if f.Type != "" {
				flagType = f.Type
			}
			varName := g.toVarName(f.Flag)
			if varName == "" {
				varName = g.toVarName(f.Param)
			}

			flag := FlagDef{
				VarName:     varName,
				ParamName:   f.Param,
				Name:        f.Flag,
				Shorthand:   f.Shorthand,
				Type:        flagType,
				CobraType:   g.toCobraType(flagType),
				Default:     g.formatDefault(f.Default, flagType),
				DefaultFunc: f.DefaultFunc,
				Description: f.Description,
				Required:    f.Required,
				IsArg:       f.Source == "arg",
			}
			sc.Flags = append(sc.Flags, flag)
			if flag.Name != "" {
				sc.FlagVars = append(sc.FlagVars, FlagVar{Name: varName, Type: flagType})
			}
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

func (g *Generator) generateSingleCommandFile(name string, ep EndpointInfo, filename string) error {
	cmdName := name
	
	sc := SubCommand{
		Name:           cmdName,
		OperationID:    ep.OperationID,
		Description:    ep.Endpoint.Description,
		Use:            ep.Endpoint.Use,
		ParentCmd:      "root",
		ResultType:     ep.Endpoint.ResultType,
		CustomTemplate: ep.Endpoint.CustomTemplate,
		Custom:         ep.Endpoint.Custom,
	}

	for _, f := range ep.Endpoint.Flags {
		if f.Flag == "" && f.Source != "arg" {
			continue
		}

		flagType := g.getParamType(ep.Operation, f.Param)
		if f.Type != "" {
			flagType = f.Type
		}
		varName := g.toVarName(f.Flag)
		if varName == "" {
			varName = g.toVarName(f.Param)
		}

		flag := FlagDef{
			VarName:     varName,
			ParamName:   f.Param,
			Name:        f.Flag,
			Shorthand:   f.Shorthand,
			Type:        flagType,
			CobraType:   g.toCobraType(flagType),
			Default:     g.formatDefault(f.Default, flagType),
			DefaultFunc: f.DefaultFunc,
			Description: f.Description,
			Required:    f.Required,
			IsArg:       f.Source == "arg",
		}
		sc.Flags = append(sc.Flags, flag)
		if flag.Name != "" {
			sc.FlagVars = append(sc.FlagVars, FlagVar{Name: varName, Type: flagType})
		}
	}

	for i, arg := range ep.Endpoint.Args {
		sc.Args = append(sc.Args, ArgDef{
			Name:  arg.Name,
			Type:  "string",
			Index: i,
		})
	}

	sc.APICall = g.buildAPICall(ep, cmdName)

	data := SingleCommandFileData{
		Package: "main",
		Name:    name,
		Command: sc,
	}

	return generateFromTemplate(filename, singleCommandTemplate, data)
}

func (g *Generator) getSubcommandName(cmd []string) string {
	if len(cmd) <= 1 {
		return ""
	}
	return strings.Join(cmd[1:], "_")
}

func (g *Generator) toVarName(s string) string {
	if s == "" {
		return ""
	}
	keywords := map[string]bool{
		"type": true, "map": true, "chan": true, "func": true,
		"var": true, "const": true, "package": true, "import": true,
	}

	parts := strings.Split(s, "-")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += strings.Title(parts[i])
	}

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
		return fmt.Sprintf("api.ListCdrIbs(ctx, startTime, endTime, %sFlags.offset, %sFlags.limit, %sFlags.phone, %sFlags.agent)", cmdName, cmdName, cmdName, cmdName)
	case "listCdrObs":
		return fmt.Sprintf("api.ListCdrObs(ctx, startTime, endTime, %sFlags.limit, %sFlags.offset, %sFlags.phone, %sFlags.agent)", cmdName, cmdName, cmdName, cmdName)
	case "listAgentStatus":
		return fmt.Sprintf("api.ListAgentStatus(ctx, %sFlags.agent)", cmdName)
	case "callout":
		return fmt.Sprintf("api.Callout(ctx, phone, %sFlags.agent, %sFlags.clid)", cmdName, cmdName)
	case "webcall":
		return fmt.Sprintf("api.Webcall(ctx, phone, %sFlags.clid, %sFlags.ivr, %sFlags.requestId)", cmdName, cmdName, cmdName)
	case "online":
		return fmt.Sprintf("api.Online(ctx, %sFlags.agent, %sFlags.queue, %sFlags.tel, %sFlags.bindType)", cmdName, cmdName, cmdName, cmdName)
	case "offline":
		return fmt.Sprintf("api.Offline(ctx, %sFlags.agent)", cmdName)
	case "pause":
		return fmt.Sprintf("api.Pause(ctx, %sFlags.agent, %sFlags.typeVal, %sFlags.reason)", cmdName, cmdName, cmdName)
	case "unpause":
		return fmt.Sprintf("api.Unpause(ctx, %sFlags.agent)", cmdName)
	case "unlink":
		return fmt.Sprintf("api.Unlink(ctx, %sFlags.agent)", cmdName)
	case "hold":
		return fmt.Sprintf("api.Hold(ctx, %sFlags.agent)", cmdName)
	case "unhold":
		return fmt.Sprintf("api.Unhold(ctx, %sFlags.agent)", cmdName)
	case "transfer":
		return fmt.Sprintf("api.Transfer(ctx, %sFlags.agent, %sFlags.typeVal, %sFlags.target)", cmdName, cmdName, cmdName)
	case "getQueueStatus":
		return fmt.Sprintf("api.GetQueueStatus(ctx, %sFlags.queue)", cmdName)
	case "listQueues":
		return fmt.Sprintf("api.ListQueues(ctx, %sFlags.offset, %sFlags.limit)", cmdName, cmdName)
	default:
		return fmt.Sprintf("api.%s(ctx)", g.toPascalCase(ep.OperationID))
	}
}

func (g *Generator) toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}

type CommandFileData struct {
	Package     string
	CommandName string
	SubCommands []SubCommand
}

type SingleCommandFileData struct {
	Package string
	Name    string
	Command SubCommand
}

type SubCommand struct {
	Name           string
	OperationID    string
	Description    string
	Use            string
	ParentCmd      string
	ResultType     string
	CustomTemplate string
	Custom         CustomConfig
	Flags          []FlagDef
	FlagVars       []FlagVar
	Args           []ArgDef
	APICall        string
}

type FlagDef struct {
	VarName     string
	ParamName   string
	Name        string
	Shorthand   string
	Type        string
	CobraType   string
	Default     string
	DefaultFunc string
	Description string
	Required    bool
	IsArg       bool
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

const rootTemplate = `// Code generated by clink-generator; DO NOT EDIT.
package main

import "github.com/spf13/cobra"

func init() {
{{range .Commands}}	rootCmd.AddCommand({{.}}Cmd)
{{end}}{{range .SingleCommands}}	rootCmd.AddCommand({{.}}Cmd)
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
	"fmt"
	"time"

	"github.com/raymondtc/clink-cli/pkg/generated"
	"github.com/raymondtc/clink-cli/pkg/renderer"
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
{{range .Flags}}{{if not .IsArg}}{{if .DefaultFunc}}
	{{$cmdName}}Cmd.Flags().{{.CobraType}}VarP(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", "{{.Shorthand}}", {{.DefaultFunc}}, "{{.Description}}"){{else if .Shorthand}}
	{{$cmdName}}Cmd.Flags().{{.CobraType}}VarP(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", "{{.Shorthand}}", {{.Default}}, "{{.Description}}"){{else}}
	{{$cmdName}}Cmd.Flags().{{.CobraType}}Var(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", {{.Default}}, "{{.Description}}"){{end}}{{if .Required}}
	{{$cmdName}}Cmd.MarkFlagRequired("{{.Name}}"){{end}}{{end}}
{{end}}}

func run{{.Name}}(cmd *cobra.Command, args []string) error {
	_ = fmt.Sprintf("")
	_ = time.Now()
	_ = context.Background
	_ = generated.CallResult{}
	_ = renderer.Table{}
	api, err := createAPI()
	if err != nil {
		return err
	}
	ctx := context.Background()
{{if .Args}}	phone := args[0]
{{end}}{{if eq .ResultType "list"}}{{if eq .OperationID "listQueues"}}
	queues, err := {{.APICall}}
	if err != nil {
		return err
	}
	return renderOutput(queues)
{{else}}
	records, total, err := {{.APICall}}
	if err != nil {
		return err
	}
	return renderList(records, total)
{{end}}{{else if eq .ResultType "simple"}}
	err = {{.APICall}}
	if err != nil {
		return err
	}
	renderer.PrintSuccess("{{.Description}}成功")
	return nil
{{else if eq .ResultType "kv"}}{{if .Custom.Conditional}}
	var result *generated.CallResult
	if {{$cmdName}}Flags.{{.Custom.ConditionField}} != "" {
		renderer.PrintSuccess(fmt.Sprintf("使用座席 %s 发起外呼...", {{$cmdName}}Flags.{{.Custom.ConditionField}}))
		result, err = api.{{.Custom.ConditionAPI}}(ctx, phone, {{$cmdName}}Flags.{{.Custom.ConditionField}}, {{$cmdName}}Flags.clid)
	} else {
		renderer.PrintSuccess("使用 WebCall 发起呼叫（无需座席）...")
		result, err = api.Webcall(ctx, phone, {{$cmdName}}Flags.clid, {{$cmdName}}Flags.ivr, nil)
	}
	if err != nil {
		return err
	}
	fmt.Println()
	renderer.PrintKV(map[string]string{
		"通话ID": deref(result.CallId),
		"状态":   deref(result.Status),
		"号码":   phone,
	})
	return nil
{{else}}
	_, err = {{.APICall}}
	if err != nil {
		return err
	}
	return nil
{{end}}{{else if eq .CustomTemplate "agentsRender"}}
	agents, err := {{.APICall}}
	if err != nil {
		return err
	}
	if outputFormat == "table" {
		fmt.Printf("座席列表 (%d 人):\n\n", len(agents))
		table := &renderer.Table{
			Headers: []string{"状态", "姓名", "座席号", "状态"},
		}
		for _, agent := range agents {
			statusIcon := "⚪"
			status := deref(agent.AgentStatus)
			switch status {
			case "空闲":
				statusIcon = "🟢"
			case "置忙":
				statusIcon = "🔴"
			case "离线":
				statusIcon = "⚪"
			}
			table.Rows = append(table.Rows, renderer.Row{
				Cells: []renderer.Cell{
					{Value: statusIcon},
					{Value: deref(agent.ClientName)},
					{Value: deref(agent.Cno)},
					{Value: status},
				},
			})
		}
		r := renderer.New(renderer.FormatTable)
		return r.Render(table)
	}
	return renderOutput(agents)
{{else if eq .CustomTemplate "queueRender"}}
	queue, err := {{.APICall}}
	if err != nil {
		return err
	}
	qname := deref(queue.Qname)
	queueID := deref(queue.Qno)
	if outputFormat == "table" {
		fmt.Printf("队列: %s (%s)\n\n", qname, queueID)
		renderer.PrintKV(map[string]string{
			"等待人数": derefInt(queue.WaitCount),
			"平均等待": fmt.Sprintf("%s 秒", derefInt(queue.QueueUpWaitTime)),
			"在线座席": derefInt(queue.OnlineAgentCount),
			"忙碌座席": derefInt(queue.BusyAgentCount),
		})
		return nil
	}
	return renderOutput(queue)
{{else}}
	_, err = {{.APICall}}
	return err
{{end}}}
{{end}}
`

const singleCommandTemplate = `// Code generated by clink-generator; DO NOT EDIT.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/raymondtc/clink-cli/pkg/generated"
	"github.com/raymondtc/clink-cli/pkg/renderer"
	"github.com/spf13/cobra"
)

{{$cmdName := .Name}}

var {{.Name}}Flags struct {
{{range .Command.FlagVars}}	{{.Name}} {{.Type}}
{{end}}}

var {{.Name}}Cmd = &cobra.Command{
	Use:   "{{if .Command.Use}}{{.Command.Use}}{{else}}{{.Name}}{{end}}",
	Short: "{{.Command.Description}}",
{{if .Command.Args}}	Args:  cobra.ExactArgs({{len .Command.Args}}),
{{end}}	RunE:  run{{.Name}},
}

func init() {
{{range .Command.Flags}}{{if not .IsArg}}{{if .DefaultFunc}}
	{{$cmdName}}Cmd.Flags().{{.CobraType}}VarP(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", "{{.Shorthand}}", {{.DefaultFunc}}, "{{.Description}}"){{else if .Shorthand}}
	{{$cmdName}}Cmd.Flags().{{.CobraType}}VarP(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", "{{.Shorthand}}", {{.Default}}, "{{.Description}}"){{else}}
	{{$cmdName}}Cmd.Flags().{{.CobraType}}Var(&{{$cmdName}}Flags.{{.VarName}}, "{{.Name}}", {{.Default}}, "{{.Description}}"){{end}}{{if .Required}}
	{{$cmdName}}Cmd.MarkFlagRequired("{{.Name}}"){{end}}{{end}}
{{end}}}

func run{{.Name}}(cmd *cobra.Command, args []string) error {
	_ = fmt.Sprintf("")
	_ = time.Now()
	_ = context.Background
	_ = generated.CallResult{}
	_ = renderer.Table{}
	api, err := createAPI()
	if err != nil {
		return err
	}
	ctx := context.Background()
{{if .Command.Args}}	phone := args[0]
{{end}}{{if eq .Command.ResultType "list"}}{{if eq .Command.OperationID "listQueues"}}
	queues, err := {{.Command.APICall}}
	if err != nil {
		return err
	}
	return renderOutput(queues)
{{else}}
	records, total, err := {{.Command.APICall}}
	if err != nil {
		return err
	}
	return renderList(records, total)
{{end}}{{else if eq .Command.ResultType "simple"}}
	err = {{.Command.APICall}}
	if err != nil {
		return err
	}
	renderer.PrintSuccess("{{.Command.Description}}成功")
	return nil
{{else if eq .Command.ResultType "kv"}}{{if .Command.Custom.Conditional}}
	var result *generated.CallResult
	if {{$cmdName}}Flags.{{.Command.Custom.ConditionField}} != "" {
		renderer.PrintSuccess(fmt.Sprintf("使用座席 %s 发起外呼...", {{$cmdName}}Flags.{{.Command.Custom.ConditionField}}))
		result, err = api.{{.Command.Custom.ConditionAPI}}(ctx, phone, {{$cmdName}}Flags.{{.Command.Custom.ConditionField}}, {{$cmdName}}Flags.clid)
	} else {
		renderer.PrintSuccess("使用 WebCall 发起呼叫（无需座席）...")
		result, err = api.Webcall(ctx, phone, {{$cmdName}}Flags.clid, {{$cmdName}}Flags.ivr, nil)
	}
	if err != nil {
		return err
	}
	fmt.Println()
	renderer.PrintKV(map[string]string{
		"通话ID": deref(result.CallId),
		"状态":   deref(result.Status),
		"号码":   phone,
	})
	return nil
{{else}}
	_, err = {{.Command.APICall}}
	if err != nil {
		return err
	}
	return nil
{{end}}{{else if eq .Command.CustomTemplate "agentsRender"}}
	agents, err := {{.Command.APICall}}
	if err != nil {
		return err
	}
	if outputFormat == "table" {
		fmt.Printf("座席列表 (%d 人):\n\n", len(agents))
		table := &renderer.Table{
			Headers: []string{"状态", "姓名", "座席号", "状态"},
		}
		for _, agent := range agents {
			statusIcon := "⚪"
			status := deref(agent.AgentStatus)
			switch status {
			case "空闲":
				statusIcon = "🟢"
			case "置忙":
				statusIcon = "🔴"
			case "离线":
				statusIcon = "⚪"
			}
			table.Rows = append(table.Rows, renderer.Row{
				Cells: []renderer.Cell{
					{Value: statusIcon},
					{Value: deref(agent.ClientName)},
					{Value: deref(agent.Cno)},
					{Value: status},
				},
			})
		}
		r := renderer.New(renderer.FormatTable)
		return r.Render(table)
	}
	return renderOutput(agents)
{{else if eq .Command.CustomTemplate "queueRender"}}
	queue, err := {{.Command.APICall}}
	if err != nil {
		return err
	}
	qname := deref(queue.Qname)
	queueID := deref(queue.Qno)
	if outputFormat == "table" {
		fmt.Printf("队列: %s (%s)\n\n", qname, queueID)
		renderer.PrintKV(map[string]string{
			"等待人数": derefInt(queue.WaitCount),
			"平均等待": fmt.Sprintf("%s 秒", derefInt(queue.QueueUpWaitTime)),
			"在线座席": derefInt(queue.OnlineAgentCount),
			"忙碌座席": derefInt(queue.BusyAgentCount),
		})
		return nil
	}
	return renderOutput(queue)
{{else}}
	_, err = {{.Command.APICall}}
	return err
{{end}}}
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
