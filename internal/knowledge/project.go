package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"

	"uwgraph/internal/graph"
	"uwgraph/internal/waterloo"
)

func CourseDocument(course waterloo.Course, syncedAt string) Document {
	courseCode := graph.CourseCode(course.SubjectCode, course.CatalogNumber)
	doc := Document{
		DocumentKey:     "course:" + courseCode,
		Kind:            KindCourse,
		Title:           strings.TrimSpace(course.Title),
		Aliases:         compactStrings(courseCode, course.SubjectCode+" "+course.CatalogNumber, course.Title),
		SourceEndpoint:  "/v3/Courses/" + url.PathEscape(strings.TrimSpace(course.TermCode)),
		SourceEntityKey: courseCode,
		EntityURI:       "uwgraph://courses/" + url.PathEscape(courseCode),
		SyncedAt:        syncedAt,
	}
	doc.Text = labeledText(
		"Course", courseCode,
		"Title", course.Title,
		"Description", course.Description,
		"Abbreviated description", course.DescriptionAbbreviated,
		"Requirements", course.RequirementsDescription,
	)
	doc.ContentHash = contentHash(doc)
	return doc
}

func SubjectDocument(subject waterloo.Subject, syncedAt string) Document {
	code := strings.TrimSpace(subject.Code)
	doc := Document{
		DocumentKey:     "subject:" + code,
		Kind:            KindSubject,
		Title:           strings.TrimSpace(subject.Name),
		Aliases:         compactStrings(code, subject.Name, subject.DescriptionAbbreviated),
		SourceEndpoint:  "/v3/Subjects",
		SourceEntityKey: code,
		EntityURI:       "uwgraph://subjects/" + url.PathEscape(code),
		SyncedAt:        syncedAt,
	}
	doc.Text = labeledText(
		"Subject", code,
		"Name", subject.Name,
		"Description", subject.Description,
		"Abbreviated description", subject.DescriptionAbbreviated,
	)
	doc.ContentHash = contentHash(doc)
	return doc
}

func AcademicOrganizationDocument(org waterloo.AcademicOrganization, syncedAt string) Document {
	code := strings.TrimSpace(org.Code)
	doc := Document{
		DocumentKey:     "academic-organization:" + code,
		Kind:            KindAcademicOrganization,
		Title:           strings.TrimSpace(org.Name),
		Aliases:         compactStrings(code, org.Name, org.DescriptionFormal),
		SourceEndpoint:  "/v3/AcademicOrganizations",
		SourceEntityKey: code,
		EntityURI:       "uwgraph://academic-organizations/" + url.PathEscape(code),
		SyncedAt:        syncedAt,
	}
	doc.Text = labeledText(
		"Academic organization", code,
		"Name", org.Name,
		"Formal description", org.DescriptionFormal,
		"Description", org.Description,
	)
	doc.ContentHash = contentHash(doc)
	return doc
}

func BuildingDocument(location waterloo.Location, syncedAt string) Document {
	code := strings.TrimSpace(location.BuildingCode)
	aliases := append([]string{code, location.BuildingName}, location.AlternateBuildingNames...)
	doc := Document{
		DocumentKey:     "building:" + code,
		Kind:            KindBuilding,
		Title:           strings.TrimSpace(location.BuildingName),
		Aliases:         compactStrings(aliases...),
		SourceEndpoint:  "/v3/Locations",
		SourceEntityKey: code,
		EntityURI:       "uwgraph://buildings/" + url.PathEscape(code),
		SyncedAt:        syncedAt,
	}
	doc.Text = labeledText(
		"Building", code,
		"Name", location.BuildingName,
		"Alternate names", strings.Join(location.AlternateBuildingNames, ", "),
		"Parent building", location.ParentBuildingCode,
	)
	doc.ContentHash = contentHash(doc)
	return doc
}

func contentHash(doc Document) string {
	canonical := struct {
		Kind            string
		Title           string
		Text            string
		Aliases         []string
		SourceEntityKey string
	}{
		Kind:            doc.Kind,
		Title:           doc.Title,
		Text:            doc.Text,
		Aliases:         doc.Aliases,
		SourceEntityKey: doc.SourceEntityKey,
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func labeledText(parts ...string) string {
	lines := make([]string, 0, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		value := strings.TrimSpace(parts[i+1])
		if value == "" {
			continue
		}
		lines = append(lines, parts[i]+": "+value)
	}
	return strings.Join(lines, "\n")
}

func compactStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
