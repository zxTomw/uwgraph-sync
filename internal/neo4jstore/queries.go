package neo4jstore

import (
	"context"
	"fmt"
	"strings"

	"uwgraph/internal/knowledge"
)

func (s *Store) GetCourse(ctx context.Context, courseCode string) (knowledge.Course, error) {
	result, err := s.read(ctx, `
MATCH (course:Course {courseCode: $courseCode})
OPTIONAL MATCH (document:KnowledgeDocument)-[:DESCRIBES]->(course)
RETURN course.courseCode AS courseCode,
       course.subjectCode AS subjectCode,
       course.catalogNumber AS catalogNumber,
       course.title AS title,
       course.descriptionAbbreviated AS descriptionAbbreviated,
       course.description AS description,
       course.requirementsDescription AS requirementsDescription,
       coalesce(document.entityUri, 'uwgraph://courses/' + replace(course.courseCode, ' ', '%20')) AS entityUri,
       coalesce(document.sourceEndpoint, course.sourceEndpoint) AS sourceEndpoint,
       coalesce(document.syncedAt, course.syncedAt) AS syncedAt`, map[string]any{
		"courseCode": strings.TrimSpace(courseCode),
	})
	if err != nil {
		return knowledge.Course{}, fmt.Errorf("query course: %w", err)
	}
	if len(result.Records) == 0 {
		return knowledge.Course{}, knowledge.ErrNotFound
	}
	record := result.Records[0]
	course := knowledge.Course{
		CourseCode:              optionalString(record, "courseCode"),
		SubjectCode:             optionalString(record, "subjectCode"),
		CatalogNumber:           optionalString(record, "catalogNumber"),
		Title:                   optionalString(record, "title"),
		DescriptionAbbreviated:  optionalString(record, "descriptionAbbreviated"),
		Description:             optionalString(record, "description"),
		RequirementsDescription: optionalString(record, "requirementsDescription"),
		Citation: knowledge.Citation{
			EntityURI:      optionalString(record, "entityUri"),
			SourceEndpoint: optionalString(record, "sourceEndpoint"),
			SyncedAt:       optionalString(record, "syncedAt"),
		},
	}
	if course.RequirementsDescription != "" {
		course.RequirementsWarning = "Requirements are unverified source text and must not be treated as a validated prerequisite graph."
	}
	return course, nil
}

