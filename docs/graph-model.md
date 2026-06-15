# Graph Model

Neo4j writes are idempotent: nodes are merged by stable identity properties,
then properties and relationships are updated. Constraints are created by
`Store.EnsureSchema`.

## Nodes

| Label | Identity property | Source |
| --- | --- | --- |
| `Term` | `termCode` | Terms |
| `Course` | `courseCode` | Courses across terms |
| `CourseOffering` | `offeringKey` | Course in a specific term/offer |
| `Subject` | `code` | Subjects |
| `AcademicOrganization` | `code` | Academic organizations |
| `Building` | `buildingCode` | Locations |
| `ClassSection` | `sectionKey` | Scheduled classes |
| `ClassMeeting` | `meetingKey` | Class schedule entries |
| `Exam` | `examKey` | Exam schedules |

Indexes exist on `Course.title` and `Building.buildingName`.

## Relationships

```mermaid
flowchart LR
    CO[CourseOffering] -->|IN_TERM| T[Term]
    CO -->|INSTANCE_OF| C[Course]
    CO -->|IN_SUBJECT| S[Subject]
    CO -->|OWNED_BY| A[AcademicOrganization]
    S -->|PART_OF| A
    B[Building] -->|PART_OF| PB[Parent Building]
    CS[ClassSection] -->|SECTION_OF| CO
    CM[ClassMeeting] -->|MEETING_OF| CS
    E[Exam] -->|IN_TERM| T
```

The current API does not provide a reliable normalized building identifier for
every class meeting or exam location, so those records keep location text
rather than creating a building relationship.

## Identity Keys

Key construction is centralized in `internal/graph`:

- `courseCode`: trimmed subject and catalog number, separated by one space.
- `offeringKey`: `termCode|courseID|courseOfferNumber`.
- `sectionKey`:
  `termCode|courseID|courseOfferNumber|sessionCode|classSection|classNumber`.
- `meetingKey`: `sectionKey|classMeetingNumber`.
- `examKey`: `exam:` plus the SHA-1 digest of trimmed term, display name,
  sections, start date, and start time joined with `|`.

SHA-1 is used here only as a deterministic identifier, not for security.
Changing any formula can create duplicate nodes and orphan existing
relationships.

## Schema Change Checklist

When adding a dataset, node, property, or relationship:

1. Update Waterloo models/client mappings and fixture tests.
2. Add or update key helpers and key tests.
3. Add constraints or indexes before writes that depend on them.
4. Keep all Cypher data in parameters and preserve batching.
5. Extend the tagged Neo4j integration test beyond schema creation where the
   new behavior warrants it.
6. Update this document in the same change.
