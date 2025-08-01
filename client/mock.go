package client

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func randomHex16() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// Mock implementation for Sense interface
var (
	shouldCreateSessionError      bool = false
	shouldUploadChunkError        bool = false
	shouldGetInferenceResultError bool = false
)

func ResetMockSenseErrors() {
	shouldCreateSessionError = false
	shouldUploadChunkError = false
	shouldGetInferenceResultError = false
}

func SetShouldMockCreateSessionError(v bool) {
	shouldCreateSessionError = v
}

func SetShouldMockUploadChunkError(v bool) {
	shouldUploadChunkError = v
}

func SetShouldMockGetInferenceResultError(v bool) {
	shouldGetInferenceResultError = v
}

type MockSense struct{}

func NewMockSense() *MockSense {
	return &MockSense{}
}

func (m *MockSense) CreateSession(fileName, contentType string, duration float64, fileSize int) (*RespCreateSession, error) {
	if shouldCreateSessionError {
		return nil, fmt.Errorf("create session error")
	}

	return &RespCreateSession{
		SessionID:     randomHex16(),
		ChunkSequence: 0,
		WindowSize:    2,
		WindowHop:     1,
	}, nil
}

func (m *MockSense) UploadChunk(sessionID string, chunkSequence int, chunk []byte) (*RespUploadChunk, error) {
	if shouldUploadChunkError {
		return nil, fmt.Errorf("upload chunk error")
	}

	return &RespUploadChunk{
		ChunkSequence: chunkSequence + 1,
		SessionID:     sessionID,
	}, nil
}

func (m *MockSense) GetInferenceResult(sessionID string) (*RespInferenceResult, error) {
	if shouldGetInferenceResultError {
		return nil, fmt.Errorf("get inference result error")
	}

	return &RespInferenceResult{
		Data: []InferenceResult{
			{
				StartTime: 0,
				EndTime:   2,
				Tags: []Tags{
					{Probability: 0.1, Name: "mock-tag-1"},
					{Probability: 0.2, Name: "mock-tag-2"},
				},
			},
			{
				StartTime: 2,
				EndTime:   4,
				Tags: []Tags{
					{Probability: 0.3, Name: "mock-tag-3"},
					{Probability: 0.4, Name: "mock-tag-4"},
					{Probability: 0.5, Name: "mock-tag-5"},
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
var (
	shouldCaptionError bool = false
)

func ResetMockCaptionErrors() {
	shouldCaptionError = false
}

func SetShouldMockCaptionError(v bool) {
	shouldCaptionError = v
}

type MockCaption struct{}

func NewMockCaption() *MockCaption {
	return &MockCaption{}
}

func (m *MockCaption) Inference(contentType, filePath string) (*RespCaptionInference, error) {
	if shouldCaptionError {
		return nil, fmt.Errorf("caption inference error")
	}

	return &RespCaptionInference{
		Caption: "This is a mock caption.",
	}, nil
}
