// extract-openapi.go - 从 Clink SDK 提取 OpenAPI 3.0 规范
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]PathItem    `json:"paths"`
	Components Components             `json:"components"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Server struct {
	URL string `json:"url"`
}

type PathItem map[string]*Operation

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	OperationID string              `json:"operationId"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Description string   `json:"description,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Schema      *Schema  `json:"schema,omitempty"`
}

type Schema struct {
	Type       string            `json:"type,omitempty"`
	Format     string            `json:"format,omitempty"`
	Items      *Schema           `json:"items,omitempty"`
	Ref        string            `json:"$ref,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
}

type RequestBody struct {
	Required bool                      `json:"required,omitempty"`
	Content  map[string]MediaType      `json:"content"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Response struct {
	Description string                 `json:"description"`
	Content     map[string]MediaType   `json:"content,omitempty"`
}

type Components struct {
	Schemas map[string]*Schema `json:"schemas"`
}

var (
	sdkPath string
	output  string
)

func main() {
	flag.StringVar(&sdkPath, "sdk", "./sdk/clink-sdk/clink-serversdk/src/main/java/com/tinet/clink", "SDK source path")
	flag.StringVar(&output, "out", "./openapi/openapi.json", "Output file path")
	flag.Parse()

	fmt.Println("=== Clink SDK OpenAPI Extractor ===")
	fmt.Printf("SDK Path: %s\n", sdkPath)
	fmt.Printf("Output: %s\n", output)
	fmt.Println()

	// 1. 解析 PathEnum
	pathMap := parsePathEnum()
	fmt.Printf("✓ Parsed PathEnum: %d APIs\n", len(pathMap))

	// 2. 解析所有 Request 类
	apis := parseAllRequests(pathMap)
	fmt.Printf("✓ Parsed Requests: %d APIs\n", len(apis))

	// 3. 解析 Response 和 Model（简化版：只记录引用）
	schemas := parseSchemas()
	fmt.Printf("✓ Parsed Schemas: %d models\n", len(schemas))

	// 4. 构建 OpenAPI 规范
	spec := buildOpenAPISpec(apis, schemas)

	// 5. 写入文件
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ OpenAPI spec generated: %s\n", output)
	fmt.Printf("  - Paths: %d\n", len(spec.Paths))
	fmt.Printf("  - Schemas: %d\n", len(spec.Components.Schemas))
}

func parsePathEnum() map[string]string {
	content, err := os.ReadFile(filepath.Join(sdkPath, "cc", "PathEnum.java"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading PathEnum: %v\n", err)
		return nil
	}

	// 提取枚举: Name("path")
	re := regexp.MustCompile(`(?m)^\s+(\w+)\("([^"]+)"\)`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	pathMap := make(map[string]string)
	for _, m := range matches {
		if len(m) == 3 {
			pathMap[m[1]] = m[2]
		}
	}
	return pathMap
}

func parseAllRequests(pathMap map[string]string) []APIInfo {
	var apis []APIInfo

	// 遍历所有 Request 文件
	requestDir := filepath.Join(sdkPath, "cc", "request")
	entries, err := os.ReadDir(requestDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request dir: %v\n", err)
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(requestDir, entry.Name())
		files, _ := os.ReadDir(subDir)
		for _, f := range files {
			if strings.HasSuffix(f.Name(), "Request.java") {
				api := parseRequestFile(filepath.Join(subDir, f.Name()), pathMap)
				if api != nil {
					apis = append(apis, *api)
				}
			}
		}
	}

	return apis
}

type APIInfo struct {
	Name        string
	Path        string
	Method      string
	Summary     string
	OperationID string
	Parameters  []Parameter
	BodyFields  []Parameter
	ResponseRef string
}

func parseRequestFile(path string, pathMap map[string]string) *APIInfo {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	s := string(content)

	// 提取类名
	classRe := regexp.MustCompile(`public\s+class\s+(\w+)Request`)
	classMatch := classRe.FindStringSubmatch(s)
	if len(classMatch) < 2 {
		return nil
	}
	className := classMatch[1]

	// 在 PathEnum 中查找对应路径
	path, ok := pathMap[className]
	if !ok {
		return nil
	}

	api := &APIInfo{
		Name:        className,
		Path:        "/" + path,
		OperationID: camelToSnake(className),
	}

	// 提取 HTTP Method
	if strings.Contains(s, "HttpMethodType.POST") {
		api.Method = "post"
	} else {
		api.Method = "get"
	}

	// 提取 Summary
	api.Summary = extractClassSummary(s)

	// 提取参数
	api.Parameters, api.BodyFields = extractRequestParameters(s, api.Method)

	// 提取 Response 引用
	respRe := regexp.MustCompile(`getResponseClass\(\)\s*\{\s*return\s+(\w+)\.class`)
	respMatch := respRe.FindStringSubmatch(s)
	if len(respMatch) > 1 {
		api.ResponseRef = respMatch[1]
	}

	return api
}

func extractClassSummary(content string) string {
	// 提取类级别 Javadoc
	re := regexp.MustCompile(`(?s)/\*\*\s*\n\s*\*\s*([^\n@]+)`)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		summary := strings.TrimSpace(match[1])
		summary = strings.TrimSuffix(summary, "请求")
		return summary
	}
	return ""
}

