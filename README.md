# Cochl MCP Server

A [Model Context Protocol(MCP)](https://modelcontextprotocol.io/introduction) Server for Cochl

[![cochl-mcp-server-userguide](https://github.com/user-attachments/assets/27ad3144-1616-4a50-b865-0c567bb35465)](https://www.youtube.com/watch?v=lRCQLkYi20A "Cochl.Sense MCP server User Guide")

## Use cases
- For easy analysis by integrating Cochl Sense results with an LLM.

## Installation

**Prerequisites**: [ffmpeg](https://ffmpeg.org/download.html) must be installed on your system.
> If using Docker, ffmpeg installation is not required as it is already included in the image.

### Option1: Download pre-built binary
- Visit [Releases page](https://github.com/cochlearai/cochl-mcp-server/releases)
- Download the appropriate version for your operating system and architecture
- Place the executable file in your system's `$PATH`

### Option2: Build docker image
```bash
git clone https://github.com/cochlearai/cochl-mcp-server
cd cochl-mcp-server
docker build -t cochl-mcp-server .
```

## Configuration

### If using the Docker
```json
{
	"cochl": {
		"command": "docker",
		"args": [
			"run",
			"--rm",
			"-i",
			"-v",
			"/allowed/directory/you-want:/allowed/directory/you-want",
			"-e" ,
			"COCHL_SENSE_PROJECT_KEY",
			"-e",
			"COCHL_SENSE_BASE_URL",
			"cochl-mcp-server:latest"
		],
		"env":{
      "COCHL_SENSE_PROJECT_KEY": "<your project key>",
			"COCHL_SENSE_BASE_URL": "https://api.cochl.ai"
		}
	},
}
```
To access files on the host from the container, you can mount them using the same path.

### If using the binary
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
  - file_url: Audio file URL or localfile absoulte path (MP3/WAV/OGG). (string, required)
  - with_caption: Generate a natural language caption for the audio file (default: false)
