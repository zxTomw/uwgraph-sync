package knowledgeapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"uwgraph/internal/knowledge"
)

type fakeService struct{}

func (fakeService) Search(_ context.Context, request knowledge.SearchRequest) (knowledge.SearchResult, error) {
	return knowledge.SearchResult{Query: request.Query, Retrieval: "hybrid_rrf"}, nil
}
func (fakeService) GetCourse(context.Context, string) (knowledge.Course, error) {
	return knowledge.Course{CourseCode: "CS 135"}, nil
}
func (fakeService) ListCourseOfferings(context.Context, string, string, int) ([]knowledge.Offering, error) {
	return nil, nil
}
func (fakeService) SearchSections(context.Context, knowledge.SectionSearchRequest) ([]knowledge.Section, error) {
	return nil, nil
}
func (fakeService) ListExams(context.Context, knowledge.ExamSearchRequest) ([]knowledge.Exam, error) {
	return nil, nil
}
func (fakeService) GetBuilding(context.Context, string) (knowledge.Building, error) {
	return knowledge.Building{BuildingCode: "DC"}, nil
}
func (fakeService) GetTerm(context.Context, string) (knowledge.Term, error) {
	return knowledge.Term{TermCode: "1265"}, nil
}
func (fakeService) Ready(context.Context) error { return nil }

func TestRESTRequiresBearerToken(t *testing.T) {
	server := New(
		fakeService{},
		"secret",
		[]string{"https://agent.example"},
		time.Second,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/search", strings.NewReader(`{"query":"programming"}`))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRESTSearchUsesAuthenticatedService(t *testing.T) {
	server := New(
		fakeService{},
		"secret",
		nil,
		time.Second,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/search", strings.NewReader(`{"query":"programming"}`))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"retrieval":"hybrid_rrf"`) {
		t.Fatalf("response = %s, want retrieval mode", response.Body.String())
	}
}

func TestRESTSearchRejectsTrailingJSON(t *testing.T) {
	server := New(
		fakeService{},
		"secret",
		nil,
		time.Second,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/search", strings.NewReader(`{"query":"programming"} {}`))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestMCPRejectsUnknownOriginBeforeProtocolHandling(t *testing.T) {
	server := New(
		fakeService{},
		"secret",
		[]string{"https://agent.example"},
		time.Second,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}