func extractRequestParameters(content, method string) ([]Parameter, []Parameter) {
	var queryParams, bodyParams []Parameter

	// 匹配字段块: /** doc */ private Type name;
	fieldRe := regexp.MustCompile(`(?s)/\*\*\s*(.*?)\*/\s*private\s+(\w+)\s+(\w+);`)
	matches := fieldRe.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		if len(m) != 4 {
			continue
		}
		docBlock := m[1]
		javaType := m[2]
		fieldName := m[3]

		if fieldName == "serialVersionUID" {
			continue
		}

		param := Parameter{
			Name:        fieldName,
			Description: cleanDescription(docBlock, fieldName),
			Schema:      javaTypeToSchema(javaType),
		}

		// 判断参数位置：看 setter 中调用 putQueryParameter 还是 putBodyParameter
		if isQueryParameter(content, fieldName) {
			param.In = "query"
			queryParams = append(queryParams, param)
		} else if method == "post" {
			param.In = "body"
			bodyParams = append(bodyParams, param)
		}
	}

	return queryParams, bodyParams
}

func isQueryParameter(content, fieldName string) bool {
	// 查找 setter 方法，看是调用 putQueryParameter 还是 putBodyParameter
	setterRe := regexp.MustCompile(`set` + capitalize(fieldName) + `\([^)]+\)\s*\{([^}]+)\}`)
	match := setterRe.FindStringSubmatch(content)
	if len(match) > 1 {
		return strings.Contains(match[1], "putQueryParameter")
	}
	// GET 请求默认都是 query
	return strings.Contains(content, "HttpMethodType.GET")
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func cleanDescription(docBlock, fieldName string) string {
	// 清理 docBlock，移除类注释残留
	docBlock = regexp.MustCompile(`(?s)^.*?@author[^\n]*\n`).ReplaceAllString(docBlock, "")
	docBlock = regexp.MustCompile(`(?s)^.*?@date[^\n]*\n`).ReplaceAllString(docBlock, "")
	docBlock = regexp.MustCompile(`请求对象?\s*/\s*\*`).ReplaceAllString(docBlock, "")
	docBlock = regexp.MustCompile(`Class for:\s*`).ReplaceAllString(docBlock, "")
	docBlock = regexp.MustCompile(`^\s*/\*+\s*`).ReplaceAllString(docBlock, "") // 移除开头的 /**
	
	lines := strings.Split(docBlock, "\n")
	var descLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = regexp.MustCompile(`^\*\s*`).ReplaceAllString(line, "")
		
		// 提取 @param fieldName description
		if strings.HasPrefix(line, "@param") {
			parts := strings.SplitN(line, " ", 3)
			if len(parts) >= 3 && parts[1] == fieldName {
				return strings.TrimSpace(parts[2])
			}
			continue
		}
		
		// 跳过其他 @ 标签
		if strings.HasPrefix(line, "@") {
			continue
		}
		
		// 过滤掉代码残留和空行
		if line != "" && 
		   !strings.Contains(line, "public class") && 
		   !strings.Contains(line, "extends") && 
		   !strings.Contains(line, "private String") &&
		   !strings.Contains(line, "/**") &&
		   line != "*/" {
			descLines = append(descLines, line)
		}
	}
	
	result := strings.Join(descLines, " ")
	// 清理多重空格和残留的 /**
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.TrimPrefix(result, "/** ")
	return strings.TrimSpace(result)
}

func javaTypeToSchema(javaType string) *Schema {
	switch javaType {
	case "String":
		return &Schema{Type: "string"}
	case "Integer", "int":
		return &Schema{Type: "integer", Format: "int32"}
	case "Long", "long":
		return &Schema{Type: "integer", Format: "int64"}
	case "Boolean", "boolean":
		return &Schema{Type: "boolean"}
	case "String[]":
		return &Schema{Type: "array", Items: &Schema{Type: "string"}}
	case "Integer[]", "int[]":
		return &Schema{Type: "array", Items: &Schema{Type: "integer"}}
	default:
		// 可能是自定义类型，作为 string 处理
		return &Schema{Type: "string"}
	}
}

func parseSchemas() map[string]*Schema {
	// 简化版：扫描所有 Model 文件，创建空 schema 占位
	schemas := make(map[string]*Schema)
	
	modelDirs := []string{
		filepath.Join(sdkPath, "cc", "model"),
	}

	for _, dir := range modelDirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if strings.HasSuffix(f.Name(), "Model.java") {
				name := strings.TrimSuffix(f.Name(), ".java")
				schemas[name] = &Schema{Type: "object"}
			}
		}
	}

	return schemas
}

func buildOpenAPISpec(apis []APIInfo, schemas map[string]*Schema) *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:   "Clink API",
			Version: "3.0.16",
		},
		Servers: []Server{
			{URL: "https://api-bj.clink.cn"},
			{URL: "https://api-sh.clink.cn"},
		},
		Paths: make(map[string]PathItem),
		Components: Components{
			Schemas: schemas,
		},
	}

	for _, api := range apis {
		op := &Operation{
			Summary:     api.Summary,
			OperationID: api.OperationID,
			Parameters:  api.Parameters,
			Responses: map[string]Response{
				"200": {
					Description: "成功",
					Content: map[string]MediaType{
						"application/json": {
							Schema: &Schema{Ref: "#/components/schemas/" + api.ResponseRef},
						},
					},
				},
			},
		}

		// 添加 body 参数（POST 请求）
		if len(api.BodyFields) > 0 {
			props := make(map[string]*Schema)
			for _, f := range api.BodyFields {
				props[f.Name] = f.Schema
			}
			op.RequestBody = &RequestBody{
				Content: map[string]MediaType{
					"application/x-www-form-urlencoded": {
						Schema: &Schema{
							Type:       "object",
							Properties: props,
						},
					},
				},
			}
		}

		pathItem := PathItem{}
		pathItem[api.Method] = op
		spec.Paths[api.Path] = pathItem
	}

	return spec
}

func camelToSnake(s string) string {
	// 简单转换：ListCdrIbs → list_cdr_ibs
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
