package retrieval

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"uwgraph/internal/embedding"
	"uwgraph/internal/knowledge"
)

var (
	ErrInvalidRequest = errors.New("invalid knowledge request")
	ErrUnavailable    = errors.New("knowledge retrieval unavailable")
)

const (
	defaultLimit   = 10
	maxLimit       = 50
	rankConstant   = 60.0
	readinessCache = 30 * time.Second
)

type Repository interface {
	KnowledgeIndexesReady(context.Context) error
	FullTextCandidates(context.Context, string, []string, string, int) ([]knowledge.Candidate, error)
	VectorCandidates(context.Context, []float32, []string, string, int) ([]knowledge.Candidate, error)
	GetCourse(context.Context, string) (knowledge.Course, error)
	ListCourseOfferings(context.Context, string, string, int) ([]knowledge.Offering, error)
	SearchSections(context.Context, knowledge.SectionSearchRequest) ([]knowledge.Section, error)
	ListExams(context.Context, knowledge.ExamSearchRequest) ([]knowledge.Exam, error)
	GetBuilding(context.Context, string) (knowledge.Building, error)
	GetTerm(context.Context, string) (knowledge.Term, error)
}

type Service struct {
	repository Repository
	provider   embedding.Provider

	readyMu  sync.Mutex
	readyAt  time.Time
	readyErr error
	now      func() time.Time
}

func NewService(repository Repository, provider embedding.Provider) *Service {
	return &Service{
		repository: repository,
		provider:   provider,
		now:        time.Now,
	}
}

