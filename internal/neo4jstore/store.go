package neo4jstore

import (
	"context"
	"log/slog"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"uwgraph/internal/graph"
	"uwgraph/internal/waterloo"
)

const batchSize = 500

type Store struct {
	driver   neo4j.DriverWithContext
	database string
	logger   *slog.Logger
}

func New(driver neo4j.DriverWithContext, database string, logger *slog.Logger) *Store {
	return &Store{driver: driver, database: database, logger: logger}
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	constraints := []string{
		"CREATE CONSTRAINT term_code IF NOT EXISTS FOR (n:Term) REQUIRE n.termCode IS UNIQUE",
		"CREATE CONSTRAINT course_code IF NOT EXISTS FOR (n:Course) REQUIRE n.courseCode IS UNIQUE",
		"CREATE CONSTRAINT course_offering_key IF NOT EXISTS FOR (n:CourseOffering) REQUIRE n.offeringKey IS UNIQUE",
		"CREATE CONSTRAINT subject_code IF NOT EXISTS FOR (n:Subject) REQUIRE n.code IS UNIQUE",
		"CREATE CONSTRAINT academic_org_code IF NOT EXISTS FOR (n:AcademicOrganization) REQUIRE n.code IS UNIQUE",
		"CREATE CONSTRAINT building_code IF NOT EXISTS FOR (n:Building) REQUIRE n.buildingCode IS UNIQUE",
		"CREATE CONSTRAINT class_section_key IF NOT EXISTS FOR (n:ClassSection) REQUIRE n.sectionKey IS UNIQUE",
		"CREATE CONSTRAINT class_meeting_key IF NOT EXISTS FOR (n:ClassMeeting) REQUIRE n.meetingKey IS UNIQUE",
		"CREATE CONSTRAINT exam_key IF NOT EXISTS FOR (n:Exam) REQUIRE n.examKey IS UNIQUE",
		"CREATE INDEX course_title IF NOT EXISTS FOR (n:Course) ON (n.title)",
		"CREATE INDEX building_name IF NOT EXISTS FOR (n:Building) ON (n.buildingName)",
	}
	for _, query := range constraints {
		if err := s.write(ctx, query, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpsertTerms(ctx context.Context, terms []waterloo.Term) (int, error) {
	rows := make([]map[string]any, 0, len(terms))
	for _, term := range terms {
		if term.TermCode == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"termCode":                 term.TermCode,
			"name":                     term.Name,
			"nameShort":                term.NameShort,
			"termBeginDate":            term.TermBeginDate,
			"termEndDate":              term.TermEndDate,
			"sixtyPercentCompleteDate": term.SixtyPercentCompleteDate,
			"associatedAcademicYear":   term.AssociatedAcademicYear,
		})
	}
	return s.runBatches(ctx, rows, `
UNWIND $rows AS row
MERGE (t:Term {termCode: row.termCode})
SET t += row`)
}

func (s *Store) UpsertAcademicOrganizations(ctx context.Context, orgs []waterloo.AcademicOrganization) (int, error) {
	rows := make([]map[string]any, 0, len(orgs))
	for _, org := range orgs {
		if org.Code == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"code":                 org.Code,
			"name":                 org.Name,
			"description":          org.Description,
			"descriptionFormal":    org.DescriptionFormal,
			"associatedCampusCode": org.AssociatedCampusCode,
		})
	}
	return s.runBatches(ctx, rows, `
UNWIND $rows AS row
MERGE (o:AcademicOrganization {code: row.code})
SET o += row`)
}

