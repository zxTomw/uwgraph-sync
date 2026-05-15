package syncer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"uwgraph/internal/waterloo"
)

func TestGlobalAPIFailureDoesNotBlockOtherWork(t *testing.T) {
	client := newFakeClient()
	client.termsErr = errors.New("terms down")
	client.scheduledCourseIDsByTerm["1251"] = []string{"100"}
	store := newFakeStore()

	err := New(client, store, []string{"1251"}, discardLogger()).Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if store.upsertTermsCalls != 0 {
		t.Fatalf("UpsertTerms calls = %d, want 0", store.upsertTermsCalls)
	}
	if store.upsertOrgCalls != 1 || store.upsertSubjectCalls != 1 || store.upsertLocationCalls != 1 {
		t.Fatalf("global upsert calls = org:%d subject:%d location:%d, want all 1", store.upsertOrgCalls, store.upsertSubjectCalls, store.upsertLocationCalls)
	}
	if store.upsertCoursesCalls != 1 || store.upsertClassesCalls != 1 || store.upsertExamsCalls != 1 {
		t.Fatalf("term upsert calls = courses:%d classes:%d exams:%d, want all 1", store.upsertCoursesCalls, store.upsertClassesCalls, store.upsertExamsCalls)
	}
}

func TestCourseAPIFailureDoesNotBlockSchedulesExamsOrLaterTerms(t *testing.T) {
	client := newFakeClient()
	client.coursesErrByTerm["1251"] = errors.New("courses down")
	client.scheduledCourseIDsByTerm["1251"] = []string{"100"}
	client.scheduledCourseIDsByTerm["1255"] = []string{"200"}
	store := newFakeStore()

	err := New(client, store, []string{"1251", "1255"}, discardLogger()).Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if got, want := client.scheduledTerms, []string{"1251", "1255"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("scheduled terms = %v, want %v", got, want)
	}
	if got, want := client.examTerms, []string{"1251", "1255"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("exam terms = %v, want %v", got, want)
	}
	if store.upsertCoursesCalls != 1 {
		t.Fatalf("UpsertCourses calls = %d, want 1 for only the successful term", store.upsertCoursesCalls)
	}
	if store.upsertClassesCalls != 2 || store.upsertExamsCalls != 2 {
		t.Fatalf("classes/exams calls = %d/%d, want 2/2", store.upsertClassesCalls, store.upsertExamsCalls)
	}
}

func TestScheduledCourseIDFailureSkipsOnlyClassesForTerm(t *testing.T) {
	client := newFakeClient()
	client.scheduledErrByTerm["1251"] = errors.New("schedule list down")
	store := newFakeStore()

	err := New(client, store, []string{"1251"}, discardLogger()).Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if len(client.classCalls) != 0 {
		t.Fatalf("class calls = %v, want none", client.classCalls)
	}
	if store.upsertCoursesCalls != 1 || store.upsertClassesCalls != 0 || store.upsertExamsCalls != 1 {
		t.Fatalf("upsert calls = courses:%d classes:%d exams:%d, want 1/0/1", store.upsertCoursesCalls, store.upsertClassesCalls, store.upsertExamsCalls)
	}
}

func TestOneClassAPIFailureDoesNotBlockOtherCourseIDs(t *testing.T) {
	client := newFakeClient()
	client.scheduledCourseIDsByTerm["1251"] = []string{"100", "200"}
	client.classesErrByKey["1251|100"] = errors.New("class down")
	store := newFakeStore()

	err := New(client, store, []string{"1251"}, discardLogger()).Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if got, want := client.classCalls, []string{"1251|100", "1251|200"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("class calls = %v, want %v", got, want)
	}
	if store.upsertClassesCalls != 1 {
		t.Fatalf("UpsertClasses calls = %d, want 1", store.upsertClassesCalls)
	}
}

func TestExamAPIFailureDoesNotReturnError(t *testing.T) {
	client := newFakeClient()
	client.examsErrByTerm["1251"] = errors.New("exams down")
	store := newFakeStore()

	err := New(client, store, []string{"1251"}, discardLogger()).Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	if store.upsertExamsCalls != 0 {
		t.Fatalf("UpsertExams calls = %d, want 0", store.upsertExamsCalls)
	}
}

func TestStoreFailureStillReturnsError(t *testing.T) {
	client := newFakeClient()
	store := newFakeStore()
	store.upsertCoursesErr = errors.New("neo4j write failed")

	err := New(client, store, []string{"1251"}, discardLogger()).Sync(context.Background())
	if !errors.Is(err, store.upsertCoursesErr) {
		t.Fatalf("Sync error = %v, want %v", err, store.upsertCoursesErr)
	}
	if len(client.scheduledTerms) != 0 || len(client.examTerms) != 0 {
		t.Fatalf("continued after store failure: scheduled=%v exams=%v", client.scheduledTerms, client.examTerms)
	}
}

