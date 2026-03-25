// generate-cli-v3.go - 基于 cli.yaml 配置生成 CLI 命令
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// CLIConfig 对应 cli.yaml 的结构
type CLIConfig struct {
	Version  string              `yaml:"version"`
	Meta     Meta                `yaml:"meta"`
	Defaults Defaults            `yaml:"defaults"`
	Groups   map[string]Group    `yaml:"groups"`
	Commands map[string]Command  `yaml:"commands"`
}

type Meta struct {
	Description    string `yaml:"description"`
	Source         string `yaml:"source"`
	OpenAPIVersion string `yaml:"openapiVersion"`
}

type Defaults struct {
	Output     OutputDefaults     `yaml:"output"`
	Pagination PaginationDefaults `yaml:"pagination"`
}

type OutputDefaults struct {
	Format    string `yaml:"format"`
	TimeFormat string `yaml:"timeFormat"`
}

type PaginationDefaults struct {
	Limit    int `yaml:"limit"`
	MaxLimit int `yaml:"maxLimit"`
}

type Group struct {
	Title       string `yaml:"title"`
	Emoji       string `yaml:"emoji"`
	Description string `yaml:"description"`
}

type Command struct {
	Path        []string           `yaml:"path"`
	Aliases     []string           `yaml:"aliases"`
	Group       string             `yaml:"group"`
	Extends     string             `yaml:"extends"`
	Dangerous   bool               `yaml:"dangerous"`
	Confirm     string             `yaml:"confirm"`
	Arguments   []Argument         `yaml:"arguments"`
	Parameters  map[string]Param   `yaml:"parameters"`
	Output      CommandOutput      `yaml:"output"`
}

type Argument struct {
	Name     string `yaml:"name"`
	From     string `yaml:"from"`
	Required bool   `yaml:"required"`
	Validate string `yaml:"validate"`
}

type Param struct {
	Flag       string      `yaml:"flag"`
	Shorthand  string      `yaml:"shorthand"`
	Default    interface{} `yaml:"default"`
	Required   *bool       `yaml:"required"`
	Hidden     bool        `yaml:"hidden"`
	Positional bool        `yaml:"positional"`
	ArgName    string      `yaml:"argName"`
	Transform  Transform   `yaml:"transform"`
	Auto       string      `yaml:"auto"`
	Enum       map[string]string `yaml:"enum"`
}

type Transform struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
	Raw    string // 字符串格式时存储
}

func (t *Transform) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		t.Raw = value.Value
		return nil
	}
	type transformStruct Transform
	return value.Decode((*transformStruct)(t))
}

type CommandOutput struct {
	Format  string   `yaml:"format"`
	Columns []Column `yaml:"columns"`
	Success string   `yaml:"success"`
}

type Column struct {
	Key    string `yaml:"key"`
	Header string `yaml:"header"`
	Width  int    `yaml:"width"`
	Mask   bool   `yaml:"mask"`
	Type   string `yaml:"type"`
	Format string `yaml:"format"`
}

// 模板数据
type CommandTemplateData struct {
	Package     string
	CommandName string
	Use         string
	Short       string
	Aliases     []string
	Flags       []FlagInfo
	Args        []ArgInfo
	APIFunc     string
	Dangerous   bool
	Confirm     string
}

type FlagInfo struct {
	Name      string
	FlagName  string
	Shorthand string
	Type      string
	Default   string
	Required  bool
	Help      string
}

type ArgInfo struct {
	Name     string
	Required bool
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <cli.yaml>\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	
	fmt.Println("=== CLI Command Generator v3 ===")
	fmt.Printf("Config: %s\n", configPath)
	fmt.Println()

	// 读取配置
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d commands\n", len(config.Commands))
	fmt.Println()

	// 创建输出目录
	outputDir := "cmd/clink"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		os.Exit(1)
	}

	// 处理继承
	config = resolveInheritance(config)

	// 按分组组织命令
	groups := organizeByGroup(config)

	// 生成根命令注册文件
	if err := generateRootCommands(outputDir, groups, config.Groups); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating root commands: %v\n", err)
		os.Exit(1)
	}

	// 为每个分组生成命令文件
	for groupName, commands := range groups {
		if err := generateGroupCommands(outputDir, groupName, commands, config.Groups); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s commands: %v\n", groupName, err)
			os.Exit(1)
		}
	}

	fmt.Println()
	fmt.Println("✓ CLI command generation complete!")
	fmt.Printf("  Generated files in: %s/\n", outputDir)
}

