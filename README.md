# Scope

## Installation

```bash
# Clone the repository
git clone https://github.com/TFMV/scope.git
cd scope

# Build the project
go build -o scope ./cmd/scope
```

## Usage

Scope runs as a local MCP (Machine Conversation Protocol) server that can be integrated with compatible clients. To use Scope:

1. Set the `GO_REPO_PATH` environment variable to point to your Go repository:

   ```bash
   export GO_REPO_PATH=/path/to/your/go/repo
   ```

2. Run the Scope server:

   ```bash
   ./scope
   ```

The server will start and listen for MCP protocol messages on stdin/stdout. It can be integrated with any MCP-compatible client to provide code analysis and assistance features.

## Available Tools

### Lookup Type

Get documentation and definition of a Go type:

```json
{
  "type_name": "YourType"
}
```

### List Methods

List public methods for a Go type:

```json
{
  "type_name": "YourType"
}
```

### Show Example

Get example usage for a type or topic:

```json
{
  "topic": "YourType"
}
```

### Code Search

Search through codebase using semantic search:

```json
{
  "query": "your search query"
}
```

### Code Edit

Edit code files with AI assistance:

```json
{
  "file": "path/to/file.go",
  "changes": "description of changes"
}
```

### Code Review

Review code changes and provide feedback:

```json
{
  "changes": "code changes to review"
}
```

## Architecture

Scope is built with a modular architecture:

- `cmd/scope`: Main application entry point and MCP server implementation
- `internal/analyzer`: Core Go code analysis functionality
- `internal/cache`: Caching system for improved performance
- `internal/tools`: Tool management and configuration

The server uses the MCP protocol for communication, which provides a standardized way for clients to interact with the code analysis tools.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
