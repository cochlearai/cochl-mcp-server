package client

import (
	"crypto/rand"
	"encoding/hex"
)

var mockIsDone bool = false

func SetMockSenseIsDone(isDone bool) {
	mockIsDone = isDone
}

func randomHex16() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// Mock implementation for Sense interface

type MockSense struct{}

func NewMockSense() *MockSense {
	return &MockSense{}
}

func (m *MockSense) CreateSession(fileName, contentType string, duration float64, fileSize int) (*RespCreateSession, error) {
	return &RespCreateSession{
		SessionID:     randomHex16(),
		ChunkSequence: 0,
		WindowSize:    2,
		WindowHop:     1,
	}, nil
}

func (m *MockSense) UploadChunk(sessionID string, chunkSequence int, chunk []byte) (*RespUploadChunk, error) {
	return &RespUploadChunk{
		ChunkSequence: chunkSequence + 1,
		SessionID:     sessionID,
	}, nil
}

func (m *MockSense) GetInferenceResult(sessionID string) (*RespInferenceResult, error) {
	if !mockIsDone {
		return &RespInferenceResult{
			Data:  []InferenceResult{},
			State: "in-progress",
		}, nil
	}

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

func NewMockCaption() *MockCaption {
	return &MockCaption{}
}

func (m *MockCaption) Inference(contentType, filePath string) (*RespCaptionInference, error) {
	return &RespCaptionInference{
		Caption: "This is a mock caption.",
	}, nil
}