func loadConfig(path string) (*CLIConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config CLIConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func resolveInheritance(config *CLIConfig) *CLIConfig {
	for name, cmd := range config.Commands {
		if cmd.Extends != "" {
			baseCmd, exists := config.Commands[cmd.Extends]
			if exists {
				// 合并基础命令的参数
				mergedParams := make(map[string]Param)
				for k, v := range baseCmd.Parameters {
					mergedParams[k] = v
				}
				for k, v := range cmd.Parameters {
					mergedParams[k] = v
				}
				cmd.Parameters = mergedParams
				config.Commands[name] = cmd
			}
		}
	}
	return config
}

func organizeByGroup(config *CLIConfig) map[string]map[string]Command {
	groups := make(map[string]map[string]Command)
	
	for name, cmd := range config.Commands {
		group := cmd.Group
		if group == "" {
			group = "default"
		}
		if groups[group] == nil {
			groups[group] = make(map[string]Command)
		}
		groups[group][name] = cmd
	}
	
	return groups
}

func generateRootCommands(outputDir string, groups map[string]map[string]Command, groupDefs map[string]Group) error {
	tmplStr := `// Code generated by generate-cli-v3; DO NOT EDIT.
package main

import "github.com/spf13/cobra"

{{ range $groupName, $commands := .Groups }}
var {{ $groupName }}Cmd = &cobra.Command{
	Use:   "{{ $groupName }}",
	Short: "{{ $groupName }} related commands",
}
{{ end }}

func init() {
{{- range $groupName, $commands := .Groups }}
	rootCmd.AddCommand({{ $groupName }}Cmd)
{{- end }}
}
`

	data := struct {
		Groups map[string]map[string]Command
	}{
		Groups: groups,
	}

	tmpl, err := template.New("root").Parse(tmplStr)
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(outputDir, "root_gen.go"))
	if err != nil {
		return err
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return err
	}

	fmt.Printf("  Generated: root_gen.go\n")
	return nil
}

