//go:build integration

package neo4jstore

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"uwgraph/internal/knowledge"
	"uwgraph/internal/waterloo"
)

func TestStoreIntegrationKnowledgeQueries(t *testing.T) {
	uri := os.Getenv("NEO4J_URI")
	user := os.Getenv("NEO4J_USERNAME")
	password := os.Getenv("NEO4J_PASSWORD")
	if uri == "" || user == "" || password == "" {
		t.Skip("NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD are required")
	}
	database := os.Getenv("NEO4J_DATABASE")
	if database == "" {
		database = "neo4j"
	}

	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		t.Fatalf("NewDriverWithContext: %v", err)
	}
	defer driver.Close(ctx)

	store := New(driver, database, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := store.write(ctx, "MATCH (node) DETACH DELETE node", nil); err != nil {
		t.Fatalf("clear database: %v", err)
	}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	if err := store.EnsureVectorIndex(ctx, 3); err != nil {
		t.Fatalf("EnsureVectorIndex: %v", err)
	}
	if _, err := store.UpsertTerms(ctx, []waterloo.Term{{TermCode: "9999", Name: "Integration Test Term"}}); err != nil {
		t.Fatalf("UpsertTerms: %v", err)
	}
	if _, err := store.UpsertAcademicOrganizations(ctx, []waterloo.AcademicOrganization{{
		Code: "ENG", Name: "Engineering", Description: "Integration engineering organization",
	}}); err != nil {
		t.Fatalf("UpsertAcademicOrganizations: %v", err)
	}
	if _, err := store.UpsertSubjects(ctx, []waterloo.Subject{{
		Code: "CS", Name: "Computer Science", Description: "Integration computing subject",
		AssociatedAcademicOrgCode: "ENG",
	}}); err != nil {
		t.Fatalf("UpsertSubjects: %v", err)
	}
	if _, err := store.UpsertLocations(ctx, []waterloo.Location{{
		BuildingCode: "DC", BuildingName: "Davis Centre",
		AlternateBuildingNames: []string{"William G. Davis Computer Research Centre"},
	}}); err != nil {
		t.Fatalf("UpsertLocations: %v", err)
	}
	course := waterloo.Course{
		CourseID: "000999", CourseOfferNumber: 1, TermCode: "9999",
		TermName: "Integration Test Term", SubjectCode: "CS", CatalogNumber: "999",
		Title: "Integration Knowledge Systems", Description: "Searchable integration course",
		RequirementsDescription:   "Prerequisite text is source data only.",
		AssociatedAcademicOrgCode: "ENG",
	}
	if _, err := store.UpsertCourses(ctx, []waterloo.Course{course}); err != nil {
		t.Fatalf("UpsertCourses: %v", err)
	}
	if _, _, err := store.UpsertClasses(ctx, []waterloo.Class{{
		CourseID: "000999", CourseOfferNumber: 1, TermCode: "9999",
		SessionCode: "1", ClassSection: 1, ClassNumber: 1234,
		CourseComponent: "LEC", MaxEnrollmentCapacity: 100, EnrolledStudents: 50,
		ScheduleData: []waterloo.ClassSchedule{{
			CourseID: "000999", CourseOfferNumber: 1, TermCode: "9999",
			SessionCode: "1", ClassSection: 1, ClassMeetingNumber: 1,
			ClassMeetingDayPatternCode: "MWF", ClassMeetingStartTime: "10:00",
			ClassMeetingEndTime: "10:50", LocationName: "DC 1350",
		}},
		InstructorData: []waterloo.ClassInstructor{{
			InstructorUniqueIdentifier: "integration-instructor",
			InstructorFirstName:        "Ada", InstructorLastName: "Lovelace",
			InstructorRoleCode: "PI", ClassMeetingNumber: 1,
		}},
	}}); err != nil {
		t.Fatalf("UpsertClasses: %v", err)
	}

	waitForKnowledgeIndexes(t, ctx, store)
	pending, err := store.PendingDocuments(ctx, "integration-model", 20)
	if err != nil {
		t.Fatalf("PendingDocuments: %v", err)
	}
	var courseDocument knowledge.PendingDocument
	for _, document := range pending {
		if document.DocumentKey == "course:CS 999" {
			courseDocument = document
			break
		}
	}
	if courseDocument.DocumentKey == "" {
		t.Fatal("course knowledge document was not projected")
	}
	if _, err := store.ApplyEmbeddings(ctx, []knowledge.EmbeddingUpdate{{
		DocumentKey: courseDocument.DocumentKey,
		ContentHash: courseDocument.ContentHash,
		Model:       "integration-model",
		Embedding:   []float32{1, 0, 0},
		EmbeddedAt:  time.Now(),
	}}); err != nil {
		t.Fatalf("ApplyEmbeddings: %v", err)
	}

	fullText := waitForCandidates(t, func() ([]knowledge.Candidate, error) {
		return store.FullTextCandidates(ctx, "Integration", []string{knowledge.KindCourse}, "9999", 10)
	})
	if fullText[0].Evidence.EntityURI != "uwgraph://courses/CS%20999" {
		t.Fatalf("unexpected full-text result: %#v", fullText[0].Evidence)
	}
	vector := waitForCandidates(t, func() ([]knowledge.Candidate, error) {
		return store.VectorCandidates(ctx, []float32{1, 0, 0}, []string{knowledge.KindCourse}, "9999", 10)
	})
	if vector[0].Evidence.EntityKey != "CS 999" {
		t.Fatalf("unexpected vector result: %#v", vector[0].Evidence)
	}

	gotCourse, err := store.GetCourse(ctx, "CS 999")
	if err != nil || gotCourse.Citation.SourceEndpoint == "" {
		t.Fatalf("GetCourse: course=%#v err=%v", gotCourse, err)
	}
	sections, err := store.SearchSections(ctx, knowledge.SectionSearchRequest{
		CourseCode: "CS 999", TermCode: "9999", Days: []string{"M"}, HasSeats: true, Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchSections: %v", err)
	}
	if len(sections) != 1 || len(sections[0].Instructors) != 1 {
		t.Fatalf("unexpected sections: %#v", sections)
	}

	course.Description = "Changed content invalidates the stored embedding"
	if _, err := store.UpsertCourses(ctx, []waterloo.Course{course}); err != nil {
		t.Fatalf("update course: %v", err)
	}
	stale, err := store.VectorCandidates(ctx, []float32{1, 0, 0}, []string{knowledge.KindCourse}, "9999", 10)
	if err != nil {
		t.Fatalf("VectorCandidates after update: %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("stale embedding remained searchable: %#v", stale)
	}
}

func waitForKnowledgeIndexes(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := store.KnowledgeIndexesReady(ctx); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("knowledge indexes did not become ready")
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func waitForCandidates(t *testing.T, query func() ([]knowledge.Candidate, error)) []knowledge.Candidate {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		candidates, err := query()
		if err == nil && len(candidates) > 0 {
			return candidates
		}
		if time.Now().After(deadline) {
			t.Fatalf("knowledge query did not return candidates: %v", err)
		}
		time.Sleep(250 * time.Millisecond)
	}
}
