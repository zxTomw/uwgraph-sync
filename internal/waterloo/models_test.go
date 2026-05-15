package waterloo

import (
	"encoding/json"
	"testing"
)

func TestDecodeCourseFixture(t *testing.T) {
	raw := []byte(`{
		"courseId":"012345",
		"courseOfferNumber":1,
		"termCode":"1251",
		"termName":"Winter 2025",
		"associatedAcademicCareer":"UG",
		"associatedAcademicGroupCode":"MAT",
		"associatedAcademicOrgCode":"CS",
		"subjectCode":"CS",
		"catalogNumber":"246",
		"title":"Object-Oriented Software Development",
		"descriptionAbbreviated":"OOP",
		"description":"Course description",
		"gradingBasis":"NUM",
		"courseComponentCode":"LEC",
		"enrollConsentCode":"N",
		"enrollConsentDescription":"No Consent Required",
		"dropConsentCode":"N",
		"dropConsentDescription":"No Consent Required",
		"requirementsDescription":"Prereq"
	}`)

	var course Course
	if err := json.Unmarshal(raw, &course); err != nil {
		t.Fatalf("Unmarshal Course: %v", err)
	}
	if course.SubjectCode != "CS" || course.CatalogNumber != "246" {
		t.Fatalf("decoded course = %+v", course)
	}
}

func TestDecodeClassFixture(t *testing.T) {
	raw := []byte(`{
		"courseId":"012345",
		"courseOfferNumber":1,
		"sessionCode":"1",
		"classSection":101,
		"termCode":"1251",
		"classNumber":6543,
		"courseComponent":"LEC",
		"associatedClassCode":1,
		"maxEnrollmentCapacity":120,
		"enrolledStudents":118,
		"enrollConsentCode":"N",
		"enrollConsentDescription":"No Consent Required",
		"dropConsentCode":"N",
		"dropConsentDescription":"No Consent Required",
		"scheduleData":[{
			"courseId":"012345",
			"courseOfferNumber":1,
			"sessionCode":"1",
			"classSection":101,
			"termCode":"1251",
			"classMeetingNumber":1,
			"scheduleStartDate":"2025-01-06T00:00:00Z",
			"scheduleEndDate":"2025-04-04T00:00:00Z",
			"classMeetingStartTime":"2025-01-06T09:30:00Z",
			"classMeetingEndTime":"2025-01-06T10:20:00Z",
			"classMeetingDayPatternCode":"MWF",
			"classMeetingWeekPatternCode":"YYYYYYYYYYYY",
			"locationName":""
		}],
		"instructorData":[]
	}`)

	var class Class
	if err := json.Unmarshal(raw, &class); err != nil {
		t.Fatalf("Unmarshal Class: %v", err)
	}
	if len(class.ScheduleData) != 1 {
		t.Fatalf("ScheduleData length = %d, want 1", len(class.ScheduleData))
	}
	if class.ScheduleData[0].ScheduleStartDate == "" {
		t.Fatal("ScheduleStartDate was empty")
	}
}

func TestDecodeExamFixture(t *testing.T) {
	raw := []byte(`{
		"examDisplayName":"CS 246",
		"sections":"LEC 001",
		"isOnlineDescription":"No",
		"day":"Monday",
		"location":"PAC",
		"examWindowStartDate":"2025-04-12",
		"examWindowStartTime":"09:00",
		"examDuration":"2.5 hours",
		"notes":"",
		"termCode":"1251"
	}`)

	var exam Exam
	if err := json.Unmarshal(raw, &exam); err != nil {
		t.Fatalf("Unmarshal Exam: %v", err)
	}
	if exam.TermCode != "1251" || exam.ExamDisplayName != "CS 246" {
		t.Fatalf("decoded exam = %+v", exam)
	}
}
