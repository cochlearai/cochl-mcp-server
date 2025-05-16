package tools

const (
	_audioCaptionDesc = `
Analyze the environmental (background) sounds in an audio file and generate a concise natural language caption.
This caption infers and summarizes the likely situation or scene.
This tool does not transcribe speech or summarize the full content,
but instead focuses on the ambient sounds to describe the environment or context in which the audio was recorded.
Example: 'A woman speaks while a television plays in the background.'
`

	_analyzeAudioDesc = `
Analyze an audio file to detect and segment environmental sounds and events over time.
This tool provides a detailed timeline, dividing the audio into temporal segments.
It identifies which sounds or events occur in each segment, along with their probability scores.
Use this tool to understand what kinds of sounds (e.g., 'Water_run', 'Laughter', 'Speech') are present at specific times in the audio.
The analysis result includes:
  - Temporal segments with start and end times
  - Tags for each segment indicating the detected sounds/events
  - Probability scores for each detected tag
Example: Detects 'Water_run' from 0-2s, 'Laughter' from 5-7s, etc.
`

	_fileAbsolutePathDesc = `
Please provide the absolute path to the file.
Avoid using URL-encoded characters.
`
)
