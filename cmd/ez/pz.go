package main

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pzCmd = &cobra.Command{
	Use:   "pz [project-name]",
	Short: "Scaffold a new EZ project",
	Long: `Scaffold a new EZ project with templates.

Examples:
  ez pz                      Interactive mode
  ez pz myproject            Create with basic template
  ez pz myproject -t cli     Create with cli template
  ez pz myproject -t lib -c  Create library with comments

Templates:
  basic  - Single file hello world (default)
  cli    - CLI application with arg handling
  lib    - Reusable library module
  multi  - Multi-module project
  server - HTTP server (use -s for minimal/normal)
  client - HTTP client (use -s for minimal/normal)`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPz,
}

func runPz(cmd *cobra.Command, args []string) {
	template, _ := cmd.Flags().GetString("template")
	comments, _ := cmd.Flags().GetBool("comments")
	force, _ := cmd.Flags().GetBool("force")
	serverType, _ := cmd.Flags().GetString("server-type")

	var name string

	if len(args) == 0 {
		// Interactive mode
		name = promptForInput("Project name: ", "")
		if name == "" {
			fmt.Println("Project name is required")
			return
		}

		if !cmd.Flags().Changed("template") {
			template = promptForInput("Template (basic/cli/lib/multi/server/client) [basic]: ", "basic")
			if template == "" {
				template = "basic"
			}
		}

		if (template == "server" || template == "client") && !cmd.Flags().Changed("server-type") {
			serverType = promptForInput("Type (minimal/normal) [normal]: ", "normal")
			if serverType == "" {
				serverType = "normal"
			}
		}

		if !cmd.Flags().Changed("comments") {
			commentsStr := promptForInput("Include comments? (y/N): ", "n")
			comments = strings.ToLower(commentsStr) == "y" || strings.ToLower(commentsStr) == "yes"
		}
	} else {
		name = args[0]
	}

	// Validate template
	validTemplates := map[string]bool{"basic": true, "cli": true, "lib": true, "multi": true, "server": true, "client": true}
	if !validTemplates[template] {
		fmt.Printf("Invalid template '%s'. Choose from: basic, cli, lib, multi, server, client\n", template)
		return
	}

	// Validate sub-type for server/client
	if template == "server" || template == "client" {
		validSubTypes := map[string]bool{"minimal": true, "normal": true}
		if !validSubTypes[serverType] {
			fmt.Printf("Invalid type '%s'. Choose from: minimal, normal\n", serverType)
			return
		}
	}

	// Create the project
	if err := createProject(name, template, comments, force, serverType); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Success message
	switch template {
	case "server":
		fmt.Printf("\nDone! Run your server:\n")
		fmt.Printf("  cd %s && ez main.ez\n", name)
		fmt.Printf("  # Then visit http://localhost:8080\n")
	case "client":
		fmt.Printf("\nDone! Run your client:\n")
		fmt.Printf("  cd %s && ez main.ez\n", name)
	default:
		fmt.Printf("\nDone! Run your project:\n")
		fmt.Printf("  cd %s && ez main.ez\n", name)
	}
}

