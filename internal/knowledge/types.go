package knowledge

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("knowledge entity not found")

const (
	KindCourse               = "course"
	KindSubject              = "subject"
	KindAcademicOrganization = "academic_organization"
	KindBuilding             = "building"
)

type Document struct {
	DocumentKey     string
	Kind            string
	Title           string
	Text            string
	Aliases         []string
	ContentHash     string
	SourceEndpoint  string
	SourceEntityKey string
	EntityURI       string
	SyncedAt        string
}

type Scores struct {
	FullText float64 `json:"fullText"`
	Vector   float64 `json:"vector"`
	Fused    float64 `json:"fused"`
}

type Evidence struct {
	EntityURI       string         `json:"entityUri"`
	Kind            string         `json:"kind"`
	EntityKey       string         `json:"entityKey"`
	Title           string         `json:"title"`
	MatchedText     string         `json:"matchedText"`
	SourceEndpoint  string         `json:"sourceEndpoint"`
	SyncedAt        string         `json:"syncedAt"`
	Facts           map[string]any `json:"facts"`
	Scores          Scores         `json:"scores"`
	RequirementsRaw bool           `json:"requirementsRaw,omitempty"`
}

type SearchRequest struct {
	Query    string   `json:"query"`
	Kinds    []string `json:"kinds,omitempty"`
	TermCode string   `json:"termCode,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

type SearchResult struct {
	Query     string     `json:"query"`
	Retrieval string     `json:"retrieval"`
	Evidence  []Evidence `json:"evidence"`
}

type Candidate struct {
	Evidence Evidence
	Score    float64
}

type Course struct {
	CourseCode              string   `json:"courseCode"`
	SubjectCode             string   `json:"subjectCode"`
	CatalogNumber           string   `json:"catalogNumber"`
	Title                   string   `json:"title"`
	DescriptionAbbreviated  string   `json:"descriptionAbbreviated,omitempty"`
	Description             string   `json:"description,omitempty"`
	RequirementsDescription string   `json:"requirementsDescription,omitempty"`
	RequirementsWarning     string   `json:"requirementsWarning,omitempty"`
	Citation                Citation `json:"citation"`
}

type Offering struct {
	OfferingKey              string   `json:"offeringKey"`
	CourseID                 string   `json:"courseId"`
	CourseOfferNumber        int64    `json:"courseOfferNumber"`
	TermCode                 string   `json:"termCode"`
	TermName                 string   `json:"termName"`
	AssociatedAcademicCareer string   `json:"associatedAcademicCareer,omitempty"`
	GradingBasis             string   `json:"gradingBasis,omitempty"`
	CourseComponentCode      string   `json:"courseComponentCode,omitempty"`
	Citation                 Citation `json:"citation"`
}

type Meeting struct {
	MeetingKey   string `json:"meetingKey"`
	Number       int64  `json:"number"`
	StartDate    string `json:"startDate,omitempty"`
	EndDate      string `json:"endDate,omitempty"`
	StartTime    string `json:"startTime,omitempty"`
	EndTime      string `json:"endTime,omitempty"`
	DayPattern   string `json:"dayPattern,omitempty"`
	WeekPattern  string `json:"weekPattern,omitempty"`
	LocationName string `json:"locationName,omitempty"`
}

type Instructor struct {
	InstructorKey string `json:"instructorKey"`
	FirstName     string `json:"firstName,omitempty"`
	LastName      string `json:"lastName,omitempty"`
	RoleCode      string `json:"roleCode,omitempty"`
	MeetingNumber int64  `json:"meetingNumber,omitempty"`
}

type Section struct {
	SectionKey            string       `json:"sectionKey"`
	CourseCode            string       `json:"courseCode"`
	TermCode              string       `json:"termCode"`
	SessionCode           string       `json:"sessionCode,omitempty"`
	ClassSection          int64        `json:"classSection"`
	ClassNumber           int64        `json:"classNumber"`
	CourseComponent       string       `json:"courseComponent,omitempty"`
	MaxEnrollmentCapacity int64        `json:"maxEnrollmentCapacity"`
	EnrolledStudents      int64        `json:"enrolledStudents"`
	Meetings              []Meeting    `json:"meetings"`
	Instructors           []Instructor `json:"instructors"`
	Citation              Citation     `json:"citation"`
}

type SectionSearchRequest struct {
	CourseCode string   `json:"courseCode,omitempty"`
	TermCode   string   `json:"termCode,omitempty"`
	Days       []string `json:"days,omitempty"`
	StartAfter string   `json:"startAfter,omitempty"`
	EndBefore  string   `json:"endBefore,omitempty"`
	HasSeats   bool     `json:"hasSeats,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

type Exam struct {
	ExamKey             string   `json:"examKey"`
	ExamDisplayName     string   `json:"examDisplayName"`
	Sections            string   `json:"sections,omitempty"`
	IsOnlineDescription string   `json:"isOnlineDescription,omitempty"`
	Day                 string   `json:"day,omitempty"`
	Location            string   `json:"location,omitempty"`
	StartDate           string   `json:"startDate,omitempty"`
	StartTime           string   `json:"startTime,omitempty"`
	Duration            string   `json:"duration,omitempty"`
	Notes               string   `json:"notes,omitempty"`
	TermCode            string   `json:"termCode"`
	Citation            Citation `json:"citation"`
}

type ExamSearchRequest struct {
	TermCode string `json:"termCode,omitempty"`
	Sections string `json:"sections,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type Building struct {
	BuildingCode           string   `json:"buildingCode"`
	BuildingID             string   `json:"buildingId,omitempty"`
	ParentBuildingCode     string   `json:"parentBuildingCode,omitempty"`
	BuildingName           string   `json:"buildingName"`
	AlternateBuildingNames []string `json:"alternateBuildingNames,omitempty"`
	Latitude               *float64 `json:"latitude,omitempty"`
	Longitude              *float64 `json:"longitude,omitempty"`
	Citation               Citation `json:"citation"`
}

type Term struct {
	TermCode                 string   `json:"termCode"`
	Name                     string   `json:"name"`
	NameShort                string   `json:"nameShort,omitempty"`
	TermBeginDate            string   `json:"termBeginDate,omitempty"`
	TermEndDate              string   `json:"termEndDate,omitempty"`
	SixtyPercentCompleteDate string   `json:"sixtyPercentCompleteDate,omitempty"`
	AssociatedAcademicYear   int64    `json:"associatedAcademicYear,omitempty"`
	Citation                 Citation `json:"citation"`
}

type Citation struct {
	EntityURI      string `json:"entityUri"`
	SourceEndpoint string `json:"sourceEndpoint"`
	SyncedAt       string `json:"syncedAt"`
}

type PendingDocument struct {
	DocumentKey string
	Text        string
	ContentHash string
}

type EmbeddingUpdate struct {
	DocumentKey string
	ContentHash string
	Model       string
	Embedding   []float32
	EmbeddedAt  time.Time
}
