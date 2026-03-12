package providers

type Job struct {
	ExternalJobID  string
	Title          string
	Location       string
	Department     string
	Team           string
	EmploymentType string
	JobURL         string
	MetadataJSON   string
	RawJSON        string
}