func promptForInput(prompt, defaultVal string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func createProject(name, template string, comments, force bool, serverType string) error {
	// Check if directory exists
	if _, err := os.Stat(name); err == nil {
		if !force {
			return fmt.Errorf("directory '%s' already exists (use --force to overwrite)", name)
		}
		os.RemoveAll(name)
	}

	fmt.Printf("\nCreating project '%s' with template '%s'...\n", name, template)

	switch template {
	case "basic":
		return createBasicProject(name, comments)
	case "cli":
		return createCLIProject(name, comments)
	case "lib":
		return createLibProject(name, comments)
	case "multi":
		return createMultiProject(name, comments)
	case "server":
		return createServerProject(name, comments, serverType)
	case "client":
		return createClientProject(name, comments, serverType)
	}

	return nil
}

func createBasicProject(name string, comments bool) error {
	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/\n", name)

	content := getBasicMainContent(comments)
	if err := writeProjectFile(filepath.Join(name, "main.ez"), content); err != nil {
		return err
	}

	return nil
}

func createCLIProject(name string, comments bool) error {
	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/\n", name)

	baseName := filepath.Base(name)
	mainContent := getCLIMainContent(baseName, comments)
	if err := writeProjectFile(filepath.Join(name, "main.ez"), mainContent); err != nil {
		return err
	}

	commandsContent := getCLICommandsContent(comments)
	if err := writeProjectFile(filepath.Join(name, "commands.ez"), commandsContent); err != nil {
		return err
	}

	return nil
}

func createLibProject(name string, comments bool) error {
	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/\n", name)

	if err := os.MkdirAll(filepath.Join(name, "internal"), 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/internal/\n", name)

	// Use just the base name for the library file
	baseName := filepath.Base(name)
	libContent := getLibMainContent(baseName, comments)
	if err := writeProjectFile(filepath.Join(name, baseName+".ez"), libContent); err != nil {
		return err
	}

	helpersContent := getLibHelpersContent(comments)
	if err := writeProjectFile(filepath.Join(name, "internal", "helpers.ez"), helpersContent); err != nil {
		return err
	}

	return nil
}

func createMultiProject(name string, comments bool) error {
	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/\n", name)

	if err := os.MkdirAll(filepath.Join(name, "src"), 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/src/\n", name)

	if err := os.MkdirAll(filepath.Join(name, "internal"), 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/internal/\n", name)

	mainContent := getMultiMainContent(comments)
	if err := writeProjectFile(filepath.Join(name, "main.ez"), mainContent); err != nil {
		return err
	}

	appContent := getMultiAppContent(comments)
	if err := writeProjectFile(filepath.Join(name, "src", "app.ez"), appContent); err != nil {
		return err
	}

	configContent := getMultiConfigContent(comments)
	if err := writeProjectFile(filepath.Join(name, "src", "config.ez"), configContent); err != nil {
		return err
	}

	utilsContent := getMultiUtilsContent(comments)
	if err := writeProjectFile(filepath.Join(name, "internal", "utils.ez"), utilsContent); err != nil {
		return err
	}

	return nil
}

func writeProjectFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("  created %s\n", path)
	return nil
}

// Template content functions

func getBasicMainContent(comments bool) string {
	if comments {
		return `// main.ez - Entry point
//
// EZ Quick Reference:
// - Variables: temp x int = 5 or temp x = 5 (inferred)
// - Constants: const PI float = 3.14
// - Functions: do greet(name string) -> string { return "Hi " + name }
// - Loops: for i in range(0, 10) { } or for item in items { }
// - Conditionals: if x > 0 { } or x < 0 { } otherwise { }

module main

import @std
using std

do main() {
    println("Hello, World!")
}
`
	}
	return `// main.ez - Entry point

module main

import @std
using std

do main() {
    println("Hello, World!")
}
`
}

func getCLIMainContent(name string, comments bool) string {
	if comments {
		return fmt.Sprintf(`// main.ez - CLI application entry point
//
// Run with: ez main.ez [command] [args...]
// Example: ez main.ez greet Alice
//
// Note: os.args() returns [program, filename, command, args...]
//       args[0] = "ez", args[1] = "main.ez", args[2] = command

module main

import @std, @os
using std

do main() {
    temp args [string] = os.args()

    if len(args) < 3 {
        showUsage()
        return
    }

    temp command string = args[2]

    // Dispatch to command handlers
    if command == "greet" {
        handleGreet(args)
    } or command == "help" {
        showUsage()
    } otherwise {
        println("Unknown command: " + command)
        showUsage()
    }
}

do showUsage() {
    println("%s - A CLI application")
    println("")
    println("Usage:")
    println("  ez main.ez <command> [args...]")
    println("")
    println("Commands:")
    println("  greet <name>  Greet someone")
    println("  help          Show this help")
}
`, name)
	}
	return fmt.Sprintf(`// main.ez - CLI application entry point

module main

import @std, @os
using std

do main() {
    temp args [string] = os.args()

    if len(args) < 3 {
        showUsage()
        return
    }

    temp command string = args[2]

    if command == "greet" {
        handleGreet(args)
    } or command == "help" {
        showUsage()
    } otherwise {
        println("Unknown command: " + command)
        showUsage()
    }
}

do showUsage() {
    println("%s - A CLI application")
    println("")
    println("Usage:")
    println("  ez main.ez <command> [args...]")
    println("")
    println("Commands:")
    println("  greet <name>  Greet someone")
    println("  help          Show this help")
}
`, name)
}

func getCLICommandsContent(comments bool) string {
	if comments {
		return `// commands.ez - Command implementations
//
// Each command handler receives the full args array.
// args[0] = program, args[1] = filename, args[2] = command, args[3+] = arguments

module main

import @std
using std

do handleGreet(args [string]) {
    if len(args) < 4 {
        println("Usage: greet <name>")
        return
    }

    temp name string = args[3]
    println("Hello, " + name + "!")
}
`
	}
	return `// commands.ez - Command implementations

module main

import @std
using std

do handleGreet(args [string]) {
    if len(args) < 4 {
        println("Usage: greet <name>")
        return
    }

    temp name string = args[3]
    println("Hello, " + name + "!")
}
`
}

func getLibMainContent(name string, comments bool) string {
	if comments {
		return fmt.Sprintf(`// %s.ez - Main library module
//
// This is a reusable library. Import it from other projects:
//   import "./%s"
//
// Public functions should be documented and well-named.
// Internal helpers go in the internal/ directory.

module %s

import "./internal/helpers"

#doc("greet returns a greeting message for the given name.")
do greet(name string) -> string {
    return formatGreeting(name)
}

#doc("add returns the sum of two integers.")
do add(a int, b int) -> int {
    return a + b
}
`, name, name, name)
	}
	return fmt.Sprintf(`// %s.ez - Main library module

module %s

import "./internal/helpers"

do greet(name string) -> string {
    return formatGreeting(name)
}

do add(a int, b int) -> int {
    return a + b
}
`, name, name)
}

func getLibHelpersContent(comments bool) string {
	if comments {
		return `// helpers.ez - Internal helper functions
//
// These are implementation details not meant for public use.
// Keep internal logic here to keep the main module clean.

module helpers

do formatGreeting(name string) -> string {
    return "Hello, " + name + "!"
}
`
	}
	return `// helpers.ez - Internal helper functions

module helpers

do formatGreeting(name string) -> string {
    return "Hello, " + name + "!"
}
`
}

func getMultiMainContent(comments bool) string {
	if comments {
		return `// main.ez - Application entry point
//
// Multi-module project structure:
//   main.ez         - Entry point
//   src/app.ez      - Core application logic
//   src/config.ez   - Configuration handling
//   internal/       - Internal utilities

module main

import "./src/app"

do main() {
    app.run()
}
`
	}
	return `// main.ez - Application entry point

module main

import "./src/app"

do main() {
    app.run()
}
`
}

func getMultiAppContent(comments bool) string {
	if comments {
		return `// app.ez - Core application logic
//
// This module contains the main application functionality.
// Import configuration and utilities as needed.

module app

import @std
import "./config"
import "../internal/utils"
using std

do run() {
    temp cfg config.Config = config.getConfig()
    println("Starting " + cfg.name + "...")

    temp message string = utils.formatMessage("Application initialized")
    println(message)
}
`
	}
	return `// app.ez - Core application logic

module app

import @std
import "./config"
import "../internal/utils"
using std

do run() {
    temp cfg config.Config = config.getConfig()
    println("Starting " + cfg.name + "...")

    temp message string = utils.formatMessage("Application initialized")
    println(message)
}
`
}

func getMultiConfigContent(comments bool) string {
	if comments {
		return `// config.ez - Configuration handling
//
// Define your application configuration here.
// Consider loading from environment or config files.

module config

const Config struct {
    name string
    version string
    debug bool
}

do getConfig() -> Config {
    return Config{
        name: "MyApp",
        version: "1.0.0",
        debug: false
    }
}
`
	}
	return `// config.ez - Configuration handling

module config

const Config struct {
    name string
    version string
    debug bool
}

do getConfig() -> Config {
    return Config{
        name: "MyApp",
        version: "1.0.0",
        debug: false
    }
}
`
}

func getMultiUtilsContent(comments bool) string {
	if comments {
		return `// utils.ez - Internal utilities
//
// Helper functions used across the application.
// Keep these generic and reusable.

module utils

do formatMessage(msg string) -> string {
    return "[INFO] " + msg
}

do formatError(msg string) -> string {
    return "[ERROR] " + msg
}
`
	}
	return `// utils.ez - Internal utilities

module utils

do formatMessage(msg string) -> string {
    return "[INFO] " + msg
}

do formatError(msg string) -> string {
    return "[ERROR] " + msg
}
`
}

func createServerProject(name string, comments bool, serverType string) error {
	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/\n", name)

	if serverType == "minimal" {
		content := getServerMinimalContent(comments)
		return writeProjectFile(filepath.Join(name, "main.ez"), content)
	}

	// Normal server
	mainContent := getServerMainContent(comments)
	if err := writeProjectFile(filepath.Join(name, "main.ez"), mainContent); err != nil {
		return err
	}

	routesContent := getServerRoutesContent(comments)
	return writeProjectFile(filepath.Join(name, "routes.ez"), routesContent)
}

func getServerMinimalContent(comments bool) string {
	if comments {
		return `// main.ez - Minimal HTTP server
//
// Run with: ez main.ez
// Test with: curl http://localhost:8080

module main

import @std, @server
using std

do main() {
    temp router Router = server.router()
    server.route(router, "GET", "/", server.text(200, "Hello, World!"))

    println("Server running on http://localhost:8080")
    server.listen(8080, router)
}
`
	}
	return `// main.ez - Minimal HTTP server

module main

import @std, @server
using std

do main() {
    temp router Router = server.router()
    server.route(router, "GET", "/", server.text(200, "Hello, World!"))

    println("Server running on http://localhost:8080")
    server.listen(8080, router)
}
`
}

func getServerMainContent(comments bool) string {
	if comments {
		return `// main.ez - HTTP server entry point
//
// Run with: ez main.ez
// Test with:
//   curl http://localhost:8080
//   curl http://localhost:8080/health
//   curl http://localhost:8080/api/status

module main

import @std, @server
using std

do main() {
    temp router Router = server.router()
    registerRoutes(router)

    println("Server running on http://localhost:8080")
    server.listen(8080, router)
}
`
	}
	return `// main.ez - HTTP server entry point

module main

import @std, @server
using std

do main() {
    temp router Router = server.router()
    registerRoutes(router)

    println("Server running on http://localhost:8080")
    server.listen(8080, router)
}
`
}

func getServerRoutesContent(comments bool) string {
	if comments {
		return `// routes.ez - Route definitions
//
// Register all routes here. Each route needs:
//   server.route(router, METHOD, PATH, RESPONSE)
//
// Response helpers:
//   server.text(status, body)  - Plain text response
//   server.json(status, data)  - JSON response
//   server.html(status, body)  - HTML response

module main

import @server

do registerRoutes(router Router) {
    // Home page
    server.route(router, "GET", "/", server.text(200, "Welcome to EZ!"))

    // Health check endpoint
    server.route(router, "GET", "/health", server.json(200, {"status": "ok"}))

    // API endpoint
    server.route(router, "GET", "/api/status", server.json(200, {
        "version": "1.0.0",
        "status": "running"
    }))
}
`
	}
	return `// routes.ez - Route definitions

module main

import @server

do registerRoutes(router Router) {
    server.route(router, "GET", "/", server.text(200, "Welcome to EZ!"))

    server.route(router, "GET", "/health", server.json(200, {"status": "ok"}))

    server.route(router, "GET", "/api/status", server.json(200, {
        "version": "1.0.0",
        "status": "running"
    }))
}
`
}

func createClientProject(name string, comments bool, clientType string) error {
	if err := os.MkdirAll(name, 0755); err != nil {
		return err
	}
	fmt.Printf("  created %s/\n", name)

	if clientType == "minimal" {
		content := getClientMinimalContent(comments)
		return writeProjectFile(filepath.Join(name, "main.ez"), content)
	}

	// Normal client
	mainContent := getClientMainContent(comments)
	if err := writeProjectFile(filepath.Join(name, "main.ez"), mainContent); err != nil {
		return err
	}

	apiContent := getClientAPIContent(comments)
	return writeProjectFile(filepath.Join(name, "api.ez"), apiContent)
}

func getClientMinimalContent(comments bool) string {
	if comments {
		return `// main.ez - Minimal HTTP client
//
// Run with: ez main.ez
//
// http.get(url) returns (Response, Error)
// Response fields: .status (int), .body (string), .headers (map)

module main

import @std, @http
using std

do main() {
    temp resp, err = http.get("https://httpbin.org/get")
    if err != nil {
        println("Error: ${err.message}")
        return
    }
    println("Status: ${resp.status}")
    println("Body: ${resp.body}")
}
`
	}
	return `// main.ez - Minimal HTTP client

module main

import @std, @http
using std

do main() {
    temp resp, err = http.get("https://httpbin.org/get")
    if err != nil {
        println("Error: ${err.message}")
        return
    }
    println("Status: ${resp.status}")
    println("Body: ${resp.body}")
}
`
}

func getClientMainContent(comments bool) string {
	if comments {
		return `// main.ez - HTTP client entry point
//
// Run with: ez main.ez
//
// This project demonstrates GET and POST requests
// with error handling and JSON payloads.

module main

import @std, @http
using std

do main() {
    // GET request
    temp resp, err = fetchStatus()
    if err != nil {
        println("Error: ${err.message}")
        return
    }
    println("GET Status: ${resp.status}")
    println("GET Body: ${resp.body}")

    // POST request with JSON body
    temp post_resp, post_err = createPost("Hello from EZ!", "This is a test post.")
    if post_err != nil {
        println("Error: ${post_err.message}")
        return
    }
    println("POST Status: ${post_resp.status}")
    println("POST Body: ${post_resp.body}")
}
`
	}
	return `// main.ez - HTTP client entry point

module main

import @std, @http
using std

do main() {
    temp resp, err = fetchStatus()
    if err != nil {
        println("Error: ${err.message}")
        return
    }
    println("GET Status: ${resp.status}")
    println("GET Body: ${resp.body}")

    temp post_resp, post_err = createPost("Hello from EZ!", "This is a test post.")
    if post_err != nil {
        println("Error: ${post_err.message}")
        return
    }
    println("POST Status: ${post_resp.status}")
    println("POST Body: ${post_resp.body}")
}
`
}

func getClientAPIContent(comments bool) string {
	if comments {
		return `// api.ez - API request functions
//
// Each function wraps an HTTP call and returns (Response, Error).
//
// Available request functions:
//   http.get(url)                              - GET request
//   http.post(url, body)                       - POST request
//   http.put(url, body)                        - PUT request
//   http.delete(url)                           - DELETE request
//   http.patch(url, body)                      - PATCH request
//   http.request(method, url, body, headers, timeout) - Custom request
//
// Helpers:
//   http.json_body(data)                       - Convert map to JSON string
//   http.build_query(params)                   - Build URL query string

module main

import @http

do fetchStatus() -> (Response, Error) {
    return http.get("https://httpbin.org/get")
}

do createPost(title string, body string) -> (Response, Error) {
    return http.post(
        "https://httpbin.org/post",
        http.json_body({"title": title, "body": body})
    )
}
`
	}
	return `// api.ez - API request functions

module main

import @http

do fetchStatus() -> (Response, Error) {
    return http.get("https://httpbin.org/get")
}

do createPost(title string, body string) -> (Response, Error) {
    return http.post(
        "https://httpbin.org/post",
        http.json_body({"title": title, "body": body})
    )
}
`
}