func generateGroupCommands(outputDir, groupName string, commands map[string]Command, groupDefs map[string]Group) error {
	fileName := fmt.Sprintf("%s_gen.go", groupName)
	
	var builder strings.Builder
	builder.WriteString("// Code generated by generate-cli-v3; DO NOT EDIT.\n")
	builder.WriteString("package main\n\n")
	builder.WriteString("import (\n")
	builder.WriteString("\t\"context\"\n")
	builder.WriteString("\t\"fmt\"\n\n")
	builder.WriteString("\t\"github.com/spf13/cobra\"\n")
	builder.WriteString(")\n\n")

	for cmdName, cmd := range commands {
		if err := generateSingleCommand(&builder, cmdName, cmd); err != nil {
			return fmt.Errorf("generating %s: %w", cmdName, err)
		}
	}

	file, err := os.Create(filepath.Join(outputDir, fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(builder.String()); err != nil {
		return err
	}

	fmt.Printf("  Generated: %s\n", fileName)
	return nil
}

func generateSingleCommand(builder *strings.Builder, name string, cmd Command) error {
	funcName := goFuncName(name)
	cmdVar := strings.ToLower(funcName) + "Cmd"
	flagsVar := strings.ToLower(funcName) + "Flags"

	// 生成 Flags 结构体
	builder.WriteString(fmt.Sprintf("var %s struct {\n", flagsVar))
	for paramName, param := range cmd.Parameters {
		if param.Hidden {
			continue
		}
		fieldName := goFieldName(paramName)
		fieldType := "string"
		if param.Default != nil {
			switch param.Default.(type) {
			case int:
				fieldType = "int"
			case bool:
				fieldType = "bool"
			}
		}
		builder.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, fieldType))
	}
	builder.WriteString("}\n\n")

	// 生成命令定义
	use := cmd.Path[len(cmd.Path)-1]
	if len(cmd.Arguments) > 0 {
		for _, arg := range cmd.Arguments {
			use += fmt.Sprintf(" \u003c%s\u003e", arg.Name)
		}
	}
	
	builder.WriteString(fmt.Sprintf("var %s = \u0026cobra.Command{\n", cmdVar))
	builder.WriteString(fmt.Sprintf("\tUse:   \"%s\",\n", use))
	builder.WriteString(fmt.Sprintf("\tShort: \"%s\",\n", getDescription(name, cmd)))
	
	if len(cmd.Aliases) > 0 {
		builder.WriteString(fmt.Sprintf("\tAliases: []string{%s},\n", formatAliases(cmd.Aliases)))
	}
	
	if len(cmd.Arguments) > 0 {
		builder.WriteString(fmt.Sprintf("\tArgs:  cobra.ExactArgs(%d),\n", len(cmd.Arguments)))
	}
	
	builder.WriteString(fmt.Sprintf("\tRunE:  run%s,\n", funcName))
	builder.WriteString("}\n\n")

	// 生成 init 函数
	parentCmd := cmd.Group + "Cmd"
	if parentCmd == "Cmd" {
		parentCmd = cmd.Path[0] + "Cmd"
	}
	builder.WriteString("func init() {\n")
	builder.WriteString(fmt.Sprintf("\t%s.AddCommand(%s)\n\n", parentCmd, cmdVar))
	
	// 生成 flag 绑定
	for paramName, param := range cmd.Parameters {
		if param.Hidden {
			continue
		}
		
		flagName := param.Flag
		if flagName == "" {
			flagName = strings.ToLower(paramName)
		}
		
		shorthand := param.Shorthand
		fieldName := goFieldName(paramName)
		
		// 判断类型
		isInt := false
		if param.Default != nil {
			switch param.Default.(type) {
			case int:
				isInt = true
			}
		}
		
		var flagDef string
		if isInt {
			defaultVal := "0"
			if param.Default != nil {
				defaultVal = fmt.Sprintf("%v", param.Default)
			}
			if shorthand != "" {
				flagDef = fmt.Sprintf("\t%s.Flags().IntVarP(\u0026%s.%s, \"%s\", \"%s\", %s, \"%s\")\n",
					cmdVar, flagsVar, fieldName, flagName, shorthand, defaultVal, getParamDesc(paramName))
			} else {
				flagDef = fmt.Sprintf("\t%s.Flags().IntVar(\u0026%s.%s, \"%s\", %s, \"%s\")\n",
					cmdVar, flagsVar, fieldName, flagName, defaultVal, getParamDesc(paramName))
			}
		} else {
			defaultVal := "\"\""
			if param.Default != nil {
				if s, ok := param.Default.(string); ok {
					defaultVal = fmt.Sprintf("\"%s\"", s)
				} else {
					defaultVal = fmt.Sprintf("\"%v\"", param.Default)
				}
			}
			if shorthand != "" {
				flagDef = fmt.Sprintf("\t%s.Flags().StringVarP(\u0026%s.%s, \"%s\", \"%s\", %s, \"%s\")\n",
					cmdVar, flagsVar, fieldName, flagName, shorthand, defaultVal, getParamDesc(paramName))
			} else {
				flagDef = fmt.Sprintf("\t%s.Flags().StringVar(\u0026%s.%s, \"%s\", %s, \"%s\")\n",
					cmdVar, flagsVar, fieldName, flagName, defaultVal, getParamDesc(paramName))
			}
		}
		builder.WriteString(flagDef)
		
		if param.Required != nil && *param.Required {
			builder.WriteString(fmt.Sprintf("\t%s.MarkFlagRequired(\"%s\")\n", cmdVar, flagName))
		}
	}
	
	builder.WriteString("}\n\n")

	// 生成 RunE 函数
	builder.WriteString(fmt.Sprintf("func run%s(cmd *cobra.Command, args []string) error {\n", funcName))
	builder.WriteString("\t_ = fmt.Sprintf(\"\")\n")  // 使用 fmt 避免导入未使用错误
	
	// 解析位置参数
	for i, arg := range cmd.Arguments {
		varName := goFieldName(arg.Name)
		builder.WriteString(fmt.Sprintf("\t%s := args[%d]\n", varName, i))
		builder.WriteString(fmt.Sprintf("\t_ = %s\n", varName))
	}
	
	builder.WriteString("\tapi, err := createAPI()\n")
	builder.WriteString("\tif err != nil {\n")
	builder.WriteString("\t\treturn err\n")
	builder.WriteString("\t}\n")
	builder.WriteString("\t_ = api\n")
	builder.WriteString("\tctx := context.Background()\n")
	builder.WriteString("\t_ = ctx\n\n")
	
	// 危险操作确认
	if cmd.Dangerous && cmd.Confirm != "" {
		builder.WriteString(fmt.Sprintf("\tfmt.Print(\"⚠️  %s [y/N]: \")\n", cmd.Confirm))
		builder.WriteString("\tvar confirm string\n")
		builder.WriteString("\tfmt.Scanln(&confirm)\n")
		builder.WriteString("\tif confirm != \"y\" && confirm != \"Y\" {\n")
		builder.WriteString("\t\tfmt.Println(\"已取消\")\n")
		builder.WriteString("\t\treturn nil\n")
		builder.WriteString("\t}\n\n")
	}
	
	// 生成参数准备代码
	generateAPICallCode(builder, name, cmd, flagsVar)
	
	builder.WriteString("\treturn nil\n")
	builder.WriteString("}\n\n")

	return nil
}

