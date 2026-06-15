package retrieval

import (
	"context"
	"errors"
	"testing"

	"uwgraph/internal/knowledge"
)

func TestFuseCandidatesUsesBothRanks(t *testing.T) {
	course := knowledge.Evidence{EntityURI: "uwgraph://courses/CS%20135", Title: "CS 135"}
	subject := knowledge.Evidence{EntityURI: "uwgraph://subjects/CS", Title: "Computer Science"}
	fullText := []knowledge.Candidate{
		{Evidence: course, Score: 9},
		{Evidence: subject, Score: 8},
	}
	vector := []knowledge.Candidate{
		{Evidence: subject, Score: 0.99},
		{Evidence: course, Score: 0.90},
	}

	result := fuseCandidates(fullText, vector, 10)
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].Scores.FullText == 0 || result[0].Scores.Vector == 0 {
		t.Fatalf("top result scores = %+v, want both retrieval scores", result[0].Scores)
	}
}

func TestFullTextQueryRemovesOperators(t *testing.T) {
	if got, want := fullTextQuery(`machine && learning:(AI)`), "machine learning AI"; got != want {
		t.Fatalf("fullTextQuery = %q, want %q", got, want)
	}
}

type fakeProvider struct{}

func (fakeProvider) Embed(context.Context, []string) ([][]float32, error) {
	return [][]float32{{1, 0}}, nil
}

type fakeRepository struct {
	fullTextQuery string
}

func (r *fakeRepository) KnowledgeIndexesReady(context.Context) error { return nil }
func (r *fakeRepository) FullTextCandidates(_ context.Context, query string, _ []string, _ string, _ int) ([]knowledge.Candidate, error) {
	r.fullTextQuery = query
	return nil, nil
}
func (r *fakeRepository) VectorCandidates(context.Context, []float32, []string, string, int) ([]knowledge.Candidate, error) {
	return nil, nil
}
func (r *fakeRepository) GetCourse(context.Context, string) (knowledge.Course, error) {
	return knowledge.Course{}, nil
}
func (r *fakeRepository) ListCourseOfferings(context.Context, string, string, int) ([]knowledge.Offering, error) {
	return nil, nil
}
func (r *fakeRepository) SearchSections(context.Context, knowledge.SectionSearchRequest) ([]knowledge.Section, error) {
	return nil, nil
}
func (r *fakeRepository) ListExams(context.Context, knowledge.ExamSearchRequest) ([]knowledge.Exam, error) {
	return nil, nil
}
func (r *fakeRepository) GetBuilding(context.Context, string) (knowledge.Building, error) {
	return knowledge.Building{}, nil
}
func (r *fakeRepository) GetTerm(context.Context, string) (knowledge.Term, error) {
	return knowledge.Term{}, nil
}

func TestSearchSanitizesFullTextQuery(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakeProvider{})
	_, err := service.Search(context.Background(), knowledge.SearchRequest{Query: `machine && learning`})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if got, want := repository.fullTextQuery, "machine learning"; got != want {
		t.Fatalf("full-text query = %q, want %q", got, want)
	}
}

func TestSearchRejectsOperatorOnlyQuery(t *testing.T) {
	service := NewService(&fakeRepository{}, fakeProvider{})
	_, err := service.Search(context.Background(), knowledge.SearchRequest{Query: `&& ! ()`})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Search error = %v, want ErrInvalidRequest", err)
	}
}

func TestSearchRejectsNegativeLimit(t *testing.T) {
	service := NewService(&fakeRepository{}, fakeProvider{})
	_, err := service.Search(context.Background(), knowledge.SearchRequest{Query: "programming", Limit: -1})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Search error = %v, want ErrInvalidRequest", err)
	}
}
