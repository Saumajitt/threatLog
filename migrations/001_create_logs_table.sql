-- Create logs table
CREATE TABLE IF NOT EXISTS logs (
    id UUID PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    severity VARCHAR(20) NOT NULL,
    source VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    ingested_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logs_severity ON logs(severity);
CREATE INDEX IF NOT EXISTS idx_logs_source ON logs(source);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp_severity ON logs(timestamp DESC, severity);

-- Create a function to validate severity
CREATE OR REPLACE FUNCTION validate_severity()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.severity NOT IN ('CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO') THEN
        RAISE EXCEPTION 'Invalid severity level: %', NEW.severity;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
CREATE TRIGGER check_severity
    BEFORE INSERT OR UPDATE ON logs
    FOR EACH ROW
    EXECUTE FUNCTION validate_severity();