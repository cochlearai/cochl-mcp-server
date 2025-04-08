package msg

type RespUploadChunk struct {
	ChunkSequence int    `json:"chunk_sequence"`
	SessionID     string `json:"session_id"`
}

type RespCreateSession struct {
	SessionID     string `json:"session_id"`
	ChunkSequence int    `json:"chunk_sequence"`
	WindowSize    int    `json:"window_size"`
	WindowHop     int    `json:"window_hop"`
}

type InferenceResult struct {
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
	Tags      []Tags `json:"tags"`
}

type Tags struct {
	Probability float64 `json:"probability"`
	Name        string  `json:"name"`
}

type RespInferenceResult struct {
	Data  []InferenceResult `json:"data"`
	State string            `json:"state"`
}