func (s *Service) Search(ctx context.Context, request knowledge.SearchRequest) (knowledge.SearchResult, error) {
	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" {
		return knowledge.SearchResult{}, fmt.Errorf("%w: query is required", ErrInvalidRequest)
	}
	var err error
	request.Limit, err = validatedLimit(request.Limit)
	if err != nil {
		return knowledge.SearchResult{}, err
	}
	if err := validateKinds(request.Kinds); err != nil {
		return knowledge.SearchResult{}, err
	}
	sanitizedQuery := fullTextQuery(request.Query)
	if sanitizedQuery == "" {
		return knowledge.SearchResult{}, fmt.Errorf("%w: query must contain searchable text", ErrInvalidRequest)
	}

	embeddings, err := s.provider.Embed(ctx, []string{request.Query})
	if err != nil {
		return knowledge.SearchResult{}, fmt.Errorf("%w: embed query: %v", ErrUnavailable, err)
	}
	if len(embeddings) != 1 {
		return knowledge.SearchResult{}, fmt.Errorf("%w: embedding provider returned %d vectors", ErrUnavailable, len(embeddings))
	}
	candidateLimit := request.Limit * 5
	if candidateLimit < 25 {
		candidateLimit = 25
	}
	fullText, err := s.repository.FullTextCandidates(
		ctx,
		sanitizedQuery,
		request.Kinds,
		strings.TrimSpace(request.TermCode),
		candidateLimit,
	)
	if err != nil {
		return knowledge.SearchResult{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	vector, err := s.repository.VectorCandidates(
		ctx,
		embeddings[0],
		request.Kinds,
		strings.TrimSpace(request.TermCode),
		candidateLimit,
	)
	if err != nil {
		return knowledge.SearchResult{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	evidence := fuseCandidates(fullText, vector, request.Limit)
	return knowledge.SearchResult{
		Query:     request.Query,
		Retrieval: "hybrid_rrf",
		Evidence:  evidence,
	}, nil
}

func (s *Service) GetCourse(ctx context.Context, courseCode string) (knowledge.Course, error) {
	if strings.TrimSpace(courseCode) == "" {
		return knowledge.Course{}, fmt.Errorf("%w: course code is required", ErrInvalidRequest)
	}
	return s.repository.GetCourse(ctx, courseCode)
}

func (s *Service) ListCourseOfferings(ctx context.Context, courseCode, termCode string, limit int) ([]knowledge.Offering, error) {
	if strings.TrimSpace(courseCode) == "" {
		return nil, fmt.Errorf("%w: course code is required", ErrInvalidRequest)
	}
	limit, err := validatedLimit(limit)
	if err != nil {
		return nil, err
	}
	return s.repository.ListCourseOfferings(ctx, courseCode, termCode, limit)
}

func (s *Service) SearchSections(ctx context.Context, request knowledge.SectionSearchRequest) ([]knowledge.Section, error) {
	if request.CourseCode == "" && request.TermCode == "" {
		return nil, fmt.Errorf("%w: courseCode or termCode is required", ErrInvalidRequest)
	}
	var err error
	request.Limit, err = validatedLimit(request.Limit)
	if err != nil {
		return nil, err
	}
	return s.repository.SearchSections(ctx, request)
}

func (s *Service) ListExams(ctx context.Context, request knowledge.ExamSearchRequest) ([]knowledge.Exam, error) {
	if strings.TrimSpace(request.TermCode) == "" {
		return nil, fmt.Errorf("%w: termCode is required", ErrInvalidRequest)
	}
	var err error
	request.Limit, err = validatedLimit(request.Limit)
	if err != nil {
		return nil, err
	}
	return s.repository.ListExams(ctx, request)
}

func (s *Service) GetBuilding(ctx context.Context, buildingCode string) (knowledge.Building, error) {
	if strings.TrimSpace(buildingCode) == "" {
		return knowledge.Building{}, fmt.Errorf("%w: building code is required", ErrInvalidRequest)
	}
	return s.repository.GetBuilding(ctx, buildingCode)
}

func (s *Service) GetTerm(ctx context.Context, termCode string) (knowledge.Term, error) {
	if strings.TrimSpace(termCode) == "" {
		return knowledge.Term{}, fmt.Errorf("%w: term code is required", ErrInvalidRequest)
	}
	return s.repository.GetTerm(ctx, termCode)
}

func (s *Service) Ready(ctx context.Context) error {
	s.readyMu.Lock()
	defer s.readyMu.Unlock()
	now := s.now()
	if !s.readyAt.IsZero() && now.Sub(s.readyAt) < readinessCache {
		return s.readyErr
	}
	err := s.repository.KnowledgeIndexesReady(ctx)
	if err == nil {
		var vectors [][]float32
		vectors, err = s.provider.Embed(ctx, []string{"uwgraph readiness probe"})
		if err == nil && len(vectors) != 1 {
			err = fmt.Errorf("embedding provider returned %d vectors", len(vectors))
		}
	}
	s.readyAt = now
	if err != nil {
		s.readyErr = fmt.Errorf("%w: %v", ErrUnavailable, err)
	} else {
		s.readyErr = nil
	}
	return s.readyErr
}

func fuseCandidates(fullText, vector []knowledge.Candidate, limit int) []knowledge.Evidence {
	type ranked struct {
		evidence knowledge.Evidence
	}
	combined := make(map[string]*ranked, len(fullText)+len(vector))
	for rank, candidate := range fullText {
		entry := combined[candidate.Evidence.EntityURI]
		if entry == nil {
			entry = &ranked{evidence: candidate.Evidence}
			combined[candidate.Evidence.EntityURI] = entry
		}
		entry.evidence.Scores.FullText = candidate.Score
		entry.evidence.Scores.Fused += 1 / (rankConstant + float64(rank+1))
	}
	for rank, candidate := range vector {
		entry := combined[candidate.Evidence.EntityURI]
		if entry == nil {
			entry = &ranked{evidence: candidate.Evidence}
			combined[candidate.Evidence.EntityURI] = entry
		}
		entry.evidence.Scores.Vector = candidate.Score
		entry.evidence.Scores.Fused += 1 / (rankConstant + float64(rank+1))
	}
	result := make([]knowledge.Evidence, 0, len(combined))
	for _, entry := range combined {
		result = append(result, entry.evidence)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Scores.Fused == result[j].Scores.Fused {
			return result[i].EntityURI < result[j].EntityURI
		}
		return result[i].Scores.Fused > result[j].Scores.Fused
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func fullTextQuery(query string) string {
	replacer := strings.NewReplacer(
		"+", " ", "-", " ", "&&", " ", "||", " ", "!", " ",
		"(", " ", ")", " ", "{", " ", "}", " ", "[", " ", "]", " ",
		"^", " ", "\"", " ", "~", " ", "*", " ", "?", " ", ":", " ",
		"\\", " ", "/", " ",
	)
	return strings.Join(strings.Fields(replacer.Replace(query)), " ")
}

func validatedLimit(limit int) (int, error) {
	if limit < 0 {
		return 0, fmt.Errorf("%w: limit must not be negative", ErrInvalidRequest)
	}
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit > maxLimit {
		return maxLimit, nil
	}
	return limit, nil
}

func validateKinds(kinds []string) error {
	for _, kind := range kinds {
		switch kind {
		case knowledge.KindCourse, knowledge.KindSubject, knowledge.KindAcademicOrganization, knowledge.KindBuilding:
		default:
			return fmt.Errorf("%w: unsupported kind %q", ErrInvalidRequest, kind)
		}
	}
	return nil
}
