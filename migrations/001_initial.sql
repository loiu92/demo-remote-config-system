-- Remote Configuration System Database Schema
-- This file creates the initial database structure

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Applications table
CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(org_id, slug)
);

-- Environments table
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(app_id, slug)
);

-- Configuration versions table
CREATE TABLE config_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    config_json JSONB NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    UNIQUE(env_id, version)
);

-- Configuration change log table
CREATE TABLE config_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    env_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    version_from INTEGER,
    version_to INTEGER NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'create', 'update', 'rollback'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255)
);

-- Indexes for better performance
CREATE INDEX idx_applications_org_id ON applications(org_id);
CREATE INDEX idx_applications_api_key ON applications(api_key);
CREATE INDEX idx_environments_app_id ON environments(app_id);
CREATE INDEX idx_config_versions_env_id ON config_versions(env_id);
CREATE INDEX idx_config_versions_active ON config_versions(env_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_config_changes_env_id ON config_changes(env_id);
CREATE INDEX idx_config_changes_created_at ON config_changes(created_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_applications_updated_at BEFORE UPDATE ON applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON environments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to ensure only one active config per environment
CREATE OR REPLACE FUNCTION ensure_single_active_config()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_active = TRUE THEN
        -- Deactivate all other configs for this environment
        UPDATE config_versions 
        SET is_active = FALSE 
        WHERE env_id = NEW.env_id AND id != NEW.id;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to ensure only one active config per environment
CREATE TRIGGER ensure_single_active_config_trigger 
    BEFORE INSERT OR UPDATE ON config_versions
    FOR EACH ROW EXECUTE FUNCTION ensure_single_active_config();

-- Insert sample data for development
INSERT INTO organizations (name, slug) VALUES 
    ('Demo Organization', 'demo-org'),
    ('Test Company', 'test-company');

INSERT INTO applications (org_id, name, slug, api_key) VALUES 
    ((SELECT id FROM organizations WHERE slug = 'demo-org'), 'Demo Application', 'demo-app', 'demo-app-key-12345'),
    ((SELECT id FROM organizations WHERE slug = 'demo-org'), 'Web Dashboard', 'web-dashboard', 'web-dashboard-key-67890'),
    ((SELECT id FROM organizations WHERE slug = 'test-company'), 'Test App', 'test-app', 'test-app-key-abcde');

INSERT INTO environments (app_id, name, slug) VALUES 
    ((SELECT id FROM applications WHERE slug = 'demo-app'), 'Production', 'production'),
    ((SELECT id FROM applications WHERE slug = 'demo-app'), 'Staging', 'staging'),
    ((SELECT id FROM applications WHERE slug = 'demo-app'), 'Development', 'development'),
    ((SELECT id FROM applications WHERE slug = 'web-dashboard'), 'Production', 'production'),
    ((SELECT id FROM applications WHERE slug = 'test-app'), 'Production', 'production');

-- Insert sample configurations
INSERT INTO config_versions (env_id, version, config_json, is_active, created_by) VALUES 
    (
        (SELECT e.id FROM environments e 
         JOIN applications a ON e.app_id = a.id 
         WHERE a.slug = 'demo-app' AND e.slug = 'production'),
        1,
        '{
            "maintenance": {
                "enabled": false,
                "message": "Scheduled maintenance in progress"
            },
            "features": {
                "dark_mode": true,
                "new_dashboard": false,
                "beta_features": true
            },
            "ui": {
                "theme_color": "#007bff",
                "max_items_per_page": 20,
                "show_footer": true
            },
            "limits": {
                "max_upload_size_mb": 10,
                "rate_limit_per_hour": 1000
            }
        }'::jsonb,
        TRUE,
        'system'
    ),
    (
        (SELECT e.id FROM environments e 
         JOIN applications a ON e.app_id = a.id 
         WHERE a.slug = 'demo-app' AND e.slug = 'staging'),
        1,
        '{
            "maintenance": {
                "enabled": false,
                "message": "Staging environment"
            },
            "features": {
                "dark_mode": true,
                "new_dashboard": true,
                "beta_features": true
            },
            "ui": {
                "theme_color": "#28a745",
                "max_items_per_page": 50,
                "show_footer": true
            },
            "limits": {
                "max_upload_size_mb": 50,
                "rate_limit_per_hour": 5000
            }
        }'::jsonb,
        TRUE,
        'system'
    );

-- Log the initial configuration creation
INSERT INTO config_changes (env_id, version_from, version_to, action, created_by)
SELECT 
    env_id,
    NULL,
    version,
    'create',
    'system'
FROM config_versions 
WHERE created_by = 'system';
