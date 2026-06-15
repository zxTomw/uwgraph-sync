package graph

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

func CourseCode(subjectCode, catalogNumber string) string {
	subjectCode = strings.TrimSpace(subjectCode)
	catalogNumber = strings.TrimSpace(catalogNumber)
	if subjectCode == "" || catalogNumber == "" {
		return ""
	}
	return subjectCode + " " + catalogNumber
}

func OfferingKey(termCode, courseID string, courseOfferNumber int) string {
	return strings.Join([]string{
		strings.TrimSpace(termCode),
		strings.TrimSpace(courseID),
		strconv.Itoa(courseOfferNumber),
	}, "|")
}

func SectionKey(termCode, courseID string, courseOfferNumber int, sessionCode string, classSection, classNumber int) string {
	return strings.Join([]string{
		strings.TrimSpace(termCode),
		strings.TrimSpace(courseID),
		strconv.Itoa(courseOfferNumber),
		strings.TrimSpace(sessionCode),
		strconv.Itoa(classSection),
		strconv.Itoa(classNumber),
	}, "|")
}

func MeetingKey(sectionKey string, classMeetingNumber int) string {
	return fmt.Sprintf("%s|%d", strings.TrimSpace(sectionKey), classMeetingNumber)
}

func ExamKey(termCode, displayName, sections, startDate, startTime string) string {
	parts := []string{
		strings.TrimSpace(termCode),
		strings.TrimSpace(displayName),
		strings.TrimSpace(sections),
		strings.TrimSpace(startDate),
		strings.TrimSpace(startTime),
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return "exam:" + hex.EncodeToString(sum[:])
}

func InstructorKey(uniqueIdentifier string) string {
	identifier := strings.TrimSpace(uniqueIdentifier)
	if identifier == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(identifier))
	return "instructor:" + hex.EncodeToString(sum[:])
}
