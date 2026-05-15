package waterloo

type Term struct {
	TermCode                 string `json:"termCode"`
	Name                     string `json:"name"`
	NameShort                string `json:"nameShort"`
	TermBeginDate            string `json:"termBeginDate"`
	TermEndDate              string `json:"termEndDate"`
	SixtyPercentCompleteDate string `json:"sixtyPercentCompleteDate"`
	AssociatedAcademicYear   int    `json:"associatedAcademicYear"`
}

type Subject struct {
	Code                      string `json:"code"`
	Name                      string `json:"name"`
	DescriptionAbbreviated    string `json:"descriptionAbbreviated"`
	Description               string `json:"description"`
	AssociatedAcademicOrgCode string `json:"associatedAcademicOrgCode"`
}

type AcademicOrganization struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Description          string `json:"description"`
	DescriptionFormal    string `json:"descriptionFormal"`
	AssociatedCampusCode string `json:"associatedCampusCode"`
}

type Location struct {
	BuildingID             string   `json:"buildingId"`
	BuildingCode           string   `json:"buildingCode"`
	ParentBuildingCode     string   `json:"parentBuildingCode"`
	BuildingName           string   `json:"buildingName"`
	AlternateBuildingNames []string `json:"alternateBuildingNames"`
	Latitude               *float64 `json:"latitude"`
	Longitude              *float64 `json:"longitude"`
}

type Course struct {
	CourseID                    string `json:"courseId"`
	CourseOfferNumber           int    `json:"courseOfferNumber"`
	TermCode                    string `json:"termCode"`
	TermName                    string `json:"termName"`
	AssociatedAcademicCareer    string `json:"associatedAcademicCareer"`
	AssociatedAcademicGroupCode string `json:"associatedAcademicGroupCode"`
	AssociatedAcademicOrgCode   string `json:"associatedAcademicOrgCode"`
	SubjectCode                 string `json:"subjectCode"`
	CatalogNumber               string `json:"catalogNumber"`
	Title                       string `json:"title"`
	DescriptionAbbreviated      string `json:"descriptionAbbreviated"`
	Description                 string `json:"description"`
	GradingBasis                string `json:"gradingBasis"`
	CourseComponentCode         string `json:"courseComponentCode"`
	EnrollConsentCode           string `json:"enrollConsentCode"`
	EnrollConsentDescription    string `json:"enrollConsentDescription"`
	DropConsentCode             string `json:"dropConsentCode"`
	DropConsentDescription      string `json:"dropConsentDescription"`
	RequirementsDescription     string `json:"requirementsDescription"`
}

type Class struct {
	CourseID                 string            `json:"courseId"`
	CourseOfferNumber        int               `json:"courseOfferNumber"`
	SessionCode              string            `json:"sessionCode"`
	ClassSection             int               `json:"classSection"`
	TermCode                 string            `json:"termCode"`
	ClassNumber              int               `json:"classNumber"`
	CourseComponent          string            `json:"courseComponent"`
	AssociatedClassCode      int               `json:"associatedClassCode"`
	MaxEnrollmentCapacity    int               `json:"maxEnrollmentCapacity"`
	EnrolledStudents         int               `json:"enrolledStudents"`
	EnrollConsentCode        string            `json:"enrollConsentCode"`
	EnrollConsentDescription string            `json:"enrollConsentDescription"`
	DropConsentCode          string            `json:"dropConsentCode"`
	DropConsentDescription   string            `json:"dropConsentDescription"`
	ScheduleData             []ClassSchedule   `json:"scheduleData"`
	InstructorData           []ClassInstructor `json:"instructorData"`
}

type ClassSchedule struct {
	CourseID                    string `json:"courseId"`
	CourseOfferNumber           int    `json:"courseOfferNumber"`
	SessionCode                 string `json:"sessionCode"`
	ClassSection                int    `json:"classSection"`
	TermCode                    string `json:"termCode"`
	ClassMeetingNumber          int    `json:"classMeetingNumber"`
	ScheduleStartDate           string `json:"scheduleStartDate"`
	ScheduleEndDate             string `json:"scheduleEndDate"`
	ClassMeetingStartTime       string `json:"classMeetingStartTime"`
	ClassMeetingEndTime         string `json:"classMeetingEndTime"`
	ClassMeetingDayPatternCode  string `json:"classMeetingDayPatternCode"`
	ClassMeetingWeekPatternCode string `json:"classMeetingWeekPatternCode"`
	LocationName                string `json:"locationName"`
}

type ClassInstructor struct {
	CourseID                   string `json:"courseId"`
	CourseOfferNumber          int    `json:"courseOfferNumber"`
	SessionCode                string `json:"sessionCode"`
	ClassSection               int    `json:"classSection"`
	TermCode                   string `json:"termCode"`
	InstructorRoleCode         string `json:"instructorRoleCode"`
	InstructorFirstName        string `json:"instructorFirstName"`
	InstructorLastName         string `json:"instructorLastName"`
	InstructorUniqueIdentifier string `json:"instructorUniqueIdentifier"`
	ClassMeetingNumber         int    `json:"classMeetingNumber"`
}

type Exam struct {
	ExamDisplayName     string `json:"examDisplayName"`
	Sections            string `json:"sections"`
	IsOnlineDescription string `json:"isOnlineDescription"`
	Day                 string `json:"day"`
	Location            string `json:"location"`
	ExamWindowStartDate string `json:"examWindowStartDate"`
	ExamWindowStartTime string `json:"examWindowStartTime"`
	ExamDuration        string `json:"examDuration"`
	Notes               string `json:"notes"`
	TermCode            string `json:"termCode"`
}
