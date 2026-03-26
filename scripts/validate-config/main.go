// validate-config.go - 验证 CLI 配置与 OpenAPI 的兼容性
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// 使用 extract-openapi.go 中定义的 OpenAPISpec 类型
type ValidateOpenAPISpec struct {
	Paths map[string]ValidatePathItem `json:"paths"`
}

type ValidatePathItem map[string]*ValidateOperation

type ValidateOperation struct {
	OperationID string              `json:"operationId"`
	Parameters  []ValidateParameter `json:"parameters"`
}

type ValidateParameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

type CLIConfig struct {
	Version  string                 `yaml:"version"`
	Groups   map[string]Group       `yaml:"groups"`
	Commands map[string]Command     `yaml:"commands"`
}

type Group struct {
	Title       string `yaml:"title"`
	Emoji       string `yaml:"emoji"`
	Description string `yaml:"description"`
}

type Command struct {
	Path       []string              `yaml:"path"`
	Aliases    []string              `yaml:"aliases"`
	Group      string                `yaml:"group"`
	Extends    string                `yaml:"extends"`
	Dangerous  bool                  `yaml:"dangerous"`
	Confirm    string                `yaml:"confirm"`
	Arguments  []Argument            `yaml:"arguments"`
	Parameters map[string]Param      `yaml:"parameters"`
	Output     Output                `yaml:"output"`
}

type Argument struct {
	Name     string `yaml:"name"`
	From     string `yaml:"from"`
	Required bool   `yaml:"required"`
	Validate string `yaml:"validate"`
}

type Param struct {
	Flag       string          `yaml:"flag"`
	Shorthand  string          `yaml:"shorthand"`
	Default    interface{}     `yaml:"default"`
	Required   *bool           `yaml:"required"`
	Hidden     bool            `yaml:"hidden"`
	Positional bool            `yaml:"positional"`
	ArgName    string          `yaml:"argName"`
	Transform  FlexibleTransform `yaml:"transform"`
	Auto       string          `yaml:"auto"`
}

// FlexibleTransform 支持字符串或对象格式的 transform
type FlexibleTransform struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
	Raw    string // 当 transform 是字符串时存储在这里
}

func (ft *FlexibleTransform) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		// 字符串格式: transform: "split:,"
		ft.Raw = value.Value
		return nil
	}
	// 对象格式: transform: {input: ..., output: ...}
	type transformStruct FlexibleTransform
	return value.Decode((*transformStruct)(ft))
}

type Output struct {
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

var (
	openAPIPath string
	configPath  string
	verbose     bool
)

func main() {
	flag.StringVar(&openAPIPath, "openapi", "./openapi/openapi.json", "OpenAPI spec path")
	flag.StringVar(&configPath, "config", "./config/cli.yaml", "CLI config path")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	fmt.Println("=== Clink CLI Config Validator ===")
	fmt.Printf("OpenAPI: %s\n", openAPIPath)
	fmt.Printf("Config:  %s\n", configPath)
	fmt.Println()

	// 加载 OpenAPI
	openAPI, err := loadOpenAPI(openAPIPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load OpenAPI: %v\n", err)
		os.Exit(1)
	}

	// 提取所有 operationId
	operationIds := extractOperationIds(openAPI)
	fmt.Printf("✓ Loaded OpenAPI: %d operations\n", len(operationIds))

	// 加载 CLI 配置
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Loaded CLI config: %d commands\n", len(config.Commands))
	fmt.Println()

	// 执行验证
	issues := validate(config, operationIds)

	// 打印结果
	fmt.Println("=== Validation Results ===")
	if len(issues) == 0 {
		fmt.Println("✅ All checks passed!")
	} else {
		for _, issue := range issues {
			fmt.Println(issue)
		}
		fmt.Printf("\n⚠️  Found %d issue(s)\n", len(issues))
		os.Exit(1)
	}
}

func loadOpenAPI(path string) (*ValidateOpenAPISpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec ValidateOpenAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func extractOperationIds(spec *ValidateOpenAPISpec) map[string]bool {
	ids := make(map[string]bool)
	for _, pathItem := range spec.Paths {
		for _, op := range pathItem {
			if op != nil && op.OperationID != "" {
				ids[op.OperationID] = true
			}
		}
	}
	return ids
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

func validate(config *CLIConfig, operationIds map[string]bool) []string {
	var issues []string

	// 1. 检查每个 command 的 operationId 是否存在于 OpenAPI
	for opId, cmd := range config.Commands {
		if !operationIds[opId] {
			issues = append(issues, fmt.Sprintf("❌ Unknown operationId: %s (command: %s)", 
				opId, strings.Join(cmd.Path, " ")))
		} else if verbose {
			fmt.Printf("✓ %s -> %s\n", opId, strings.Join(cmd.Path, " "))
		}

		// 2. 检查 extends 引用
		if cmd.Extends != "" {
			if _, exists := config.Commands[cmd.Extends]; !exists {
				issues = append(issues, fmt.Sprintf("❌ %s extends unknown command: %s", 
					opId, cmd.Extends))
			}
		}

		// 3. 检查 group 引用
		if cmd.Group != "" {
			if _, exists := config.Groups[cmd.Group]; !exists {
				issues = append(issues, fmt.Sprintf("❌ %s references unknown group: %s", 
					opId, cmd.Group))
			}
		}

		// 4. 检查参数映射
		for paramName, param := range cmd.Parameters {
			if param.Positional && param.ArgName == "" {
				issues = append(issues, fmt.Sprintf("⚠️  %s parameter '%s' is positional but has no argName", 
					opId, paramName))
			}
		}

		// 5. 检查 arguments 引用
		for _, arg := range cmd.Arguments {
			if arg.From == "" {
				issues = append(issues, fmt.Sprintf("❌ %s argument '%s' has no 'from' mapping", 
					opId, arg.Name))
			}
			// 检查对应的 parameter 是否存在
			if _, exists := cmd.Parameters[arg.From]; !exists {
				issues = append(issues, fmt.Sprintf("⚠️  %s argument '%s' maps to unknown parameter: %s", 
					opId, arg.Name, arg.From))
			}
		}
	}

	// 6. 检查是否有重复的 command path
	paths := make(map[string]string)
	for opId, cmd := range config.Commands {
		pathKey := strings.Join(cmd.Path, "/")
		if existing, dup := paths[pathKey]; dup {
			issues = append(issues, fmt.Sprintf("❌ Duplicate command path: %s (used by %s and %s)", 
				pathKey, existing, opId))
		} else {
			paths[pathKey] = opId
		}
	}

	// 7. 检查 aliases 冲突
	allAliases := make(map[string]string)
	for opId, cmd := range config.Commands {
		for _, alias := range cmd.Aliases {
			if existing, dup := allAliases[alias]; dup {
				issues = append(issues, fmt.Sprintf("❌ Duplicate alias '%s': used by %s and %s", 
					alias, existing, opId))
			} else {
				allAliases[alias] = opId
			}
		}
	}

	return issues
}
