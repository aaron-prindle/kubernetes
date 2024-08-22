package linters

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var TodoAnalyzer = &analysis.Analyzer{
	Name: "todo",
	Doc:  "finds todos without author",
	Run:  run,
}

var (
	logger  *log.Logger
	logFile *os.File
)

func init() {
	var err error
	logFile, err = os.OpenFile("feature_gate_linter.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("error opening log file: %v", err))
	}
	logger = log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Println("In init function")
}

type featureInfo struct {
	Name       string
	Default    bool
	PreRelease string
	Spec       *ast.ValueSpec
}

func run(pass *analysis.Pass) (interface{}, error) {
	logger.Println("Starting run function")
	for _, file := range pass.Files {
		logger.Printf("Processing file: %s", pass.Fset.File(file.Pos()).Name())
		if !hasFeatureGateImport(file) {
			logger.Println("File does not have feature gate import, skipping")
			continue
		}
		var allFeatures []featureInfo
		ast.Inspect(file, func(n ast.Node) bool {
			if constDecl, ok := n.(*ast.GenDecl); ok && constDecl.Tok == token.CONST {
				logger.Println("Found const declaration")
				for _, spec := range constDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						logger.Printf("Processing value spec: %s", valueSpec.Names[0].Name)
						if feature := extractFeatureInfo(valueSpec); feature != nil {
							allFeatures = append(allFeatures, *feature)
						}
					}
				}
			}
			return true
		})
		for _, feature := range allFeatures {
			logger.Printf("Reporting feature: %s", feature.Name)
			pass.Report(analysis.Diagnostic{
				Pos:      feature.Spec.Pos(),
				Message:  formatFeatureInfo(feature),
				Category: "featuregate",
			})
		}
	}
	logger.Println("Finished run function")
	return nil, nil
}

func hasFeatureGateImport(file *ast.File) bool {
	logger.Println("Checking for feature gate import")
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == `"k8s.io/component-base/featuregate"` {
			logger.Println("Feature gate import found")
			return true
		}
	}
	logger.Println("Feature gate import not found")
	return false
}

// TODO(aaron-prindle) ISSUE IS THIS IS always nil MOST LIKELY
func extractFeatureInfo(spec *ast.ValueSpec) *featureInfo {
	logger.Printf("Extracting feature info from spec: %s", spec.Names[0].Name)

	if len(spec.Names) == 0 || len(spec.Values) == 0 {
		logger.Printf("Returning nil: spec.Names or spec.Values is empty for spec: %v", spec)
		return nil
	}

	name := spec.Names[0].Name

	// Check if the value is a BasicLit (which is likely the case for string literals)
	basicLit, ok := spec.Values[0].(*ast.BasicLit)
	if !ok {
		logger.Printf("spec.Values[0] is not *ast.BasicLit, it's %T", spec.Values[0])
		return nil
	}

	// Check if the BasicLit is a string
	if basicLit.Kind != token.STRING {
		logger.Printf("BasicLit is not a string: %v", basicLit)
		return nil
	}

	info := &featureInfo{
		Name: name,
		Spec: spec,
	}

	// Extract Default and PreRelease from comments
	if spec.Doc != nil {
		for _, comment := range spec.Doc.List {
			text := comment.Text
			if strings.Contains(text, "Default:") {
				info.Default = strings.Contains(text, "true")
				logger.Printf("Found Default in comment: %v", info.Default)
			}
			if strings.Contains(text, "PreRelease:") {
				info.PreRelease = strings.TrimSpace(strings.Split(text, "PreRelease:")[1])
				logger.Printf("Found PreRelease in comment: %s", info.PreRelease)
			}
		}
	} else {
		logger.Printf("No Doc comments found for spec: %v", spec)
	}

	logger.Printf("Successfully extracted feature: %s, Default: %v, PreRelease: %s", info.Name, info.Default, info.PreRelease)
	return info
}

func formatFeatureInfo(info featureInfo) string {
	formattedInfo := fmt.Sprintf("Feature: %s, Default: %v, PreRelease: %s",
		info.Name, info.Default, info.PreRelease)
	logger.Printf("Formatted feature info: %s", formattedInfo)
	return formattedInfo
}
