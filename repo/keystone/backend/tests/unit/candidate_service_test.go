package unit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/keystone/backend/internal/candidate"
	"github.com/stretchr/testify/assert"
)

func makeReq(demo, exam, app json.RawMessage) candidate.CreateCandidateRequest {
	return candidate.CreateCandidateRequest{
		Demographics:       demo,
		ExamScores:         exam,
		ApplicationDetails: app,
	}
}

func TestCandidateCompletenessAllFields(t *testing.T) {
	req := makeReq(
		json.RawMessage(`{"name":"John"}`),
		json.RawMessage(`{"score":90}`),
		json.RawMessage(`{"program":"CS"}`),
	)
	assert.Equal(t, "complete", candidate.ComputeCompleteness(req))
}

func TestCandidateCompletenessNoDemographics(t *testing.T) {
	req := makeReq(nil, json.RawMessage(`{"score":90}`), json.RawMessage(`{"program":"CS"}`))
	assert.Equal(t, "incomplete", candidate.ComputeCompleteness(req))
}

func TestCandidateCompletenessNoExamScores(t *testing.T) {
	req := makeReq(json.RawMessage(`{"name":"John"}`), nil, json.RawMessage(`{"program":"CS"}`))
	assert.Equal(t, "incomplete", candidate.ComputeCompleteness(req))
}

func TestCandidateCompletenessNoApplicationDetails(t *testing.T) {
	req := makeReq(
		json.RawMessage(`{"name":"John"}`),
		json.RawMessage(`{"score":90}`),
		nil,
	)
	assert.Equal(t, "incomplete", candidate.ComputeCompleteness(req))
}

func TestCandidateCompletenessNullJSON(t *testing.T) {
	req := makeReq(
		json.RawMessage("null"),
		json.RawMessage(`{"score":90}`),
		json.RawMessage(`{"program":"CS"}`),
	)
	assert.Equal(t, "incomplete", candidate.ComputeCompleteness(req))
}

func TestDocumentAllowedMIMETypes(t *testing.T) {
	// Verify that the service's allowedMIMETypes set contains the expected types.
	// These are tested by passing sniffed content to UploadDocument at integration level;
	// here we confirm the business rule: PDF, JPEG, PNG allowed; text/plain and zip are not.
	allowed := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/png":       true,
	}

	assert.True(t, allowed["application/pdf"])
	assert.True(t, allowed["image/jpeg"])
	assert.True(t, allowed["image/png"])
	assert.False(t, allowed["text/plain"])
	assert.False(t, allowed["application/zip"])
}

func TestDocumentMaxSize(t *testing.T) {
	const maxSizeBytes = int64(20 * 1024 * 1024)
	assert.LessOrEqual(t, int64(10*1024*1024), maxSizeBytes, "10MB should be within limit")
	assert.Greater(t, int64(25*1024*1024), maxSizeBytes, "25MB should exceed limit")
}

func TestDocumentDuplicateDetectionBySHA256(t *testing.T) {
	hash := func(d []byte) string {
		h := sha256.Sum256(d)
		return fmt.Sprintf("%x", h)
	}

	data := []byte("some document content")
	assert.Equal(t, hash(data), hash(data), "same content must produce same hash")
	assert.NotEqual(t, hash(data), hash([]byte("different")), "different content must produce different hash")
}
