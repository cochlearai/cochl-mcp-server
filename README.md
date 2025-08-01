# Cochl MCP Server

A [Model Context Protocol(MCP)](https://modelcontextprotocol.io/introduction) Server for Cochl

[![cochl-mcp-server-userguide](https://github.com/user-attachments/assets/27ad3144-1616-4a50-b865-0c567bb35465)](https://www.youtube.com/watch?v=lRCQLkYi20A "Cochl.Sense MCP server User Guide")

## Use cases
- For easy analysis by integrating Cochl Sense results with an LLM.

## Installation

### Option1: Build from source
```bash
git clone https://github.com/cochlearai/cochl-mcp-server.git
go build -o cochl-mcp-server cmd/cochl-mcp-server/main.go

# OR

go install github.com/cochlearai/cochl-mcp-server/cmd/cochl-mcp-server@v0.0.2
```
- Place the executable file in your system's `$PATH`

### Option2: Download pre-built binary
- Visit [Releases page](https://github.com/cochlearai/cochl-mcp-server/releases)
- Download the appropriate version for your operating system and architecture
- Place the executable file in your system's `$PATH`

## Configuration

### Claude Desktop / Cursor IDE
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
