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
	}

	got := sampleEngineeringJobs(jobs, 3)

	if len(got) != 3 {
		t.Fatalf("expected 3 sampled jobs, got %d", len(got))
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
