package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getkin/kin-openapi/openapi3"
)

type multiFlag []string

func (m *multiFlag) String() string {
	return fmt.Sprintf("%v", []string(*m))
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	var inputs multiFlag
	var output string
	flag.Var(&inputs, "in", "input OpenAPI document (repeatable)")
	flag.StringVar(&output, "out", filepath.Join("docs", "openapi", "combined.json"), "output path")
	flag.Parse()

	if len(inputs) == 0 {
		fmt.Fprintln(os.Stderr, "swagger-merge: at least one --in file is required")
		os.Exit(1)
	}

	loader := &openapi3.Loader{IsExternalRefsAllowed: true}

	combined := &openapi3.T{
		OpenAPI: "3.0.3",
		Info:    &openapi3.Info{Title: "Combined API", Version: "0.1.0"},
		Paths:   make(openapi3.Paths),
		Components: openapi3.Components{
			Schemas:         openapi3.Schemas{},
			SecuritySchemes: openapi3.SecuritySchemes{},
		},
	}

	seenOperationIDs := make(map[string]string)

	for _, input := range inputs {
		doc, err := loader.LoadFromFile(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "swagger-merge: load %s: %v\n", input, err)
			os.Exit(1)
		}

		if combined.Info == nil || combined.Info.Title == "Combined API" {
			combined.Info = doc.Info
		}

		if err := mergeComponents(combined, doc); err != nil {
			fmt.Fprintf(os.Stderr, "swagger-merge: %v\n", err)
			os.Exit(1)
		}

		for path, item := range doc.Paths {
			if item == nil {
				continue
			}
			if existing, ok := combined.Paths[path]; ok {
				if err := mergePathItem(existing, item, seenOperationIDs); err != nil {
					fmt.Fprintf(os.Stderr, "swagger-merge: %v\n", err)
					os.Exit(1)
				}
			} else {
				if err := registerOperations(item, seenOperationIDs); err != nil {
					fmt.Fprintf(os.Stderr, "swagger-merge: %v\n", err)
					os.Exit(1)
				}
				combined.Paths[path] = item
			}
		}
	}

	data, err := json.MarshalIndent(combined, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "swagger-merge: marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "swagger-merge: create dir: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(output, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "swagger-merge: write output: %v\n", err)
		os.Exit(1)
	}
}

func mergeComponents(target, source *openapi3.T) error {
	if source.Components.Schemas != nil {
		if target.Components.Schemas == nil {
			target.Components.Schemas = openapi3.Schemas{}
		}
		for name, schema := range source.Components.Schemas {
			if _, exists := target.Components.Schemas[name]; exists {
				return fmt.Errorf("duplicate schema %s", name)
			}
			target.Components.Schemas[name] = schema
		}
	}
	if source.Components.SecuritySchemes != nil {
		if target.Components.SecuritySchemes == nil {
			target.Components.SecuritySchemes = openapi3.SecuritySchemes{}
		}
		for name, scheme := range source.Components.SecuritySchemes {
			if _, exists := target.Components.SecuritySchemes[name]; exists {
				continue
			}
			target.Components.SecuritySchemes[name] = scheme
		}
	}
	return nil
}

func mergePathItem(target, source *openapi3.PathItem, seen map[string]string) error {
	if err := mergeOperation(&target.Get, source.Get, seen, "GET"); err != nil {
		return err
	}
	if err := mergeOperation(&target.Put, source.Put, seen, "PUT"); err != nil {
		return err
	}
	if err := mergeOperation(&target.Post, source.Post, seen, "POST"); err != nil {
		return err
	}
	if err := mergeOperation(&target.Delete, source.Delete, seen, "DELETE"); err != nil {
		return err
	}
	if err := mergeOperation(&target.Options, source.Options, seen, "OPTIONS"); err != nil {
		return err
	}
	if err := mergeOperation(&target.Head, source.Head, seen, "HEAD"); err != nil {
		return err
	}
	if err := mergeOperation(&target.Patch, source.Patch, seen, "PATCH"); err != nil {
		return err
	}
	if err := mergeOperation(&target.Trace, source.Trace, seen, "TRACE"); err != nil {
		return err
	}
	return nil
}

func mergeOperation(target **openapi3.Operation, source *openapi3.Operation, seen map[string]string, method string) error {
	if source == nil {
		return nil
	}
	if err := registerOperation(source, seen); err != nil {
		return err
	}
	if *target != nil {
		return fmt.Errorf("duplicate operation for method %s with id %s", method, source.OperationID)
	}
	*target = source
	return nil
}

func registerOperations(item *openapi3.PathItem, seen map[string]string) error {
	if err := registerOperation(item.Get, seen); err != nil {
		return err
	}
	if err := registerOperation(item.Put, seen); err != nil {
		return err
	}
	if err := registerOperation(item.Post, seen); err != nil {
		return err
	}
	if err := registerOperation(item.Delete, seen); err != nil {
		return err
	}
	if err := registerOperation(item.Options, seen); err != nil {
		return err
	}
	if err := registerOperation(item.Head, seen); err != nil {
		return err
	}
	if err := registerOperation(item.Patch, seen); err != nil {
		return err
	}
	if err := registerOperation(item.Trace, seen); err != nil {
		return err
	}
	return nil
}

func registerOperation(op *openapi3.Operation, seen map[string]string) error {
	if op == nil || op.OperationID == "" {
		return nil
	}
	if existing, ok := seen[op.OperationID]; ok {
		return fmt.Errorf("duplicate operationId %s (already used at %s)", op.OperationID, existing)
	}
	seen[op.OperationID] = op.OperationID
	return nil
}