func (s *Store) UpsertSubjects(ctx context.Context, subjects []waterloo.Subject) (int, error) {
	rows := make([]map[string]any, 0, len(subjects))
	for _, subject := range subjects {
		if subject.Code == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"code":                      subject.Code,
			"name":                      subject.Name,
			"descriptionAbbreviated":    subject.DescriptionAbbreviated,
			"description":               subject.Description,
			"associatedAcademicOrgCode": subject.AssociatedAcademicOrgCode,
		})
	}
	if _, err := s.runBatches(ctx, rows, `
UNWIND $rows AS row
MERGE (s:Subject {code: row.code})
SET s += row`); err != nil {
		return 0, err
	}
	if _, err := s.runBatches(ctx, rowsWith(rows, "associatedAcademicOrgCode"), `
UNWIND $rows AS row
MATCH (s:Subject {code: row.code})
MERGE (o:AcademicOrganization {code: row.associatedAcademicOrgCode})
MERGE (s)-[:PART_OF]->(o)`); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *Store) UpsertLocations(ctx context.Context, locations []waterloo.Location) (int, error) {
	rows := make([]map[string]any, 0, len(locations))
	for _, location := range locations {
		if location.BuildingCode == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"buildingId":             location.BuildingID,
			"buildingCode":           location.BuildingCode,
			"parentBuildingCode":     location.ParentBuildingCode,
			"buildingName":           location.BuildingName,
			"alternateBuildingNames": location.AlternateBuildingNames,
			"latitude":               floatValue(location.Latitude),
			"longitude":              floatValue(location.Longitude),
		})
	}
	if _, err := s.runBatches(ctx, rows, `
UNWIND $rows AS row
MERGE (b:Building {buildingCode: row.buildingCode})
SET b += row`); err != nil {
		return 0, err
	}
	if _, err := s.runBatches(ctx, rowsWith(rows, "parentBuildingCode"), `
UNWIND $rows AS row
MATCH (b:Building {buildingCode: row.buildingCode})
MERGE (parent:Building {buildingCode: row.parentBuildingCode})
MERGE (b)-[:PART_OF]->(parent)`); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *Store) UpsertCourses(ctx context.Context, courses []waterloo.Course) (int, error) {
	rows := make([]map[string]any, 0, len(courses))
	for _, course := range courses {
		if course.TermCode == "" || course.CourseID == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"offeringKey":                 graph.OfferingKey(course.TermCode, course.CourseID, course.CourseOfferNumber),
			"courseCode":                  graph.CourseCode(course.SubjectCode, course.CatalogNumber),
			"courseId":                    course.CourseID,
			"courseOfferNumber":           course.CourseOfferNumber,
			"termCode":                    course.TermCode,
			"termName":                    course.TermName,
			"associatedAcademicCareer":    course.AssociatedAcademicCareer,
			"associatedAcademicGroupCode": course.AssociatedAcademicGroupCode,
			"associatedAcademicOrgCode":   course.AssociatedAcademicOrgCode,
			"subjectCode":                 course.SubjectCode,
			"catalogNumber":               course.CatalogNumber,
			"title":                       course.Title,
			"descriptionAbbreviated":      course.DescriptionAbbreviated,
			"description":                 course.Description,
			"gradingBasis":                course.GradingBasis,
			"courseComponentCode":         course.CourseComponentCode,
			"enrollConsentCode":           course.EnrollConsentCode,
			"enrollConsentDescription":    course.EnrollConsentDescription,
			"dropConsentCode":             course.DropConsentCode,
			"dropConsentDescription":      course.DropConsentDescription,
			"requirementsDescription":     course.RequirementsDescription,
		})
	}
	if _, err := s.runBatches(ctx, rows, `
UNWIND $rows AS row
MERGE (o:CourseOffering {offeringKey: row.offeringKey})
SET o += row
MERGE (t:Term {termCode: row.termCode})
MERGE (o)-[:IN_TERM]->(t)`); err != nil {
		return 0, err
	}
	if _, err := s.runBatches(ctx, rowsWith(rows, "courseCode"), `
UNWIND $rows AS row
MATCH (o:CourseOffering {offeringKey: row.offeringKey})
MERGE (c:Course {courseCode: row.courseCode})
SET c.subjectCode = row.subjectCode,
    c.catalogNumber = row.catalogNumber,
    c.title = row.title,
    c.descriptionAbbreviated = row.descriptionAbbreviated,
    c.description = row.description,
    c.requirementsDescription = row.requirementsDescription
MERGE (o)-[:INSTANCE_OF]->(c)`); err != nil {
		return 0, err
	}
	if _, err := s.runBatches(ctx, rowsWith(rows, "subjectCode"), `
UNWIND $rows AS row
MATCH (o:CourseOffering {offeringKey: row.offeringKey})
MERGE (s:Subject {code: row.subjectCode})
MERGE (o)-[:IN_SUBJECT]->(s)`); err != nil {
		return 0, err
	}
	if _, err := s.runBatches(ctx, rowsWith(rows, "associatedAcademicOrgCode"), `
UNWIND $rows AS row
MATCH (o:CourseOffering {offeringKey: row.offeringKey})
MERGE (a:AcademicOrganization {code: row.associatedAcademicOrgCode})
MERGE (o)-[:OWNED_BY]->(a)`); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *Store) UpsertClasses(ctx context.Context, classes []waterloo.Class) (int, int, error) {
	sectionRows := make([]map[string]any, 0, len(classes))
	meetingRows := make([]map[string]any, 0)
	for _, class := range classes {
		if class.TermCode == "" || class.CourseID == "" {
			continue
		}
		sectionKey := graph.SectionKey(class.TermCode, class.CourseID, class.CourseOfferNumber, class.SessionCode, class.ClassSection, class.ClassNumber)
		offeringKey := graph.OfferingKey(class.TermCode, class.CourseID, class.CourseOfferNumber)
		sectionRows = append(sectionRows, map[string]any{
			"sectionKey":               sectionKey,
			"offeringKey":              offeringKey,
			"courseId":                 class.CourseID,
			"courseOfferNumber":        class.CourseOfferNumber,
			"sessionCode":              class.SessionCode,
			"classSection":             class.ClassSection,
			"termCode":                 class.TermCode,
			"classNumber":              class.ClassNumber,
			"courseComponent":          class.CourseComponent,
			"associatedClassCode":      class.AssociatedClassCode,
			"maxEnrollmentCapacity":    class.MaxEnrollmentCapacity,
			"enrolledStudents":         class.EnrolledStudents,
			"enrollConsentCode":        class.EnrollConsentCode,
			"enrollConsentDescription": class.EnrollConsentDescription,
			"dropConsentCode":          class.DropConsentCode,
			"dropConsentDescription":   class.DropConsentDescription,
		})
		for _, meeting := range class.ScheduleData {
			meetingRows = append(meetingRows, map[string]any{
				"meetingKey":                  graph.MeetingKey(sectionKey, meeting.ClassMeetingNumber),
				"sectionKey":                  sectionKey,
				"courseId":                    meeting.CourseID,
				"courseOfferNumber":           meeting.CourseOfferNumber,
				"sessionCode":                 meeting.SessionCode,
				"classSection":                meeting.ClassSection,
				"termCode":                    meeting.TermCode,
				"classMeetingNumber":          meeting.ClassMeetingNumber,
				"scheduleStartDate":           meeting.ScheduleStartDate,
				"scheduleEndDate":             meeting.ScheduleEndDate,
				"classMeetingStartTime":       meeting.ClassMeetingStartTime,
				"classMeetingEndTime":         meeting.ClassMeetingEndTime,
				"classMeetingDayPatternCode":  meeting.ClassMeetingDayPatternCode,
				"classMeetingWeekPatternCode": meeting.ClassMeetingWeekPatternCode,
				"locationName":                meeting.LocationName,
			})
		}
	}
	if _, err := s.runBatches(ctx, sectionRows, `
UNWIND $rows AS row
MERGE (section:ClassSection {sectionKey: row.sectionKey})
SET section += row
MERGE (offering:CourseOffering {offeringKey: row.offeringKey})
MERGE (section)-[:SECTION_OF]->(offering)`); err != nil {
		return 0, 0, err
	}
	if _, err := s.runBatches(ctx, meetingRows, `
UNWIND $rows AS row
MERGE (meeting:ClassMeeting {meetingKey: row.meetingKey})
SET meeting += row
MERGE (section:ClassSection {sectionKey: row.sectionKey})
MERGE (meeting)-[:MEETING_OF]->(section)`); err != nil {
		return 0, 0, err
	}
	return len(sectionRows), len(meetingRows), nil
}

