package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"html/template"
	"os"
	"path/filepath"
)

// pluginInfo stores plugin analysis information
type pluginInfo struct {
	Package    string         // Package name
	PluginType string         // Plugin type name
	Functions  []functionInfo // Exported function list
}

// functionInfo stores function metadata
type functionInfo struct {
	Name    string      // Function name
	Params  []paramInfo // Parameter list
	Results []paramInfo // Return value list
	IsInit  bool        // Whether it's an Init method
}

// paramInfo stores parameter metadata
type paramInfo struct {
	Name       string // Parameter name
	Type       string // Parameter type
	IsVariadic bool   // Whether it's a variadic parameter
}

// analyzeFuncDecl extracts function information from AST
func analyzeFuncDecl(fn *ast.FuncDecl) functionInfo {
	f := functionInfo{
		Name: fn.Name.Name,
	}

	// Special handling for Bureau interface methods
	switch f.Name {
	case "Name", "Version", "Free":
		return f
	case "Init":
		f.IsInit = true
		return f
	}

	// Analyze method parameters
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			typeExpr := &bytes.Buffer{}
			printer.Fprint(typeExpr, token.NewFileSet(), param.Type)
			typeStr := typeExpr.String()

			// Check if it's a variadic parameter
			if _, ok := param.Type.(*ast.Ellipsis); ok {
				for _, name := range param.Names {
					f.Params = append(f.Params, paramInfo{
						Name:       name.Name,
						Type:       typeStr,
						IsVariadic: true,
					})
				}
				continue
			}

			for _, name := range param.Names {
				f.Params = append(f.Params, paramInfo{
					Name: name.Name,
					Type: typeStr,
				})
			}
		}
	}

	// Analyze return values
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			typeExpr := &bytes.Buffer{}
			printer.Fprint(typeExpr, token.NewFileSet(), result.Type)
			typeStr := typeExpr.String()

			if len(result.Names) == 0 {
				f.Results = append(f.Results, paramInfo{
					Name: "",
					Type: typeStr,
				})
			} else {
				for _, name := range result.Names {
					f.Results = append(f.Results, paramInfo{
						Name: name.Name,
						Type: typeStr,
					})
				}
			}
		}
	}

	return f
}

const pluginTpl = `package {{ .Package }}

import (
    "context"
    "fmt"
    "github.com/zyanho/chameleon/pkg/plugin"
)

// Functions exports plugin functions
var Functions = map[string]plugin.InvokeFunc{
    {{- range .Functions }}
    "{{ .Name }}": func(ctx context.Context, args ...interface{}) (interface{}, error) {
        impl := Export.(*{{ $.PluginType }})
        
        {{- if eq .Name "Name" }}
        if len(args) != 0 {
            return nil, fmt.Errorf("Name requires 0 arguments")
        }
        return impl.Name(), nil
        {{- else if eq .Name "Version" }}
        if len(args) != 0 {
            return nil, fmt.Errorf("Version requires 0 arguments")
        }
        return impl.Version(), nil
        {{- else if eq .Name "Init" }}
        // Init method passes all parameters directly
        return nil, impl.Init(args...)
        {{- else if eq .Name "Free" }}
        if len(args) != 0 {
            return nil, fmt.Errorf("Free requires 0 arguments")
        }
        return nil, impl.Free()
        {{- else }}
        // Normal method handling
        if len(args) != {{ len .Params | add -1 }} {
            return nil, fmt.Errorf("{{ .Name }} requires {{ len .Params | add -1 }} arguments")
        }

        // Parameter type conversion
        {{- range $i, $param := .Params }}
        {{- if ne $i 0 }}
        {{ $param.Name }}, ok{{ $i }} := args[{{ add $i -1 }}].({{ $param.Type }})
        if !ok{{ $i }} {
            return nil, fmt.Errorf("argument {{ add $i -1 }} must be {{ $param.Type }}")
        }
        {{- end }}
        {{- end }}

        // Call the function and handle the return value
        {{- if eq (len .Results) 0 }}
        err := impl.{{ .Name }}(ctx{{ range $i, $param := .Params }}{{ if ne $i 0 }}, {{ $param.Name }}{{ end }}{{ end }})
        return nil, err
        {{- else if eq (len .Results) 1 }}
        result := impl.{{ .Name }}(ctx{{ range $i, $param := .Params }}{{ if ne $i 0 }}, {{ $param.Name }}{{ end }}{{ end }})
        return result, nil
        {{- else if eq (len .Results) 2 }}
        return impl.{{ .Name }}(ctx{{ range $i, $param := .Params }}{{ if ne $i 0 }}, {{ $param.Name }}{{ end }}{{ end }})
        {{- end }}
        {{- end }}
    },
    {{- end }}
}
`

// Generate analyzes plugin source code and generates wrapper code
func Generate(pluginDir string) error {
	// 1. Analyze plugin source code
	info, err := analyzePlugin(pluginDir)
	if err != nil {
		return err
	}

	// 2. Generate wrapper code
	return generateWrapper(pluginDir, info)
}

// analyzePlugin parses and analyzes plugin source code
func analyzePlugin(dir string) (*pluginInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !fi.IsDir() && filepath.Ext(fi.Name()) == ".go"
	}, 0)
	if err != nil {
		return nil, err
	}

	for pkgName, pkg := range pkgs {
		info := &pluginInfo{Package: pkgName}

		// Find types that implement the Bureau interface
		ast.Inspect(pkg, func(n ast.Node) bool {
			switch t := n.(type) {
			case *ast.TypeSpec:
				// TODO: Check if it implements the Bureau interface
				info.PluginType = t.Name.Name
			case *ast.FuncDecl:
				// Collect exported methods
				if t.Recv != nil && t.Name.IsExported() {
					info.Functions = append(info.Functions, analyzeFuncDecl(t))
				}
			}
			return true
		})

		if info.PluginType != "" {
			return info, nil
		}
	}

	return nil, fmt.Errorf("no plugin implementation found")
}

// generateWrapper generates the plugin wrapper code
func generateWrapper(dir string, info *pluginInfo) error {
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	tmpl, err := template.New("plugin").Funcs(funcMap).Parse(pluginTpl)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, info); err != nil {
		return err
	}

	outputPath := filepath.Join(dir, "plugin_wrapper.go")
	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}
