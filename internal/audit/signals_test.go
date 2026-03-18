package audit

import (
	"testing"

	"github.com/jaykbpark/ats-job-monitor/internal/providers"
	"github.com/jaykbpark/ats-job-monitor/internal/signals"
)

func TestSampleEngineeringJobs(t *testing.T) {
	jobs := []providers.Job{
		{ExternalJobID: "1", Title: "Solutions Engineer", Department: "Go To Market"},
		{ExternalJobID: "2", Title: "Senior Backend Engineer", Department: "Engineering"},
		{ExternalJobID: "3", Title: "Product Designer", Department: "Design"},
		{ExternalJobID: "4", Title: "Platform Developer", Team: "Infrastructure"},
		{ExternalJobID: "5", Title: "Account Executive, Platforms", Department: "Sales"},
		{ExternalJobID: "6", Title: "Developer Advocacy Director", Department: "Marketing"},
		{ExternalJobID: "7", Title: "Senior Security Engineer", Department: "Engineering"},
		{ExternalJobID: "8", Title: "Director, Safety & Physical Security", Department: "Corporate Security"},
		{ExternalJobID: "9", Title: "Developer Success Engineer", Department: "Field Engineering"},
		{ExternalJobID: "10", Title: "Enablement Systems Engineer", Department: "GTM Enablement"},
		{ExternalJobID: "11", Title: "Engineering Manager, CDN", Department: "Engineering"},
		{ExternalJobID: "12", Title: "Technical Program Manager, Compute Infrastructure", Department: "Technical Program Management"},
		{ExternalJobID: "13", Title: "Lead Software Engineer", Department: "Engineering"},
		{ExternalJobID: "14", Title: "Cloud Network Engineer - Security Clearance Required", Department: "Engineering"},
		{ExternalJobID: "15", Title: "AI Applications Ops Lead, GPS", Department: "IPS Engineering"},
		{ExternalJobID: "16", Title: "Systems Engineer", Department: "Engineering"},
	}

	got := sampleEngineeringJobs(jobs, 6)

	if len(got) != 6 {
		t.Fatalf("expected 6 sampled jobs, got %d", len(got))
	}

	if got[0].ExternalJobID != "2" {
		t.Fatalf("unexpected first sampled job: %q", got[0].ExternalJobID)
	}

	if got[1].ExternalJobID != "4" {
		t.Fatalf("unexpected second sampled job: %q", got[1].ExternalJobID)
	}

	if got[2].ExternalJobID != "7" {
		t.Fatalf("unexpected third sampled job: %q", got[2].ExternalJobID)
	}

	if got[3].ExternalJobID != "11" {
		t.Fatalf("unexpected fourth sampled job: %q", got[3].ExternalJobID)
	}

	if got[4].ExternalJobID != "13" {
		t.Fatalf("unexpected fifth sampled job: %q", got[4].ExternalJobID)
	}

	if got[5].ExternalJobID != "14" {
		t.Fatalf("unexpected sixth sampled job: %q", got[5].ExternalJobID)
	}
}

func TestIsEngineeringJob(t *testing.T) {
	tests := []struct {
		name string
		job  providers.Job
		want bool
	}{
		{
			name: "includes backend engineer",
			job:  providers.Job{Title: "Senior Backend Engineer", Department: "Engineering"},
			want: true,
		},
		{
			name: "includes software engineering manager",
			job:  providers.Job{Title: "Engineering Manager, CDN", Department: "Engineering"},
			want: true,
		},
		{
			name: "includes lead software engineer",
			job:  providers.Job{Title: "Lead Software Engineer", Department: "Engineering"},
			want: true,
		},
		{
			name: "includes cloud network engineer",
			job:  providers.Job{Title: "Cloud Network Engineer - Security Clearance Required", Department: "Engineering"},
			want: true,
		},
		{
			name: "excludes solutions engineer",
			job:  providers.Job{Title: "Solutions Engineer", Department: "Go To Market"},
			want: false,
		},
		{
			name: "excludes developer success engineer",
			job:  providers.Job{Title: "Developer Success Engineer", Department: "Field Engineering"},
			want: false,
		},
		{
			name: "excludes enablement systems engineer",
			job:  providers.Job{Title: "Enablement Systems Engineer", Department: "GTM Enablement"},
			want: false,
		},
		{
			name: "excludes technical program manager",
			job:  providers.Job{Title: "Technical Program Manager, Compute Infrastructure", Department: "Technical Program Management"},
			want: false,
		},
		{
			name: "excludes operations lead",
			job:  providers.Job{Title: "AI Applications Ops Lead, GPS", Department: "IPS Engineering"},
			want: false,
		},
		{
			name: "excludes systems engineer by default",
			job:  providers.Job{Title: "Systems Engineer", Department: "Engineering"},
			want: false,
		},
		{
			name: "excludes pre sales solutions architect",
			job:  providers.Job{Title: "Pre-Sales Solutions Architect", Department: "Sales & Success"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEngineeringJob(tt.job)
			if got != tt.want {
				t.Fatalf("isEngineeringJob(%q) = %t, want %t", tt.job.Title, got, tt.want)
			}
		})
	}
}

func TestBuildChecksUsesProviderGroundTruth(t *testing.T) {
	job := providers.Job{
		Title:          "Senior Backend Engineer",
		Location:       "Remote - US",
		EmploymentType: "FullTime",
		MetadataJSON:   `{"isRemote":true,"workplaceType":"Remote"}`,
		RawJSON:        `{"descriptionHtml":"<p>Requires 5+ years of experience</p>"}`,
	}

	derived := signals.Derive(job)
	checks := buildChecks("ashby", job, derived)

	assertCheckStatus(t, checks, "search_text_populated", "pass")
	assertCheckStatus(t, checks, "normalized_location_populated", "pass")
	assertCheckStatus(t, checks, "employment_type_captured", "pass")
	assertCheckStatus(t, checks, "evidence_text_present", "pass")
	assertCheckStatus(t, checks, "remote_matches_provider_signal", "pass")
	assertCheckStatus(t, checks, "seniority_matches_title", "pass")
	assertCheckStatus(t, checks, "experience_matches_evidence", "pass")
}

func TestExtractEvidenceTextStripsHTML(t *testing.T) {
	job := providers.Job{
		RawJSON: `{"descriptionHtml":"<p>Requires 5+ years &amp; strong Go experience.</p>"}`,
	}

	got := extractEvidenceText(job)
	want := "Requires 5+ years & strong Go experience."
	if got != want {
		t.Fatalf("extractEvidenceText() = %q, want %q", got, want)
	}
}

func TestBuildChecksUsesGreenhouseContentForEvidence(t *testing.T) {
	job := providers.Job{
		Title:        "Software Engineer",
		Location:     "Remote",
		Department:   "Engineering",
		RawJSON:      `{"content":"&lt;p&gt;Candidates need at least 4 years in backend systems.&lt;/p&gt;"}`,
		MetadataJSON: `{}`,
	}

	derived := signals.Derive(job)
	checks := buildChecks("greenhouse", job, derived)

	assertCheckStatus(t, checks, "evidence_text_present", "pass")
	assertCheckStatus(t, checks, "experience_matches_evidence", "pass")
}

func assertCheckStatus(t *testing.T, checks []SignalAuditCheck, name string, want string) {
	t.Helper()

	for _, check := range checks {
		if check.Name != name {
			continue
		}
		if check.Status != want {
			t.Fatalf("check %q status = %q, want %q", name, check.Status, want)
		}
		return
	}

	t.Fatalf("check %q not found", name)
}
