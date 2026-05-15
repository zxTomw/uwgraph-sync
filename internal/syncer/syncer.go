package syncer

import (
	"context"
	"log/slog"

	"uwgraph/internal/waterloo"
)

type WaterlooClient interface {
	Terms(context.Context) ([]waterloo.Term, error)
	Subjects(context.Context) ([]waterloo.Subject, error)
	AcademicOrganizations(context.Context) ([]waterloo.AcademicOrganization, error)
	Locations(context.Context) ([]waterloo.Location, error)
	Courses(context.Context, string) ([]waterloo.Course, error)
	ScheduledCourseIDs(context.Context, string) ([]string, error)
	Classes(context.Context, string, string) ([]waterloo.Class, error)
	Exams(context.Context, string) ([]waterloo.Exam, error)
}

type Store interface {
	EnsureSchema(context.Context) error
	UpsertTerms(context.Context, []waterloo.Term) (int, error)
	UpsertSubjects(context.Context, []waterloo.Subject) (int, error)
	UpsertAcademicOrganizations(context.Context, []waterloo.AcademicOrganization) (int, error)
	UpsertLocations(context.Context, []waterloo.Location) (int, error)
	UpsertCourses(context.Context, []waterloo.Course) (int, error)
	UpsertClasses(context.Context, []waterloo.Class) (int, int, error)
	UpsertExams(context.Context, []waterloo.Exam) (int, error)
}

type Service struct {
	client    WaterlooClient
	store     Store
	termCodes []string
	logger    *slog.Logger
}

func New(client WaterlooClient, store Store, termCodes []string, logger *slog.Logger) *Service {
	return &Service{client: client, store: store, termCodes: termCodes, logger: logger}
}

func (s *Service) Sync(ctx context.Context) error {
	s.logger.Info("sync started")
	if err := s.store.EnsureSchema(ctx); err != nil {
		return err
	}

	terms, err := s.client.Terms(ctx)
	if err != nil {
		return err
	}
	termCount, err := s.store.UpsertTerms(ctx, terms)
	if err != nil {
		return err
	}
	s.logger.Info("synced terms", "fetched", len(terms), "written", termCount)

	orgs, err := s.client.AcademicOrganizations(ctx)
	if err != nil {
		return err
	}
	orgCount, err := s.store.UpsertAcademicOrganizations(ctx, orgs)
	if err != nil {
		return err
	}
	s.logger.Info("synced academic organizations", "fetched", len(orgs), "written", orgCount)

	subjects, err := s.client.Subjects(ctx)
	if err != nil {
		return err
	}
	subjectCount, err := s.store.UpsertSubjects(ctx, subjects)
	if err != nil {
		return err
	}
	s.logger.Info("synced subjects", "fetched", len(subjects), "written", subjectCount)

	locations, err := s.client.Locations(ctx)
	if err != nil {
		return err
	}
	locationCount, err := s.store.UpsertLocations(ctx, locations)
	if err != nil {
		return err
	}
	s.logger.Info("synced locations", "fetched", len(locations), "written", locationCount)

	for _, termCode := range s.termCodes {
		if err := s.syncTerm(ctx, termCode); err != nil {
			return err
		}
	}

	s.logger.Info("sync finished")
	return nil
}

func (s *Service) syncTerm(ctx context.Context, termCode string) error {
	courses, err := s.client.Courses(ctx, termCode)
	if err != nil {
		return err
	}
	courseCount, err := s.store.UpsertCourses(ctx, courses)
	if err != nil {
		return err
	}
	s.logger.Info("synced courses", "termCode", termCode, "fetched", len(courses), "written", courseCount)

	courseIDs, err := s.client.ScheduledCourseIDs(ctx, termCode)
	if err != nil {
		return err
	}
	s.logger.Info("fetched scheduled course ids", "termCode", termCode, "count", len(courseIDs))

	for _, courseID := range courseIDs {
		classes, err := s.client.Classes(ctx, termCode, courseID)
		if err != nil {
			return err
		}
		sectionCount, meetingCount, err := s.store.UpsertClasses(ctx, classes)
		if err != nil {
			return err
		}
		s.logger.Info("synced classes", "termCode", termCode, "courseId", courseID, "fetched", len(classes), "sections", sectionCount, "meetings", meetingCount)
	}

	exams, err := s.client.Exams(ctx, termCode)
	if err != nil {
		return err
	}
	examCount, err := s.store.UpsertExams(ctx, exams)
	if err != nil {
		return err
	}
	s.logger.Info("synced exams", "termCode", termCode, "fetched", len(exams), "written", examCount)
	return nil
}
