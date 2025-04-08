# Cochl MCP Server

A [Model Context Protocol(MCP)](https://modelcontextprotocol.io/introduction) Server for Cochl

## Use cases
- For easy analysis by integrating Cochl Sense results with an LLM.

## Usage

1. Build from source
```bash
git clone https://github.com/cochlearai/cochl-mcp-server.git
go build -o cochl-mcp-server cmd/main.go
```
and then place the executable file in your `$PATH`

2. Download pre-built binary
- TBD


### Cursor IDE
```json
{
  "mcpServers": {
    "cochl": {
      "command": "cochl-mcp-server",
      "args": [],
      "env": {
        "COCHL_SENSE_BASE_URL": "https://api.cochl.ai",
        "COCHL_SENSE_PROJECT_KEY": "<your project key>"
      }
    }
  }
}
```

## Tools

### Cochl Sense
- analyze_audio
  - file_absolute_path: absolute path of the audio file (string, required)
    - supported audio type (mp3, ogg, wav)
