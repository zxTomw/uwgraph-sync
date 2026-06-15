package knowledgeapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"uwgraph/internal/knowledge"
	"uwgraph/internal/retrieval"
)

type Service interface {
	Search(context.Context, knowledge.SearchRequest) (knowledge.SearchResult, error)
	GetCourse(context.Context, string) (knowledge.Course, error)
	ListCourseOfferings(context.Context, string, string, int) ([]knowledge.Offering, error)
	SearchSections(context.Context, knowledge.SectionSearchRequest) ([]knowledge.Section, error)
	ListExams(context.Context, knowledge.ExamSearchRequest) ([]knowledge.Exam, error)
	GetBuilding(context.Context, string) (knowledge.Building, error)
	GetTerm(context.Context, string) (knowledge.Term, error)
	Ready(context.Context) error
}

type Server struct {
	service      Service
	apiKey       string
	origins      map[string]struct{}
	queryTimeout time.Duration
	logger       *slog.Logger
}

func New(service Service, apiKey string, allowedOrigins []string, queryTimeout time.Duration, logger *slog.Logger) *Server {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[strings.TrimSpace(origin)] = struct{}{}
	}
	return &Server{
		service:      service,
		apiKey:       apiKey,
		origins:      origins,
		queryTimeout: queryTimeout,
		logger:       logger,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.Handle("POST /v1/search", s.authenticate(http.HandlerFunc(s.handleSearch)))
	mux.Handle("GET /v1/courses/{courseCode}", s.authenticate(http.HandlerFunc(s.handleGetCourse)))
	mux.Handle("GET /v1/courses/{courseCode}/offerings", s.authenticate(http.HandlerFunc(s.handleListOfferings)))
	mux.Handle("POST /v1/sections/search", s.authenticate(http.HandlerFunc(s.handleSearchSections)))
	mux.Handle("GET /v1/exams", s.authenticate(http.HandlerFunc(s.handleListExams)))
	mux.Handle("GET /v1/buildings/{buildingCode}", s.authenticate(http.HandlerFunc(s.handleGetBuilding)))

	mcpServer := s.newMCPServer()
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpServer },
		&mcp.StreamableHTTPOptions{
			Stateless:      true,
			SessionTimeout: 5 * time.Minute,
			Logger:         s.logger,
		},
	)
	mux.Handle("/mcp", s.authenticate(s.validateOrigin(mcpHandler)))
	return s.logRequests(mux)
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	if err := s.service.Ready(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var request knowledge.SearchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	result, err := s.service.Search(ctx, request)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetCourse(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	course, err := s.service.GetCourse(ctx, r.PathValue("courseCode"))
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, course)
}

func (s *Server) handleListOfferings(w http.ResponseWriter, r *http.Request) {
	limit, err := queryLimit(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	offerings, err := s.service.ListCourseOfferings(
		ctx,
		r.PathValue("courseCode"),
		r.URL.Query().Get("termCode"),
		limit,
	)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"offerings": offerings})
}

func (s *Server) handleSearchSections(w http.ResponseWriter, r *http.Request) {
	var request knowledge.SectionSearchRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	sections, err := s.service.SearchSections(ctx, request)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sections": sections})
}

func (s *Server) handleListExams(w http.ResponseWriter, r *http.Request) {
	limit, err := queryLimit(r)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	exams, err := s.service.ListExams(ctx, knowledge.ExamSearchRequest{
		TermCode: r.URL.Query().Get("termCode"),
		Sections: r.URL.Query().Get("sections"),
		Limit:    limit,
	})
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"exams": exams})
}

func (s *Server) handleGetBuilding(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	building, err := s.service.GetBuilding(ctx, r.PathValue("buildingCode"))
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, building)
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) || !secureEqual(strings.TrimPrefix(header, prefix), s.apiKey) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) validateOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if _, ok := s.origins[origin]; !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "origin not allowed"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		s.logger.InfoContext(
			r.Context(),
			"knowledge request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(started),
		)
	})
}

func secureEqual(actual, expected string) bool {
	if len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("%w: invalid JSON body", retrieval.ErrInvalidRequest)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: multiple JSON values", retrieval.ErrInvalidRequest)
	}
	return nil
}

