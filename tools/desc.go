package tools

const (
	_analyzeAudioDescWithCaption = `
Analyze an audio file to detect and segment environmental sounds and events over time.
This tool also generates a concise natural language caption.
This tool provides a detailed timeline, dividing the audio into temporal segments.
It identifies which sounds or events occur in each segment, along with their probability scores.
Use this tool to understand what kinds of sounds (e.g., 'Water_run', 'Laughter', 'Speech') are present at specific times in the audio.
The analysis result includes:
  - Temporal segments with start and end times
  - Tags for each segment indicating the detected sounds/events
  - Probability scores for each detected tag
  - Caption summarizing the likely situation or scene
Example: Detects 'Water_run' from 0-2s, 'Laughter' from 5-7s, etc.
Example: 'A woman speaks while a television plays in the background.'
`
)
