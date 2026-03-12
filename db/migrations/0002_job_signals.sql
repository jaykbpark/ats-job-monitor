ALTER TABLE jobs ADD COLUMN search_text TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN normalized_location TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN is_remote INTEGER NOT NULL DEFAULT 0;
ALTER TABLE jobs ADD COLUMN normalized_employment_type TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE jobs ADD COLUMN seniority TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE jobs ADD COLUMN min_years_experience INTEGER;
ALTER TABLE jobs ADD COLUMN max_years_experience INTEGER;
ALTER TABLE jobs ADD COLUMN experience_confidence TEXT NOT NULL DEFAULT 'unknown';