// generateAPICallCode 生成 API 调用代码
func generateAPICallCode(builder *strings.Builder, cmdName string, cmd Command, flagsVar string) {
	// 检查 API 是否已实现
	implementedAPIs := map[string]bool{
		"ListCdrIbs":            true,
		"ListCdrObs":            true,
		"AgentStatus":           true,
		"QueueStatus":           true,
		"ListQueues":            true,
		"DescribeRecordFileUrl": true,
		"Unpause":               true,
		"Online":                true,
		"Offline":               true,
		"Pause":                 true,
		"Unlink":                true,
		"Hold":                  true,
		"Unhold":                true,
		"Transfer":              true,
		"Webcall":               true,
		"Callout":               true,
		"Consult":               true,
		"ConsultTransfer":       true,
		"Mute":                  true,
		"Unmute":                true,
		"CalloutCancel":         true,
	}

	if !implementedAPIs[cmdName] {
		// 未实现的 API 只生成骨架代码
		builder.WriteString("\t// TODO: API not yet implemented: " + cmdName + "\n")
		builder.WriteString("\tfmt.Println(\"API not yet implemented: " + cmdName + "\")\n")
		builder.WriteString("\treturn nil\n")
		return
	}

	switch cmdName {
	case "ListCdrIbs":
		builder.WriteString("\t// 参数转换\n")
		builder.WriteString("\tstartTime, err := parseRelativeTime(" + flagsVar + ".StartTime)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn fmt.Errorf(\"invalid start time: %w\", err)\n")
		builder.WriteString("\t}\n")
		builder.WriteString("\tendTime, err := parseRelativeTime(" + flagsVar + ".EndTime)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn fmt.Errorf(\"invalid end time: %w\", err)\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\trecords, total, err := api.ListCdrIbs(ctx, startTime, endTime, 0, 10, " + flagsVar + ".CustomerNumber, " + flagsVar + ".Cno)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn err\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\treturn renderList(records, total)\n")

	case "ListCdrObs":
		builder.WriteString("\tstartTime, err := parseRelativeTime(" + flagsVar + ".StartTime)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn fmt.Errorf(\"invalid start time: %w\", err)\n")
		builder.WriteString("\t}\n")
		builder.WriteString("\tendTime, err := parseRelativeTime(" + flagsVar + ".EndTime)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn fmt.Errorf(\"invalid end time: %w\", err)\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\trecords, total, err := api.ListCdrObs(ctx, startTime, endTime, 10, 0, " + flagsVar + ".CustomerNumber, " + flagsVar + ".Cno)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn err\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\treturn renderList(records, total)\n")

	case "AgentStatus":
		builder.WriteString("\tagents, total, err := api.ListAgentStatus(ctx, " + flagsVar + ".Cno)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn err\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\treturn renderList(agents, total)\n")

	case "QueueStatus":
		builder.WriteString("\tqueues, total, err := api.GetQueueStatus(ctx, " + flagsVar + ".Qnos)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn err\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\treturn renderList(queues, total)\n")

	case "ListQueues":
		builder.WriteString("\tqueues, total, err := api.ListQueues(ctx, 0, 10)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn err\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\treturn renderList(queues, total)\n")

	case "DescribeRecordFileUrl":
		builder.WriteString("\t// 获取 callId 从位置参数\n")
		builder.WriteString("\tcallId := \"\"\n")
		builder.WriteString("\tif len(args) > 0 {\n")
		builder.WriteString("\t\tcallId = args[0]\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\tresult, err := api.DescribeRecordFileUrl(ctx, callId, 0, 3600, 1)\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\treturn err\n")
		builder.WriteString("\t}\n\n")
		builder.WriteString("\treturn renderOutput(result)\n")

	default:
		// 其他已实现的命令（写操作）
		builder.WriteString("\t// TODO: Implement API call for " + cmdName + "\n")
		builder.WriteString("\tfmt.Println(\"TODO: Implement API call\")\n")
		builder.WriteString("\treturn nil\n")
	}
}

func goFuncName(name string) string {
	// 将 operationId 转换为函数名
	return name
}

func goFieldName(name string) string {
	// 首字母大写，移除连字符和下划线
	if len(name) == 0 {
		return name
	}
	// 替换连字符和下划线为空格，然后转换为驼峰命名
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, "")
}

func formatAliases(aliases []string) string {
	var parts []string
	for _, a := range aliases {
		parts = append(parts, fmt.Sprintf("\"%s\"", a))
	}
	return strings.Join(parts, ", ")
}

func getDescription(name string, cmd Command) string {
	return name
}

func getParamDesc(name string) string {
	descs := map[string]string{
		"startTime": "开始时间",
		"endTime":   "结束时间",
		"cno":       "座席号",
		"qno":       "队列号",
		"customerNumber": "客户号码",
	}
	if d, ok := descs[name]; ok {
		return d
	}
	return name
}
