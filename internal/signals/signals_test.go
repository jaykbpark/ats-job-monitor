package signals

import (
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "collapses punctuation and whitespace",
			input: "Senior Backend Engineer / Platform (Remote - US)",
			want:  "senior backend engineer platform remote us",
		},
		{
			name:  "keeps numbers and strips JSON punctuation",
			input: `{"description":"Requires 5+ years of experience"}`,
			want:  "description requires 5 years of experience",
		},
		{
			name:  "trims leading and trailing separators",
			input: "  Vancouver, British Columbia  ",
			want:  "vancouver british columbia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeText(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeriveIsRemote(t *testing.T) {
	tests := []struct {
		name         string
		location     string
		metadataJSON string
		rawJSON      string
		want         bool
	}{
		{
			name:     "matches remote in location",
			location: "Remote - US",
			want:     true,
		},
		{
			name:         "matches remote in metadata",
			location:     "San Francisco, CA",
			metadataJSON: `{"workplaceType":"remote"}`,
			want:         true,
		},
		{
			name:    "matches distributed in raw payload",
			rawJSON: `{"description":"Join our distributed engineering team"}`,
			want:    true,
		},
		{
			name:         "hybrid alone does not count as remote",
			location:     "New York, NY",
			metadataJSON: `{"workplaceType":"hybrid"}`,
			rawJSON:      `{"description":"Hybrid in-office schedule"}`,
			want:         false,
		},
		{
			name:         "explicit false isRemote beats key name",
			location:     "San Francisco, CA",
			metadataJSON: `{"isRemote":false}`,
			want:         false,
		},
		{
			name:     "explicit false in raw payload beats remote text elsewhere",
			location: "Menlo Park, CA",
			rawJSON:  `{"isRemote":false,"description":"Remote collaboration with distributed teammates"}`,
			want:     false,
		},
		{
			name:     "onsite role stays false",
			location: "Austin, TX",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveIsRemote(tt.location, tt.metadataJSON, tt.rawJSON)
			if got != tt.want {
				t.Fatalf("deriveIsRemote(%q, %q, %q) = %t, want %t", tt.location, tt.metadataJSON, tt.rawJSON, got, tt.want)
			}
		})
	}
}

func TestNormalizeEmploymentType(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full time variant",
			input: "Regular Full Time (Salary)",
			want:  "full-time",
		},
		{
			name:  "part time variant",
			input: "PartTime",
			want:  "part-time",
		},
		{
			name:  "contract variant",
			input: "Contractor",
			want:  "contract",
		},
		{
			name:  "internship variant",
			input: "Software Engineering Intern",
			want:  "internship",
		},
		{
			name:  "temporary variant",
			input: "Temp Assignment",
			want:  "temporary",
		},
		{
			name:  "unknown empty string",
			input: "",
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeEmploymentType(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeEmploymentType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeriveSeniority(t *testing.T) {
	tests := []struct {
		name      string
		titleText string
		want      string
	}{
		{
			name:      "principal role",
			titleText: "principal software engineer",
			want:      "principal",
		},
		{
			name:      "staff role",
			titleText: "staff platform engineer",
			want:      "staff",
		},
		{
			name:      "director role",
			titleText: "director of engineering",
			want:      "director",
		},
		{
			name:      "manager role",
			titleText: "engineering manager",
			want:      "manager",
		},
		{
			name:      "lead role maps to senior",
			titleText: "lead software engineer",
			want:      "senior",
		},
		{
			name:      "senior role",
			titleText: "senior backend engineer",
			want:      "senior",
		},
		{
			name:      "junior role",
			titleText: "associate software engineer",
			want:      "junior",
		},
		{
			name:      "intern role",
			titleText: "software engineering intern",
			want:      "intern",
		},
		{
			name:      "entry role",
			titleText: "new grad software engineer",
			want:      "entry",
		},
		{
			name:      "mid role",
			titleText: "mid level backend engineer",
			want:      "mid",
		},
		{
			name:      "unknown role",
			titleText: "backend engineer",
			want:      "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveSeniority(tt.titleText)
			if got != tt.want {
				t.Fatalf("deriveSeniority(%q) = %q, want %q", tt.titleText, got, tt.want)
			}
		})
	}
}

func TestDeriveExperience(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMin   *int
		wantMax   *int
		wantLevel string
	}{
		{
			name:      "range expression",
			input:     "Looking for 2-4 years experience in backend systems",
			wantMin:   intPtr(2),
			wantMax:   intPtr(4),
			wantLevel: "high",
		},
		{
			name:      "plus expression",
			input:     "Requires 5+ years of experience building backend systems",
			wantMin:   intPtr(5),
			wantLevel: "high",
		},
		{
			name:      "at least expression",
			input:     "Candidates need at least 3 years in software engineering",
			wantMin:   intPtr(3),
			wantLevel: "high",
		},
		{
			name:      "minimum of expression",
			input:     "Minimum of 7 years leading infrastructure work",
			wantMin:   intPtr(7),
			wantLevel: "high",
		},
		{
			name:      "unknown when no numeric pattern exists",
			input:     "Strong experience with backend systems required",
			wantLevel: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin, gotMax, gotLevel := deriveExperience(tt.input)

			assertIntPtrEqual(t, gotMin, tt.wantMin, "min years")
			assertIntPtrEqual(t, gotMax, tt.wantMax, "max years")

			if gotLevel != tt.wantLevel {
				t.Fatalf("deriveExperience(%q) confidence = %q, want %q", tt.input, gotLevel, tt.wantLevel)
			}
		})
	}
}

