package knowledge

import (
	"testing"

	"uwgraph/internal/waterloo"
)

func TestCourseDocumentHashIgnoresSyncTimestamp(t *testing.T) {
	course := waterloo.Course{
		TermCode:                "1265",
		SubjectCode:             "CS",
		CatalogNumber:           "135",
		Title:                   "Designing Functional Programs",
		Description:             "An introduction to program design.",
		RequirementsDescription: "Prerequisite text.",
	}

	first := CourseDocument(course, "2026-06-15T10:00:00Z")
	second := CourseDocument(course, "2026-06-15T11:00:00Z")

	if first.ContentHash != second.ContentHash {
		t.Fatalf("content hash changed with sync timestamp: %q != %q", first.ContentHash, second.ContentHash)
	}
	if first.EntityURI != "uwgraph://courses/CS%20135" {
		t.Fatalf("EntityURI = %q, want uwgraph://courses/CS%%20135", first.EntityURI)
	}
}

func TestCourseDocumentHashChangesWithContent(t *testing.T) {
	course := waterloo.Course{
		TermCode:      "1265",
		SubjectCode:   "CS",
		CatalogNumber: "135",
		Title:         "Original",
	}
	first := CourseDocument(course, "2026-06-15T10:00:00Z")
	course.Title = "Changed"
	second := CourseDocument(course, "2026-06-15T10:00:00Z")

	if first.ContentHash == second.ContentHash {
		t.Fatal("content hash did not change with document content")
	}
}