func queryLimit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: limit must be an integer", retrieval.ErrInvalidRequest)
	}
	if limit <= 0 {
		return 0, fmt.Errorf("%w: limit must be greater than zero", retrieval.ErrInvalidRequest)
	}
	return limit, nil
}

func writeAPIError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, retrieval.ErrInvalidRequest):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, knowledge.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, retrieval.ErrUnavailable), errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "knowledge retrieval unavailable"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type courseInput struct {
	CourseCode string `json:"courseCode" jsonschema:"University course code, for example CS 135"`
}

type offeringsInput struct {
	CourseCode string `json:"courseCode" jsonschema:"University course code, for example CS 135"`
	TermCode   string `json:"termCode,omitempty" jsonschema:"Optional Waterloo term code"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Maximum number of offerings"`
}

type buildingInput struct {
	BuildingCode string `json:"buildingCode" jsonschema:"Waterloo building code, for example DC"`
}

type offeringsOutput struct {
	Offerings []knowledge.Offering `json:"offerings"`
}

type sectionsOutput struct {
	Sections []knowledge.Section `json:"sections"`
}

type examsOutput struct {
	Exams []knowledge.Exam `json:"exams"`
}

func (s *Server) newMCPServer() *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "uwgraph-knowledge", Version: "v1"},
		&mcp.ServerOptions{HasResources: true},
	)
	annotations := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPointer(false)}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_catalog",
		Description: "Hybrid semantic and full-text search over Waterloo courses, subjects, organizations, and buildings.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input knowledge.SearchRequest) (*mcp.CallToolResult, knowledge.SearchResult, error) {
		ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
		defer cancel()
		result, err := s.service.Search(ctx, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_course",
		Description: "Get authoritative course details and cited requirements text.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input courseInput) (*mcp.CallToolResult, knowledge.Course, error) {
		ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
		defer cancel()
		result, err := s.service.GetCourse(ctx, input.CourseCode)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_course_offerings",
		Description: "List term-specific offerings for a course.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input offeringsInput) (*mcp.CallToolResult, offeringsOutput, error) {
		ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
		defer cancel()
		result, err := s.service.ListCourseOfferings(ctx, input.CourseCode, input.TermCode, input.Limit)
		return nil, offeringsOutput{Offerings: result}, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_sections",
		Description: "Find class sections by course, term, days, times, and seat availability.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input knowledge.SectionSearchRequest) (*mcp.CallToolResult, sectionsOutput, error) {
		ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
		defer cancel()
		result, err := s.service.SearchSections(ctx, input)
		return nil, sectionsOutput{Sections: result}, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_exams",
		Description: "List cited exams by term and optional section text.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input knowledge.ExamSearchRequest) (*mcp.CallToolResult, examsOutput, error) {
		ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
		defer cancel()
		result, err := s.service.ListExams(ctx, input)
		return nil, examsOutput{Exams: result}, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_building",
		Description: "Get a Waterloo building and its location metadata.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input buildingInput) (*mcp.CallToolResult, knowledge.Building, error) {
		ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
		defer cancel()
		result, err := s.service.GetBuilding(ctx, input.BuildingCode)
		return nil, result, err
	})

	resourceHandler := func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
		defer cancel()
		parsed, err := url.Parse(request.Params.URI)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(request.Params.URI)
		}
		key, err := url.PathUnescape(strings.TrimPrefix(parsed.Path, "/"))
		if err != nil {
			return nil, mcp.ResourceNotFoundError(request.Params.URI)
		}
		var value any
		switch parsed.Host {
		case "courses":
			value, err = s.service.GetCourse(ctx, key)
		case "terms":
			value, err = s.service.GetTerm(ctx, key)
		default:
			return nil, mcp.ResourceNotFoundError(request.Params.URI)
		}
		if errors.Is(err, knowledge.ErrNotFound) {
			return nil, mcp.ResourceNotFoundError(request.Params.URI)
		}
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal resource: %w", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			}},
		}, nil
	}
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "course",
		Title:       "Waterloo course",
		Description: "Cited course record by course code.",
		MIMEType:    "application/json",
		URITemplate: "uwgraph://courses/{courseCode}",
	}, resourceHandler)
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "term",
		Title:       "Waterloo term",
		Description: "Cited academic term record by term code.",
		MIMEType:    "application/json",
		URITemplate: "uwgraph://terms/{termCode}",
	}, resourceHandler)
	return server
}

func boolPointer(value bool) *bool {
	return &value
}
