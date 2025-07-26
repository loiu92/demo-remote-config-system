package db

// Repositories holds all repository instances
type Repositories struct {
	Organizations  *OrganizationRepository
	Applications   *ApplicationRepository
	Environments   *EnvironmentRepository
	ConfigVersions *ConfigVersionRepository
	ConfigChanges  *ConfigChangeRepository
}

// NewRepositories creates a new repositories instance
func NewRepositories(db *DB) *Repositories {
	return &Repositories{
		Organizations:  NewOrganizationRepository(db),
		Applications:   NewApplicationRepository(db),
		Environments:   NewEnvironmentRepository(db),
		ConfigVersions: NewConfigVersionRepository(db),
		ConfigChanges:  NewConfigChangeRepository(db),
	}
}
