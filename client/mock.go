package client

// Mock implementation for Sense interface

type MockSense struct{}

func (m *MockSense) CreateSession(fileName, contentType string, duration float64, fileSize int) (*RespCreateSession, error) {
	return &RespCreateSession{
		SessionID:     "mock-session-id",
		ChunkSequence: 0,
		WindowSize:    2,
		WindowHop:     1,
	}, nil
}

func (m *MockSense) UploadChunk(sessionID string, chunkSequence int, chunk []byte) (*RespUploadChunk, error) {
	return &RespUploadChunk{
		ChunkSequence: chunkSequence,
		SessionID:     sessionID,
	}, nil
}

func (m *MockSense) GetInferenceResult(sessionID string) (*RespInferenceResult, error) {
	return &RespInferenceResult{
		Data: []InferenceResult{
			{
				StartTime: 0,
				EndTime:   2,
				Tags: []Tags{
					{Probability: 0.9, Name: "mock-tag"},
				},
			},
		},
		State: "done",
	}, nil
}

func (m *MockSense) DeleteSession(sessionID string) error {
	return nil
}

// Mock implementation for Caption interface

type MockCaption struct{}

func (m *MockCaption) Inference(contentType, filePath string) (*RespCaptionInference, error) {
	return &RespCaptionInference{
		Caption: "This is a mock caption.",
	}, nil
}
