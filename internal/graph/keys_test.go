package graph

import "testing"

func TestCourseCode(t *testing.T) {
	if got, want := CourseCode(" CS ", " 246 "), "CS 246"; got != want {
		t.Fatalf("CourseCode = %q, want %q", got, want)
	}
	if got := CourseCode("", "246"); got != "" {
		t.Fatalf("CourseCode with missing subject = %q, want empty", got)
	}
}

func TestOfferingKey(t *testing.T) {
	if got, want := OfferingKey("1251", "012345", 1), "1251|012345|1"; got != want {
		t.Fatalf("OfferingKey = %q, want %q", got, want)
	}
}

func TestSectionKey(t *testing.T) {
	got := SectionKey("1251", "012345", 1, "1", 101, 6543)
	want := "1251|012345|1|1|101|6543"
	if got != want {
		t.Fatalf("SectionKey = %q, want %q", got, want)
	}
}

func TestMeetingKey(t *testing.T) {
	if got, want := MeetingKey("1251|012345|1|1|101|6543", 2), "1251|012345|1|1|101|6543|2"; got != want {
		t.Fatalf("MeetingKey = %q, want %q", got, want)
	}
}

func TestExamKeyIsStableHash(t *testing.T) {
	first := ExamKey("1251", "CS 246", "LEC 001", "2025-04-12", "09:00")
	second := ExamKey("1251", "CS 246", "LEC 001", "2025-04-12", "09:00")
	if first != second {
		t.Fatalf("ExamKey not stable: %q != %q", first, second)
	}
	if len(first) != len("exam:")+40 {
		t.Fatalf("ExamKey length = %d, want %d", len(first), len("exam:")+40)
	}
}

func TestInstructorKeyDoesNotExposeIdentifier(t *testing.T) {
	key := InstructorKey("private-identifier")
	if key == "" {
		t.Fatal("InstructorKey returned an empty key")
	}
	if key == "private-identifier" {
		t.Fatal("InstructorKey exposed the upstream identifier")
	}
	if got := InstructorKey(" private-identifier "); got != key {
		t.Fatalf("trimmed InstructorKey = %q, want %q", got, key)
	}
	if got := InstructorKey(" "); got != "" {
		t.Fatalf("empty InstructorKey = %q, want empty", got)
	}
}