func TestDerive(t *testing.T) {
	job := providers.Job{
		Title:          "Senior Backend Engineer",
		Department:     "Engineering",
		Team:           "Platform",
		Location:       "Remote - US",
		EmploymentType: "Regular Full Time (Salary)",
		MetadataJSON:   `{"workplaceType":"remote"}`,
		RawJSON:        `{"description":"Requires 5+ years of experience building backend systems"}`,
	}

	got := Derive(job)

	if got.SearchText == "" {
		t.Fatal("expected search text to be populated")
	}

	if got.NormalizedLocation != "remote us" {
		t.Fatalf("unexpected normalized location: %q", got.NormalizedLocation)
	}

	if !got.IsRemote {
		t.Fatal("expected job to be remote")
	}

	if got.NormalizedEmploymentType != "full-time" {
		t.Fatalf("unexpected employment type: %q", got.NormalizedEmploymentType)
	}

	if got.Seniority != "senior" {
		t.Fatalf("unexpected seniority: %q", got.Seniority)
	}

	assertIntPtrEqual(t, got.MinYearsExperience, intPtr(5), "min years")
	assertIntPtrEqual(t, got.MaxYearsExperience, nil, "max years")

	if got.ExperienceConfidence != "high" {
		t.Fatalf("unexpected experience confidence: %q", got.ExperienceConfidence)
	}
}

func TestDeriveSeniorityPrefersTitleOverBodyText(t *testing.T) {
	tests := []struct {
		name      string
		job       providers.Job
		wantLevel string
	}{
		{
			name: "manager title does not become staff from body text",
			job: providers.Job{
				Title:   "Manager, Software Engineering - Billing",
				RawJSON: `{"content":"manage, support, and develop a team of engineers, including staff level engineers"}`,
			},
			wantLevel: "manager",
		},
		{
			name: "engineering manager does not become principal from body text",
			job: providers.Job{
				Title:   "Engineering Manager, CDN",
				RawJSON: `{"content":"this is a manager role, not a principal ic role"}`,
			},
			wantLevel: "manager",
		},
		{
			name: "director title does not become staff from chief of staff text",
			job: providers.Job{
				Title:   "Director, Engineering Operations",
				RawJSON: `{"content":"reporting to the chief of staff to the cto"}`,
			},
			wantLevel: "director",
		},
		{
			name: "senior title does not become director from head of engineering text",
			job: providers.Job{
				Title:   "Senior Full Stack Software Engineer",
				RawJSON: `{"descriptionPlain":"you will work closely with our head of engineering"}`,
			},
			wantLevel: "senior",
		},
		{
			name: "neutral title does not become staff from manual staff work text",
			job: providers.Job{
				Title:   "AI Engineer (Agentic/LLMs)",
				RawJSON: `{"descriptionPlain":"building an ai native system that replaces manual staff work with intelligent explainable automation"}`,
			},
			wantLevel: "unknown",
		},
		{
			name: "lead title does not become junior from mentor text",
			job: providers.Job{
				Title:   "Lead Software Engineer",
				RawJSON: `{"descriptionPlain":"they will coach and mentor junior engineers and drive engineering best practices"}`,
			},
			wantLevel: "senior",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Derive(tt.job)
			if got.Seniority != tt.wantLevel {
				t.Fatalf("Derive(%q).Seniority = %q, want %q", tt.job.Title, got.Seniority, tt.wantLevel)
			}
		})
	}
}

func assertIntPtrEqual(t *testing.T, got *int, want *int, label string) {
	t.Helper()

	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("unexpected %s: got %#v want %#v", label, got, want)
	case *got != *want:
		t.Fatalf("unexpected %s: got %d want %d", label, *got, *want)
	}
}

func intPtr(value int) *int {
	return &value
}
