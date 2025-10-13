package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/shravanasati/shadowfax/internal/response"
)

func main() {
	// Example 1: Simple template with string data
	fmt.Println("=== Example 1: Simple Template ===")
	simpleTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>Welcome</title>
</head>
<body>
    <h1>Hello, {{.}}!</h1>
    <p>Welcome to Shadowfax template response.</p>
</body>
</html>`

	resp1, err := response.NewTemplateResponse(simpleTemplate, "World")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Writing simple template response:")
	resp1.Write(os.Stdout)
	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 2: Template with struct data
	fmt.Println("=== Example 2: Template with Struct Data ===")
	type PageData struct {
		Title   string
		Content string
		Items   []string
	}

	structTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Content}}</p>
    <ul>
    {{range .Items}}
        <li>{{.}}</li>
    {{end}}
    </ul>
</body>
</html>`

	data := PageData{
		Title:   "My Todo List",
		Content: "Here are the things I need to do:",
		Items:   []string{"Buy groceries", "Walk the dog", "Finish the project"},
	}

	resp2, err := response.NewTemplateResponse(structTemplate, data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Writing struct template response:")
	resp2.Write(os.Stdout)
	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 3: Template with custom functions
	fmt.Println("=== Example 3: Template with Custom Functions ===")
	funcTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>{{upper .title}}</title>
</head>
<body>
    <h1>{{upper .title}}</h1>
    <p>Total items: {{add (len .items) 1}}</p>
    <ul>
    {{range .items}}
        <li>{{upper .}}</li>
    {{end}}
    </ul>
</body>
</html>`

	funcMap := template.FuncMap{
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
	}

	funcData := map[string]interface{}{
		"title": "custom functions demo",
		"items": []string{"item one", "item two", "item three"},
	}

	resp3, err := response.NewTemplateResponseWithFuncs(funcTemplate, funcMap, funcData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Writing template with custom functions:")
	resp3.Write(os.Stdout)
	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 4: Setting custom status code and headers
	fmt.Println("=== Example 4: Custom Status and Headers ===")
	customTemplate := `<h1>{{.message}}</h1>`
	customData := map[string]string{"message": "Not Found"}

	resp4, err := response.NewTemplateResponse(customTemplate, customData)
	if err != nil {
		log.Fatal(err)
	}

	// Modify the response with custom headers and status
	resp4 = resp4.WithStatusCode(404).
		WithHeader("X-Custom-Header", "Template-Response").
		WithHeader("Cache-Control", "no-cache")

	fmt.Println("Writing custom template response with 404 status:")
	resp4.Write(os.Stdout)
}
