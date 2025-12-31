# Cochl MCP Server

A [Model Context Protocol(MCP)](https://modelcontextprotocol.io/introduction) Server for Cochl

[![cochl-mcp-server-userguide](https://github.com/user-attachments/assets/27ad3144-1616-4a50-b865-0c567bb35465)](https://www.youtube.com/watch?v=lRCQLkYi20A "Cochl.Sense MCP server User Guide")

## Use cases
- For easy analysis by integrating Cochl Sense results with an LLM.
- Audio and video file analysis with sound event detection and automatic captioning.
- Video frame analysis using Ollama (LLaVA) for visual understanding.

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
			"-e",
			"COCHL_SENSE_PROJECT_KEY",
			"-e",
			"COCHL_SENSE_BASE_URL",
			"-e",
			"OLLAMA_BASE_URL",
			"-e",
			"OLLAMA_MODEL",
			"cochl-mcp-server:latest"
		],
		"env": {
			"COCHL_SENSE_PROJECT_KEY": "<your project key>",
			"COCHL_SENSE_BASE_URL": "https://api.cochl.ai",
			"OLLAMA_BASE_URL": "http://host.docker.internal:11434",
			"OLLAMA_MODEL": "llava"
		}
	}
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
        "COCHL_SENSE_PROJECT_KEY": "<your project key>",
        "OLLAMA_BASE_URL": "http://localhost:11434",
        "OLLAMA_MODEL": "llava"
      }
    }
  }
}
```

## Tools

### analyze_media
Analyzes audio and video files using Cochl Sense API for sound event detection, automatic captioning, and video frame analysis.

#### Parameters
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `file_url` | string | Yes | - | Media file URL or local file absolute path. Supports audio (MP3/WAV/OGG) and video (MP4/WebM/AVI). Google Drive and Dropbox URLs are also supported. |
| `with_caption` | boolean | No | false | Generate a natural language caption for the audio content. |
| `max_frames` | integer | No | 8 | Maximum number of frames to extract from video for analysis (max: 16). Only applies to video files. |

#### Output
- **media_type**: `"audio"` or `"video"`
- **sense**: Sound event detection results with timestamps and probability scores
- **caption**: Natural language caption for audio content (if `with_caption` is true)
- **video_caption**: Frame-by-frame descriptions and video summary (video only, requires Ollama)

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `COCHL_SENSE_PROJECT_KEY` | Yes | - | Your Cochl Sense API project key |
| `COCHL_SENSE_BASE_URL` | No | `https://api.cochl.ai` | Cochl Sense API base URL |
| `OLLAMA_BASE_URL` | No | `http://localhost:11434` | Ollama server URL (for video frame analysis) |
| `OLLAMA_MODEL` | No | `llava` | Ollama model to use for video frame analysis |