func (s *Store) UpsertExams(ctx context.Context, exams []waterloo.Exam) (int, error) {
	rows := make([]map[string]any, 0, len(exams))
	for _, exam := range exams {
		if exam.TermCode == "" {
			continue
		}
		rows = append(rows, map[string]any{
			"examKey":             graph.ExamKey(exam.TermCode, exam.ExamDisplayName, exam.Sections, exam.ExamWindowStartDate, exam.ExamWindowStartTime),
			"examDisplayName":     exam.ExamDisplayName,
			"sections":            exam.Sections,
			"isOnlineDescription": exam.IsOnlineDescription,
			"day":                 exam.Day,
			"location":            exam.Location,
			"examWindowStartDate": exam.ExamWindowStartDate,
			"examWindowStartTime": exam.ExamWindowStartTime,
			"examDuration":        exam.ExamDuration,
			"notes":               exam.Notes,
			"termCode":            exam.TermCode,
		})
	}
	return s.runBatches(ctx, rows, `
UNWIND $rows AS row
MERGE (exam:Exam {examKey: row.examKey})
SET exam += row
MERGE (term:Term {termCode: row.termCode})
MERGE (exam)-[:IN_TERM]->(term)`)
}

func (s *Store) runBatches(ctx context.Context, rows []map[string]any, query string) (int, error) {
	for start := 0; start < len(rows); start += batchSize {
		end := start + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := s.write(ctx, query, map[string]any{"rows": rows[start:end]}); err != nil {
			return start, err
		}
	}
	return len(rows), nil
}

func (s *Store) write(ctx context.Context, query string, params map[string]any) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer func() {
		if err := session.Close(ctx); err != nil {
			s.logger.Error("close neo4j session", "error", err)
		}
	}()
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		_, err = result.Consume(ctx)
		return nil, err
	})
	return err
}

func rowsWith(rows []map[string]any, key string) []map[string]any {
	filtered := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		value, ok := row[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok || text == "" {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func floatValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