func (s *Store) ListCourseOfferings(ctx context.Context, courseCode, termCode string, limit int) ([]knowledge.Offering, error) {
	result, err := s.read(ctx, `
MATCH (course:Course {courseCode: $courseCode})<-[:INSTANCE_OF]-(offering:CourseOffering)-[:IN_TERM]->(term:Term)
WHERE $termCode = '' OR term.termCode = $termCode
RETURN offering.offeringKey AS offeringKey,
       offering.courseId AS courseId,
       offering.courseOfferNumber AS courseOfferNumber,
       term.termCode AS termCode,
       offering.termName AS termName,
       offering.associatedAcademicCareer AS associatedAcademicCareer,
       offering.gradingBasis AS gradingBasis,
       offering.courseComponentCode AS courseComponentCode,
       'uwgraph://offerings/' + offering.offeringKey AS entityUri,
       offering.sourceEndpoint AS sourceEndpoint,
       offering.syncedAt AS syncedAt
ORDER BY term.termCode DESC, offering.courseOfferNumber
LIMIT $limit`, map[string]any{
		"courseCode": strings.TrimSpace(courseCode),
		"termCode":   strings.TrimSpace(termCode),
		"limit":      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("query course offerings: %w", err)
	}
	offerings := make([]knowledge.Offering, 0, len(result.Records))
	for _, record := range result.Records {
		offerings = append(offerings, knowledge.Offering{
			OfferingKey:              optionalString(record, "offeringKey"),
			CourseID:                 optionalString(record, "courseId"),
			CourseOfferNumber:        optionalInt64(record, "courseOfferNumber"),
			TermCode:                 optionalString(record, "termCode"),
			TermName:                 optionalString(record, "termName"),
			AssociatedAcademicCareer: optionalString(record, "associatedAcademicCareer"),
			GradingBasis:             optionalString(record, "gradingBasis"),
			CourseComponentCode:      optionalString(record, "courseComponentCode"),
			Citation: knowledge.Citation{
				EntityURI:      optionalString(record, "entityUri"),
				SourceEndpoint: optionalString(record, "sourceEndpoint"),
				SyncedAt:       optionalString(record, "syncedAt"),
			},
		})
	}
	return offerings, nil
}

func (s *Store) SearchSections(ctx context.Context, request knowledge.SectionSearchRequest) ([]knowledge.Section, error) {
	result, err := s.read(ctx, `
MATCH (section:ClassSection)-[:SECTION_OF]->(offering:CourseOffering)-[:INSTANCE_OF]->(course:Course)
MATCH (offering)-[:IN_TERM]->(term:Term)
WHERE ($courseCode = '' OR course.courseCode = $courseCode)
  AND ($termCode = '' OR term.termCode = $termCode)
  AND (NOT $hasSeats OR section.enrolledStudents < section.maxEnrollmentCapacity)
  AND (
    (size($days) = 0 AND $startAfter = '' AND $endBefore = '')
    OR EXISTS {
      MATCH (meeting:ClassMeeting)-[:MEETING_OF]->(section)
      WHERE (size($days) = 0 OR any(day IN $days WHERE meeting.classMeetingDayPatternCode CONTAINS day))
        AND ($startAfter = '' OR meeting.classMeetingStartTime >= $startAfter)
        AND ($endBefore = '' OR meeting.classMeetingEndTime <= $endBefore)
    }
  )
CALL {
  WITH section
  OPTIONAL MATCH (meeting:ClassMeeting)-[:MEETING_OF]->(section)
  RETURN collect(DISTINCT meeting {
    .meetingKey,
    number: meeting.classMeetingNumber,
    startDate: meeting.scheduleStartDate,
    endDate: meeting.scheduleEndDate,
    startTime: meeting.classMeetingStartTime,
    endTime: meeting.classMeetingEndTime,
    dayPattern: meeting.classMeetingDayPatternCode,
    weekPattern: meeting.classMeetingWeekPatternCode,
    .locationName
  }) AS meetings
}
CALL {
  WITH section
  OPTIONAL MATCH (instructor:Instructor)-[teaches:TEACHES]->(section)
  RETURN collect(DISTINCT instructor {
    .instructorKey,
    .firstName,
    .lastName,
    roleCode: teaches.roleCode,
    meetingNumber: teaches.classMeetingNumber
  }) AS instructors
}
RETURN section.sectionKey AS sectionKey,
       course.courseCode AS courseCode,
       term.termCode AS termCode,
       section.sessionCode AS sessionCode,
       section.classSection AS classSection,
       section.classNumber AS classNumber,
       section.courseComponent AS courseComponent,
       section.maxEnrollmentCapacity AS maxEnrollmentCapacity,
       section.enrolledStudents AS enrolledStudents,
       meetings,
       instructors,
       'uwgraph://sections/' + section.sectionKey AS entityUri,
       section.sourceEndpoint AS sourceEndpoint,
       section.syncedAt AS syncedAt
ORDER BY course.courseCode, section.classNumber
LIMIT $limit`, map[string]any{
		"courseCode": strings.TrimSpace(request.CourseCode),
		"termCode":   strings.TrimSpace(request.TermCode),
		"days":       request.Days,
		"startAfter": strings.TrimSpace(request.StartAfter),
		"endBefore":  strings.TrimSpace(request.EndBefore),
		"hasSeats":   request.HasSeats,
		"limit":      request.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("query sections: %w", err)
	}

	sections := make([]knowledge.Section, 0, len(result.Records))
	for _, record := range result.Records {
		meetings, err := meetingsFromValue(recordValue(record, "meetings"))
		if err != nil {
			return nil, err
		}
		instructors, err := instructorsFromValue(recordValue(record, "instructors"))
		if err != nil {
			return nil, err
		}
		sections = append(sections, knowledge.Section{
			SectionKey:            optionalString(record, "sectionKey"),
			CourseCode:            optionalString(record, "courseCode"),
			TermCode:              optionalString(record, "termCode"),
			SessionCode:           optionalString(record, "sessionCode"),
			ClassSection:          optionalInt64(record, "classSection"),
			ClassNumber:           optionalInt64(record, "classNumber"),
			CourseComponent:       optionalString(record, "courseComponent"),
			MaxEnrollmentCapacity: optionalInt64(record, "maxEnrollmentCapacity"),
			EnrolledStudents:      optionalInt64(record, "enrolledStudents"),
			Meetings:              meetings,
			Instructors:           instructors,
			Citation: knowledge.Citation{
				EntityURI:      optionalString(record, "entityUri"),
				SourceEndpoint: optionalString(record, "sourceEndpoint"),
				SyncedAt:       optionalString(record, "syncedAt"),
			},
		})
	}
	return sections, nil
}

func (s *Store) ListExams(ctx context.Context, request knowledge.ExamSearchRequest) ([]knowledge.Exam, error) {
	result, err := s.read(ctx, `
MATCH (exam:Exam)-[:IN_TERM]->(term:Term)
WHERE ($termCode = '' OR term.termCode = $termCode)
  AND ($sections = '' OR toLower(exam.sections) CONTAINS toLower($sections))
RETURN exam.examKey AS examKey,
       exam.examDisplayName AS examDisplayName,
       exam.sections AS sections,
       exam.isOnlineDescription AS isOnlineDescription,
       exam.day AS day,
       exam.location AS location,
       exam.examWindowStartDate AS startDate,
       exam.examWindowStartTime AS startTime,
       exam.examDuration AS duration,
       exam.notes AS notes,
       term.termCode AS termCode,
       'uwgraph://exams/' + exam.examKey AS entityUri,
       exam.sourceEndpoint AS sourceEndpoint,
       exam.syncedAt AS syncedAt
ORDER BY exam.examWindowStartDate, exam.examWindowStartTime
LIMIT $limit`, map[string]any{
		"termCode": strings.TrimSpace(request.TermCode),
		"sections": strings.TrimSpace(request.Sections),
		"limit":    request.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("query exams: %w", err)
	}
	exams := make([]knowledge.Exam, 0, len(result.Records))
	for _, record := range result.Records {
		exams = append(exams, knowledge.Exam{
			ExamKey:             optionalString(record, "examKey"),
			ExamDisplayName:     optionalString(record, "examDisplayName"),
			Sections:            optionalString(record, "sections"),
			IsOnlineDescription: optionalString(record, "isOnlineDescription"),
			Day:                 optionalString(record, "day"),
			Location:            optionalString(record, "location"),
			StartDate:           optionalString(record, "startDate"),
			StartTime:           optionalString(record, "startTime"),
			Duration:            optionalString(record, "duration"),
			Notes:               optionalString(record, "notes"),
			TermCode:            optionalString(record, "termCode"),
			Citation: knowledge.Citation{
				EntityURI:      optionalString(record, "entityUri"),
				SourceEndpoint: optionalString(record, "sourceEndpoint"),
				SyncedAt:       optionalString(record, "syncedAt"),
			},
		})
	}
	return exams, nil
}

func (s *Store) GetBuilding(ctx context.Context, buildingCode string) (knowledge.Building, error) {
	result, err := s.read(ctx, `
MATCH (building:Building {buildingCode: $buildingCode})
OPTIONAL MATCH (document:KnowledgeDocument)-[:DESCRIBES]->(building)
RETURN building.buildingCode AS buildingCode,
       building.buildingId AS buildingId,
       building.parentBuildingCode AS parentBuildingCode,
       building.buildingName AS buildingName,
       building.alternateBuildingNames AS alternateBuildingNames,
       building.latitude AS latitude,
       building.longitude AS longitude,
       coalesce(document.entityUri, 'uwgraph://buildings/' + building.buildingCode) AS entityUri,
       coalesce(document.sourceEndpoint, building.sourceEndpoint) AS sourceEndpoint,
       coalesce(document.syncedAt, building.syncedAt) AS syncedAt`, map[string]any{
		"buildingCode": strings.TrimSpace(buildingCode),
	})
	if err != nil {
		return knowledge.Building{}, fmt.Errorf("query building: %w", err)
	}
	if len(result.Records) == 0 {
		return knowledge.Building{}, knowledge.ErrNotFound
	}
	record := result.Records[0]
	return knowledge.Building{
		BuildingCode:           optionalString(record, "buildingCode"),
		BuildingID:             optionalString(record, "buildingId"),
		ParentBuildingCode:     optionalString(record, "parentBuildingCode"),
		BuildingName:           optionalString(record, "buildingName"),
		AlternateBuildingNames: optionalStringSlice(record, "alternateBuildingNames"),
		Latitude:               optionalFloat64Pointer(record, "latitude"),
		Longitude:              optionalFloat64Pointer(record, "longitude"),
		Citation: knowledge.Citation{
			EntityURI:      optionalString(record, "entityUri"),
			SourceEndpoint: optionalString(record, "sourceEndpoint"),
			SyncedAt:       optionalString(record, "syncedAt"),
		},
	}, nil
}

func (s *Store) GetTerm(ctx context.Context, termCode string) (knowledge.Term, error) {
	result, err := s.read(ctx, `
MATCH (term:Term {termCode: $termCode})
RETURN term.termCode AS termCode,
       term.name AS name,
       term.nameShort AS nameShort,
       term.termBeginDate AS termBeginDate,
       term.termEndDate AS termEndDate,
       term.sixtyPercentCompleteDate AS sixtyPercentCompleteDate,
       term.associatedAcademicYear AS associatedAcademicYear,
       'uwgraph://terms/' + term.termCode AS entityUri,
       term.sourceEndpoint AS sourceEndpoint,
       term.syncedAt AS syncedAt`, map[string]any{
		"termCode": strings.TrimSpace(termCode),
	})
	if err != nil {
		return knowledge.Term{}, fmt.Errorf("query term: %w", err)
	}
	if len(result.Records) == 0 {
		return knowledge.Term{}, knowledge.ErrNotFound
	}
	record := result.Records[0]
	return knowledge.Term{
		TermCode:                 optionalString(record, "termCode"),
		Name:                     optionalString(record, "name"),
		NameShort:                optionalString(record, "nameShort"),
		TermBeginDate:            optionalString(record, "termBeginDate"),
		TermEndDate:              optionalString(record, "termEndDate"),
		SixtyPercentCompleteDate: optionalString(record, "sixtyPercentCompleteDate"),
		AssociatedAcademicYear:   optionalInt64(record, "associatedAcademicYear"),
		Citation: knowledge.Citation{
			EntityURI:      optionalString(record, "entityUri"),
			SourceEndpoint: optionalString(record, "sourceEndpoint"),
			SyncedAt:       optionalString(record, "syncedAt"),
		},
	}, nil
}

func optionalString(record interface{ Get(string) (any, bool) }, key string) string {
	value, ok := record.Get(key)
	if !ok || value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

func optionalInt64(record interface{ Get(string) (any, bool) }, key string) int64 {
	value, ok := record.Get(key)
	if !ok || value == nil {
		return 0
	}
	number, _ := value.(int64)
	return number
}

func optionalFloat64Pointer(record interface{ Get(string) (any, bool) }, key string) *float64 {
	value, ok := record.Get(key)
	if !ok || value == nil {
		return nil
	}
	number, ok := value.(float64)
	if !ok {
		return nil
	}
	return &number
}

func optionalStringSlice(record interface{ Get(string) (any, bool) }, key string) []string {
	value, ok := record.Get(key)
	if !ok || value == nil {
		return nil
	}
	return stringsFromValue(value)
}

func recordValue(record interface{ Get(string) (any, bool) }, key string) any {
	value, _ := record.Get(key)
	return value
}

func meetingsFromValue(value any) ([]knowledge.Meeting, error) {
	items, ok := value.([]any)
	if !ok && value != nil {
		return nil, fmt.Errorf("meetings have type %T, want []any", value)
	}
	meetings := make([]knowledge.Meeting, 0, len(items))
	for _, item := range items {
		values, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("meeting has type %T, want map[string]any", item)
		}
		meetings = append(meetings, knowledge.Meeting{
			MeetingKey:   stringValue(values["meetingKey"]),
			Number:       int64Value(values["number"]),
			StartDate:    stringValue(values["startDate"]),
			EndDate:      stringValue(values["endDate"]),
			StartTime:    stringValue(values["startTime"]),
			EndTime:      stringValue(values["endTime"]),
			DayPattern:   stringValue(values["dayPattern"]),
			WeekPattern:  stringValue(values["weekPattern"]),
			LocationName: stringValue(values["locationName"]),
		})
	}
	return meetings, nil
}

func instructorsFromValue(value any) ([]knowledge.Instructor, error) {
	items, ok := value.([]any)
	if !ok && value != nil {
		return nil, fmt.Errorf("instructors have type %T, want []any", value)
	}
	instructors := make([]knowledge.Instructor, 0, len(items))
	for _, item := range items {
		values, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("instructor has type %T, want map[string]any", item)
		}
		instructors = append(instructors, knowledge.Instructor{
			InstructorKey: stringValue(values["instructorKey"]),
			FirstName:     stringValue(values["firstName"]),
			LastName:      stringValue(values["lastName"]),
			RoleCode:      stringValue(values["roleCode"]),
			MeetingNumber: int64Value(values["meetingNumber"]),
		})
	}
	return instructors, nil
}

func stringsFromValue(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		result := make([]string, 0, len(values))
		for _, item := range values {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func int64Value(value any) int64 {
	number, _ := value.(int64)
	return number
}
