package tools

const (
	_analyzeMediaDesc = `
Analyze audio or video files to detect sounds, events, and visual content.

For AUDIO files (MP3/WAV/OGG):
  - Detects and segments environmental sounds and events over time
  - Provides temporal segments with start/end times and probability scores
  - Optionally generates a natural language caption summarizing the audio
  Example sounds: 'Water_run', 'Laughter', 'Speech', 'Music', etc.

For VIDEO files (MP4/WebM/AVI):
  - Extracts and analyzes audio track (same as audio analysis)
  - Extracts frames uniformly across the video and analyzes all frames in a single batch using AI (LLaVA)
  - Generates descriptions for each frame and an overall video summary
  - Processes audio and video analysis concurrently for efficiency

Input parameters:
  - file_url: Path or URL to the media file
  - with_caption: Generate audio caption (default: false)
  - max_frames: Maximum number of frames to extract for video, uniformly sampled (default: 8, max: 16)

Output includes:
  - media_type: "audio" or "video"
  - sense: Temporal sound/event detection results
  - caption: Natural language audio caption (if requested)
  - video_caption: Frame-by-frame descriptions and summary (video only)
`
)