type fakeClient struct {
	termsErr     error
	orgsErr      error
	subjectsErr  error
	locationsErr error

	coursesErrByTerm         map[string]error
	scheduledErrByTerm       map[string]error
	classesErrByKey          map[string]error
	examsErrByTerm           map[string]error
	scheduledCourseIDsByTerm map[string][]string

	scheduledTerms []string
	classCalls     []string
	examTerms      []string
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		coursesErrByTerm:         make(map[string]error),
		scheduledErrByTerm:       make(map[string]error),
		classesErrByKey:          make(map[string]error),
		examsErrByTerm:           make(map[string]error),
		scheduledCourseIDsByTerm: make(map[string][]string),
	}
}

func (c *fakeClient) Terms(context.Context) ([]waterloo.Term, error) {
	if c.termsErr != nil {
		return nil, c.termsErr
	}
	return []waterloo.Term{{TermCode: "1251"}}, nil
}

func (c *fakeClient) Subjects(context.Context) ([]waterloo.Subject, error) {
	if c.subjectsErr != nil {
		return nil, c.subjectsErr
	}
	return []waterloo.Subject{{Code: "CS"}}, nil
}

func (c *fakeClient) AcademicOrganizations(context.Context) ([]waterloo.AcademicOrganization, error) {
	if c.orgsErr != nil {
		return nil, c.orgsErr
	}
	return []waterloo.AcademicOrganization{{Code: "CS"}}, nil
}

func (c *fakeClient) Locations(context.Context) ([]waterloo.Location, error) {
	if c.locationsErr != nil {
		return nil, c.locationsErr
	}
	return []waterloo.Location{{BuildingCode: "DC"}}, nil
}

func (c *fakeClient) Courses(_ context.Context, termCode string) ([]waterloo.Course, error) {
	if err := c.coursesErrByTerm[termCode]; err != nil {
		return nil, err
	}
	return []waterloo.Course{{TermCode: termCode, CourseID: "course-" + termCode}}, nil
}

func (c *fakeClient) ScheduledCourseIDs(_ context.Context, termCode string) ([]string, error) {
	c.scheduledTerms = append(c.scheduledTerms, termCode)
	if err := c.scheduledErrByTerm[termCode]; err != nil {
		return nil, err
	}
	if ids, ok := c.scheduledCourseIDsByTerm[termCode]; ok {
		return ids, nil
	}
	return nil, nil
}

func (c *fakeClient) Classes(_ context.Context, termCode, courseID string) ([]waterloo.Class, error) {
	key := termCourseKey(termCode, courseID)
	c.classCalls = append(c.classCalls, key)
	if err := c.classesErrByKey[key]; err != nil {
		return nil, err
	}
	return []waterloo.Class{{TermCode: termCode, CourseID: courseID}}, nil
}

func (c *fakeClient) Exams(_ context.Context, termCode string) ([]waterloo.Exam, error) {
	c.examTerms = append(c.examTerms, termCode)
	if err := c.examsErrByTerm[termCode]; err != nil {
		return nil, err
	}
	return []waterloo.Exam{{TermCode: termCode, ExamDisplayName: "Exam " + termCode}}, nil
}

type fakeStore struct {
	ensureSchemaErr error

	upsertTermsErr     error
	upsertOrgsErr      error
	upsertSubjectsErr  error
	upsertLocationsErr error
	upsertCoursesErr   error
	upsertClassesErr   error
	upsertExamsErr     error

	upsertTermsCalls    int
	upsertOrgCalls      int
	upsertSubjectCalls  int
	upsertLocationCalls int
	upsertCoursesCalls  int
	upsertClassesCalls  int
	upsertExamsCalls    int
}

func newFakeStore() *fakeStore {
	return &fakeStore{}
}

func (s *fakeStore) EnsureSchema(context.Context) error {
	return s.ensureSchemaErr
}

func (s *fakeStore) UpsertTerms(context.Context, []waterloo.Term) (int, error) {
	s.upsertTermsCalls++
	return 1, s.upsertTermsErr
}

func (s *fakeStore) UpsertSubjects(context.Context, []waterloo.Subject) (int, error) {
	s.upsertSubjectCalls++
	return 1, s.upsertSubjectsErr
}

func (s *fakeStore) UpsertAcademicOrganizations(context.Context, []waterloo.AcademicOrganization) (int, error) {
	s.upsertOrgCalls++
	return 1, s.upsertOrgsErr
}

func (s *fakeStore) UpsertLocations(context.Context, []waterloo.Location) (int, error) {
	s.upsertLocationCalls++
	return 1, s.upsertLocationsErr
}

func (s *fakeStore) UpsertCourses(context.Context, []waterloo.Course) (int, error) {
	s.upsertCoursesCalls++
	return 1, s.upsertCoursesErr
}

func (s *fakeStore) UpsertClasses(context.Context, []waterloo.Class) (int, int, error) {
	s.upsertClassesCalls++
	return 1, 0, s.upsertClassesErr
}

func (s *fakeStore) UpsertExams(context.Context, []waterloo.Exam) (int, error) {
	s.upsertExamsCalls++
	return 1, s.upsertExamsErr
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func termCourseKey(termCode, courseID string) string {
	return termCode + "|" + courseID
}
