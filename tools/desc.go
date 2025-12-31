package tools

import "fmt"

const (
	//   - max_frames: Maximum number of frames to extract for video analysis (default: 64)
	_analyzeMediaDesc = `
Analyze audio or video files to detect sounds, events, and visual content.

For AUDIO files (MP3/WAV/OGG):
  - Detects and segments environmental sounds and events over time
  - Provides temporal segments with start/end times and probability scores
  - Optionally generates a natural language caption summarizing the audio
  Example sounds: 'Water_run', 'Laughter', 'Speech', 'Music', etc.

For VIDEO files (MP4/WebM/AVI):
  - Extracts and analyzes audio track (same as audio analysis)
  - Extracts frames using intelligent sampling and analyzes them using visual AI
  - Generates a chronological summary and timeline of key events
  - Processes audio and video analysis concurrently for efficiency

Input parameters:
  - file_url: Path or URL to the media file
  - with_caption: Generate audio caption (default: false)
  - max_frames: Optional limit on video frames (default: 64, max: 180)

Output includes:
  - media_type: "audio" or "video"
  - sense: Temporal sound/event detection results
  - audio_caption: Natural language audio caption (if requested)
  - video_caption: Video summary and timeline of key events (video only)
`
	_videoAnalysisPrompt = `Analyze this video which has been sampled into %d frames. %s

Provide a detailed chronological narrative of the events in the video.
It is CRITICAL to describe actions in the exact order they happen, from the beginning (Frame 1) to the end.

Focus on:
1. The setting and environment (indoor/outdoor, time of day, lighting, weather).
2. Key objects and people visible in the scene.
3. Any visible text, signs, or timestamps (OCR).
4. The camera perspective (static CCTV, handheld, drone, etc.) and any movement.
5. A step-by-step account of the actions as they unfold over time.
6. Identify key moments or significant events in specific frames.

Pay special attention to:
- Interactions between people (conflicts, conversations, physical contact).
- Vehicles: people entering/exiting, movement, or stopping.
- Rapid movements or sudden changes in the scene.

Respond ONLY with valid JSON in this exact format (no markdown, no extra text):
{
	"summary": "Step-by-step chronological narrative. Start with 'At the beginning...', then 'Then...', 'Finally...'. Describe exactly what happens in order.",
	"key_events": [
		{
			"timestamp": "Timestamp of the frame (e.g. '1.5s', '3.0s')",
			"description": "Description of the specific event or detail in this frame"
		}
	]
}
Ensure that 'key_events' includes all significant actions and changes in the scene, ordered chronologically.
Do NOT summarize the video as a whole first; instead, describe the progression of events.`
)

func getVideoAnalysisPrompt(frameCount int, timestampInfo string) string {
	return fmt.Sprintf(_videoAnalysisPrompt, frameCount, timestampInfo)
}
