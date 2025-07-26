// Dashboard Application
class Dashboard {
    constructor() {
        this.currentSection = 'dashboard';
        this.data = {
            organizations: [],
            applications: [],
            environments: [],
            cacheStats: null,
            sseStats: null
        };
        this.refreshInterval = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadInitialData();
        this.startAutoRefresh();
        this.checkSystemHealth();
    }

    setupEventListeners() {
        // Sidebar navigation
        document.querySelectorAll('.menu-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const section = e.currentTarget.dataset.section;
                this.switchSection(section);
            });
        });

        // Global refresh button
        window.refreshData = () => this.refreshCurrentSection();
        
        // Filter handlers
        window.filterApplications = () => this.filterApplications();
        window.filterEnvironments = () => this.filterEnvironments();
        window.filterConfigurations = () => this.filterConfigurations();
        
        // Cache management
        window.warmCache = () => this.warmCache();
        window.clearCache = () => this.clearCache();
    }

    switchSection(section) {
        // Update active menu item
        document.querySelectorAll('.menu-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-section="${section}"]`).classList.add('active');

        // Update active content section
        document.querySelectorAll('.content-section').forEach(section => {
            section.classList.remove('active');
        });
        document.getElementById(`${section}-section`).classList.add('active');

        // Update page title
        const titles = {
            dashboard: 'Dashboard',
            organizations: 'Organizations',
            applications: 'Applications',
            environments: 'Environments',
            configurations: 'Configurations',
            monitoring: 'System Monitoring',
            cache: 'Cache Management',
            sse: 'Real-time Connections'
        };
        document.getElementById('page-title').textContent = titles[section];

        this.currentSection = section;
        this.loadSectionData(section);
    }

    async loadInitialData() {
        try {
            await Promise.all([
                this.loadOrganizations(),
                this.loadCacheStats(),
                this.loadSSEStats()
            ]);
            this.updateDashboardStats();
        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.showError('Failed to load initial data');
        }
    }

    async loadSectionData(section) {
        switch (section) {
            case 'dashboard':
                await this.loadDashboardData();
                break;
            case 'organizations':
                await this.loadOrganizations();
                this.renderOrganizations();
                break;
            case 'applications':
                await this.loadApplications();
                this.renderApplications();
                break;
            case 'environments':
                await this.loadEnvironments();
                this.renderEnvironments();
                break;
            case 'configurations':
                this.renderConfigurations();
                break;
            case 'monitoring':
                await this.loadMonitoringData();
                break;
            case 'cache':
                await this.loadCacheStats();
                this.renderCacheDetails();
                break;
            case 'sse':
                await this.loadSSEStats();
                this.renderSSEDetails();
                break;
        }
    }

    async loadDashboardData() {
        await Promise.all([
            this.loadOrganizations(),
            this.loadCacheStats(),
            this.loadSSEStats()
        ]);
        this.updateDashboardStats();
        this.renderDashboardStats();
        this.loadRecentActivity();
    }

    async loadOrganizations() {
        try {
            const response = await API.get('/admin/orgs');
            this.data.organizations = response.data || [];
            this.updateOrgFilter();
        } catch (error) {
            console.error('Failed to load organizations:', error);
            this.data.organizations = [];
        }
    }

    async loadApplications() {
        try {
            // Load applications for all organizations
            const apps = [];
            for (const org of this.data.organizations) {
                try {
                    const response = await API.get(`/admin/orgs/${org.slug}/apps`);
                    const orgApps = (response.data || []).map(app => ({
                        ...app,
                        organization: org
                    }));
                    apps.push(...orgApps);
                } catch (error) {
                    console.error(`Failed to load apps for org ${org.slug}:`, error);
                }
            }
            this.data.applications = apps;
            this.updateAppFilter();
        } catch (error) {
            console.error('Failed to load applications:', error);
            this.data.applications = [];
        }
    }

    async loadEnvironments() {
        try {
            // Load environments for all applications
            const envs = [];
            for (const app of this.data.applications) {
                try {
                    const response = await API.get(`/admin/orgs/${app.organization.slug}/apps/${app.slug}/envs`);
                    const appEnvs = (response.data || []).map(env => ({
                        ...env,
                        application: app,
                        organization: app.organization
                    }));
                    envs.push(...appEnvs);
                } catch (error) {
                    console.error(`Failed to load envs for app ${app.slug}:`, error);
                }
            }
            this.data.environments = envs;
            this.updateEnvFilter();
        } catch (error) {
            console.error('Failed to load environments:', error);
            this.data.environments = [];
        }
    }

    async loadCacheStats() {
        try {
            const response = await API.get('/admin/cache/stats');
            this.data.cacheStats = response;
        } catch (error) {
            console.error('Failed to load cache stats:', error);
            this.data.cacheStats = null;
        }
    }

    async loadSSEStats() {
        try {
            const response = await API.get('/admin/sse/stats');
            this.data.sseStats = response;
        } catch (error) {
            console.error('Failed to load SSE stats:', error);
            this.data.sseStats = null;
        }
    }

    updateDashboardStats() {
        document.getElementById('total-orgs').textContent = this.data.organizations.length;
        document.getElementById('total-apps').textContent = this.data.applications.length;
        document.getElementById('total-envs').textContent = this.data.environments.length;
        
        const activeConnections = this.data.sseStats?.stats?.active_connections || 0;
        document.getElementById('active-connections').textContent = activeConnections;
    }

    renderDashboardStats() {
        // Render cache stats
        const cacheStatsEl = document.getElementById('cache-stats');
        if (this.data.cacheStats && this.data.cacheStats.enabled) {
            const stats = this.data.cacheStats.stats;
            const hitRatio = stats.hits + stats.misses > 0 
                ? ((stats.hits / (stats.hits + stats.misses)) * 100).toFixed(1)
                : '0.0';
            
            cacheStatsEl.innerHTML = `
                <div class="stat-row">
                    <span>Hit Ratio:</span>
                    <span class="stat-value">${hitRatio}%</span>
                </div>
                <div class="stat-row">
                    <span>Total Keys:</span>
                    <span class="stat-value">${stats.total_keys}</span>
                </div>
                <div class="stat-row">
                    <span>Hits:</span>
                    <span class="stat-value">${stats.hits}</span>
                </div>
                <div class="stat-row">
                    <span>Misses:</span>
                    <span class="stat-value">${stats.misses}</span>
                </div>
            `;
        } else {
            cacheStatsEl.innerHTML = '<div class="error">Cache not available</div>';
        }

        // Render SSE stats
        const sseStatsEl = document.getElementById('sse-stats');
        if (this.data.sseStats) {
            const stats = this.data.sseStats.stats;
            sseStatsEl.innerHTML = `
                <div class="stat-row">
                    <span>Active Connections:</span>
                    <span class="stat-value">${stats.active_connections}</span>
                </div>
                <div class="stat-row">
                    <span>Total Connections:</span>
                    <span class="stat-value">${stats.total_connections}</span>
                </div>
                <div class="stat-row">
                    <span>Messages Sent:</span>
                    <span class="stat-value">${stats.messages_sent}</span>
                </div>
                <div class="stat-row">
                    <span>Dropped:</span>
                    <span class="stat-value">${stats.connections_dropped}</span>
                </div>
            `;
        } else {
            sseStatsEl.innerHTML = '<div class="error">SSE stats not available</div>';
        }
    }

    loadRecentActivity() {
        const activityEl = document.getElementById('recent-activity');
        // For now, show placeholder activity
        activityEl.innerHTML = `
            <div class="activity-item">
                <div class="activity-icon">
                    <i class="fas fa-edit"></i>
                </div>
                <div class="activity-content">
                    <div class="activity-title">Configuration updated</div>
                    <div class="activity-meta">mycompany/webapp/prod • 2 minutes ago</div>
                </div>
            </div>
            <div class="activity-item">
                <div class="activity-icon">
                    <i class="fas fa-plus"></i>
                </div>
                <div class="activity-content">
                    <div class="activity-title">New environment created</div>
                    <div class="activity-meta">mycompany/webapp/staging • 15 minutes ago</div>
                </div>
            </div>
            <div class="activity-item">
                <div class="activity-icon">
                    <i class="fas fa-undo"></i>
                </div>
                <div class="activity-content">
                    <div class="activity-title">Configuration rolled back</div>
                    <div class="activity-meta">mycompany/api/prod • 1 hour ago</div>
                </div>
            </div>
        `;
    }

    renderOrganizations() {
        const tbody = document.querySelector('#organizations-table tbody');

        if (this.data.organizations.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4" class="loading">No organizations found</td></tr>';
            return;
        }

        tbody.innerHTML = this.data.organizations.map(org => `
            <tr>
                <td>${org.name}</td>
                <td><code>${org.slug}</code></td>
                <td>${new Date(org.created_at).toLocaleDateString()}</td>
                <td>
                    <div class="action-buttons">
                        <button class="action-btn edit" onclick="editOrganization('${org.slug}')">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="action-btn delete" onclick="deleteOrganization('${org.slug}')">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
    }

    renderApplications() {
        const tbody = document.querySelector('#applications-table tbody');

        if (this.data.applications.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">No applications found</td></tr>';
            return;
        }

        tbody.innerHTML = this.data.applications.map(app => `
            <tr>
                <td>${app.name}</td>
                <td><code>${app.slug}</code></td>
                <td>${app.organization.name}</td>
                <td><code class="api-key">${app.api_key ? app.api_key.substring(0, 20) + '...' : 'N/A'}</code></td>
                <td>${new Date(app.created_at).toLocaleDateString()}</td>
                <td>
                    <div class="action-buttons">
                        <button class="action-btn edit" onclick="editApplication('${app.organization.slug}', '${app.slug}')">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="action-btn delete" onclick="deleteApplication('${app.organization.slug}', '${app.slug}')">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
    }

    renderEnvironments() {
        const tbody = document.querySelector('#environments-table tbody');

        if (this.data.environments.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">No environments found</td></tr>';
            return;
        }

        tbody.innerHTML = this.data.environments.map(env => `
            <tr>
                <td>${env.name}</td>
                <td><code>${env.slug}</code></td>
                <td>${env.application.name}</td>
                <td>${env.organization.name}</td>
                <td>${new Date(env.created_at).toLocaleDateString()}</td>
                <td>
                    <div class="action-buttons">
                        <button class="action-btn edit" onclick="editEnvironment('${env.organization.slug}', '${env.application.slug}', '${env.slug}')">
                            <i class="fas fa-edit"></i>
                        </button>
                        <button class="action-btn delete" onclick="deleteEnvironment('${env.organization.slug}', '${env.application.slug}', '${env.slug}')">
                            <i class="fas fa-trash"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `).join('');
    }

    renderConfigurations() {
        const configContent = document.getElementById('config-content');
        const updateBtn = document.getElementById('update-config-btn');

        configContent.innerHTML = `
            <div class="config-placeholder">
                <i class="fas fa-file-code"></i>
                <p>Select an environment to view its configuration</p>
            </div>
        `;
        updateBtn.disabled = true;
    }

    async filterConfigurations() {
        const envFilter = document.getElementById('env-filter');
        const selectedEnv = envFilter.value;
        const configContent = document.getElementById('config-content');
        const updateBtn = document.getElementById('update-config-btn');

        if (!selectedEnv) {
            this.renderConfigurations();
            return;
        }

        const [orgSlug, appSlug, envSlug] = selectedEnv.split('/');

        try {
            configContent.innerHTML = '<div class="loading">Loading configuration...</div>';
            const config = await API.getConfiguration(orgSlug, appSlug, envSlug);

            if (config && config.config) {
                configContent.innerHTML = `
                    <div class="config-editor">
                        <div class="config-header">
                            <div>
                                <h4>${config.organization}/${config.application}/${config.environment}</h4>
                                <small>Version ${config.version} • Updated ${APIUtils.formatDate(config.updated_at)}</small>
                            </div>
                        </div>
                        <div class="config-content">
                            <pre class="config-json">${APIUtils.formatJSON(config.config)}</pre>
                        </div>
                    </div>
                `;
                updateBtn.disabled = false;
            } else {
                configContent.innerHTML = `
                    <div class="config-placeholder">
                        <i class="fas fa-file-code"></i>
                        <p>No configuration found for this environment</p>
                        <button class="btn btn-primary" onclick="showUpdateConfigModal()">
                            <i class="fas fa-plus"></i> Create Configuration
                        </button>
                    </div>
                `;
                updateBtn.disabled = false;
            }
        } catch (error) {
            configContent.innerHTML = `
                <div class="config-placeholder">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>Failed to load configuration</p>
                    <small>${APIUtils.formatError(error)}</small>
                </div>
            `;
            updateBtn.disabled = true;
        }
    }

    updateOrgFilter() {
        const select = document.getElementById('org-filter');
        if (!select) return;
        
        select.innerHTML = '<option value="">All Organizations</option>' +
            this.data.organizations.map(org => 
                `<option value="${org.slug}">${org.name}</option>`
            ).join('');
    }

    updateAppFilter() {
        const select = document.getElementById('app-filter');
        if (!select) return;
        
        select.innerHTML = '<option value="">All Applications</option>' +
            this.data.applications.map(app => 
                `<option value="${app.slug}">${app.name} (${app.organization.name})</option>`
            ).join('');
    }

    updateEnvFilter() {
        const select = document.getElementById('env-filter');
        if (!select) return;
        
        select.innerHTML = '<option value="">Select Environment</option>' +
            this.data.environments.map(env => 
                `<option value="${env.organization.slug}/${env.application.slug}/${env.slug}">
                    ${env.name} (${env.organization.name}/${env.application.name})
                </option>`
            ).join('');
    }

    async checkSystemHealth() {
        try {
            const response = await fetch('/health');
            const isHealthy = response.ok;
            
            const statusDot = document.getElementById('connection-status');
            const statusText = document.getElementById('connection-text');
            
            if (isHealthy) {
                statusDot.className = 'status-dot connected';
                statusText.textContent = 'System Healthy';
            } else {
                statusDot.className = 'status-dot disconnected';
                statusText.textContent = 'System Issues';
            }
        } catch (error) {
            const statusDot = document.getElementById('connection-status');
            const statusText = document.getElementById('connection-text');
            statusDot.className = 'status-dot disconnected';
            statusText.textContent = 'Connection Error';
        }
    }

    startAutoRefresh() {
        // Refresh data every 30 seconds
        this.refreshInterval = setInterval(() => {
            if (this.currentSection === 'dashboard') {
                this.loadDashboardData();
            }
            this.checkSystemHealth();
        }, 30000);
    }

    async refreshCurrentSection() {
        const refreshBtn = document.querySelector('.header .btn-primary');
        const originalText = refreshBtn.innerHTML;
        
        refreshBtn.innerHTML = '<i class="fas fa-sync-alt fa-spin"></i> Refreshing...';
        refreshBtn.disabled = true;
        
        try {
            await this.loadSectionData(this.currentSection);
        } catch (error) {
            console.error('Refresh failed:', error);
            this.showError('Failed to refresh data');
        } finally {
            refreshBtn.innerHTML = originalText;
            refreshBtn.disabled = false;
        }
    }

    showError(message) {
        // Simple error notification - could be enhanced with a proper notification system
        alert(`Error: ${message}`);
    }

    showSuccess(message) {
        // Simple success notification - could be enhanced with a proper notification system
        alert(`Success: ${message}`);
    }

    async warmCache() {
        try {
            await API.warmCache();
            this.showSuccess('Cache warming completed successfully');
            await this.loadCacheStats();
            if (this.currentSection === 'cache') {
                this.renderCacheDetails();
            }
        } catch (error) {
            this.showError('Failed to warm cache: ' + APIUtils.formatError(error));
        }
    }

    async clearCache() {
        if (!confirm('Are you sure you want to clear all cache? This action cannot be undone.')) {
            return;
        }

        try {
            await API.clearCache();
            this.showSuccess('Cache cleared successfully');
            await this.loadCacheStats();
            if (this.currentSection === 'cache') {
                this.renderCacheDetails();
            }
        } catch (error) {
            this.showError('Failed to clear cache: ' + APIUtils.formatError(error));
        }
    }

    renderCacheDetails() {
        const cacheDetails = document.getElementById('cache-details');

        if (!this.data.cacheStats || !this.data.cacheStats.enabled) {
            cacheDetails.innerHTML = `
                <div class="error">
                    <i class="fas fa-exclamation-triangle"></i>
                    <h3>Cache Not Available</h3>
                    <p>Redis cache is not enabled or not accessible.</p>
                </div>
            `;
            return;
        }

        const stats = this.data.cacheStats.stats;
        const hitRatio = stats.hits + stats.misses > 0
            ? ((stats.hits / (stats.hits + stats.misses)) * 100).toFixed(1)
            : '0.0';

        cacheDetails.innerHTML = `
            <div class="cache-stats-grid">
                <div class="cache-stat-card">
                    <h4>Performance</h4>
                    <div class="stat-row">
                        <span>Hit Ratio:</span>
                        <span class="stat-value">${hitRatio}%</span>
                    </div>
                    <div class="stat-row">
                        <span>Total Hits:</span>
                        <span class="stat-value">${stats.hits.toLocaleString()}</span>
                    </div>
                    <div class="stat-row">
                        <span>Total Misses:</span>
                        <span class="stat-value">${stats.misses.toLocaleString()}</span>
                    </div>
                </div>
                <div class="cache-stat-card">
                    <h4>Operations</h4>
                    <div class="stat-row">
                        <span>Total Keys:</span>
                        <span class="stat-value">${stats.total_keys.toLocaleString()}</span>
                    </div>
                    <div class="stat-row">
                        <span>Sets:</span>
                        <span class="stat-value">${stats.sets.toLocaleString()}</span>
                    </div>
                    <div class="stat-row">
                        <span>Deletes:</span>
                        <span class="stat-value">${stats.deletes.toLocaleString()}</span>
                    </div>
                    <div class="stat-row">
                        <span>Errors:</span>
                        <span class="stat-value">${stats.errors.toLocaleString()}</span>
                    </div>
                </div>
            </div>
        `;
    }

    renderSSEDetails() {
        const sseDetails = document.getElementById('sse-details');

        if (!this.data.sseStats) {
            sseDetails.innerHTML = `
                <div class="error">
                    <i class="fas fa-exclamation-triangle"></i>
                    <h3>SSE Stats Not Available</h3>
                    <p>Unable to load SSE connection statistics.</p>
                </div>
            `;
            return;
        }

        const stats = this.data.sseStats.stats;
        const clients = this.data.sseStats.clients || [];

        sseDetails.innerHTML = `
            <div class="sse-stats-grid">
                <div class="sse-stat-card">
                    <h4>Connection Statistics</h4>
                    <div class="stat-row">
                        <span>Active Connections:</span>
                        <span class="stat-value">${stats.active_connections}</span>
                    </div>
                    <div class="stat-row">
                        <span>Total Connections:</span>
                        <span class="stat-value">${stats.total_connections.toLocaleString()}</span>
                    </div>
                    <div class="stat-row">
                        <span>Messages Sent:</span>
                        <span class="stat-value">${stats.messages_sent.toLocaleString()}</span>
                    </div>
                    <div class="stat-row">
                        <span>Connections Dropped:</span>
                        <span class="stat-value">${stats.connections_dropped.toLocaleString()}</span>
                    </div>
                </div>
                <div class="sse-clients-card">
                    <h4>Active Clients</h4>
                    <div class="clients-list">
                        ${clients.length === 0 ?
                            '<div class="no-clients">No active connections</div>' :
                            clients.map(client => `
                                <div class="client-item">
                                    <div class="client-info">
                                        <strong>${client.organization}/${client.application}/${client.environment}</strong>
                                        <small>Connected: ${APIUtils.formatDate(client.connected_at)}</small>
                                    </div>
                                    <div class="client-id">
                                        <code>${client.id.substring(0, 8)}...</code>
                                    </div>
                                </div>
                            `).join('')
                        }
                    </div>
                </div>
            </div>
        `;
    }

    async loadMonitoringData() {
        const healthStatus = document.getElementById('health-status');
        const performanceMetrics = document.getElementById('performance-metrics');

        // Load health status
        try {
            const health = await API.healthCheck();
            healthStatus.innerHTML = `
                <div class="health-item">
                    <span>API Server</span>
                    <span class="health-status connected">Healthy</span>
                </div>
                <div class="health-item">
                    <span>Database</span>
                    <span class="health-status connected">Connected</span>
                </div>
                <div class="health-item">
                    <span>Cache</span>
                    <span class="health-status ${this.data.cacheStats?.enabled ? 'connected' : 'disconnected'}">
                        ${this.data.cacheStats?.enabled ? 'Available' : 'Unavailable'}
                    </span>
                </div>
            `;
        } catch (error) {
            healthStatus.innerHTML = `
                <div class="health-item">
                    <span>System Health</span>
                    <span class="health-status disconnected">Error</span>
                </div>
            `;
        }

        // Load performance metrics
        performanceMetrics.innerHTML = `
            <div class="metric-item">
                <span>Organizations:</span>
                <span class="metric-value">${this.data.organizations.length}</span>
            </div>
            <div class="metric-item">
                <span>Applications:</span>
                <span class="metric-value">${this.data.applications.length}</span>
            </div>
            <div class="metric-item">
                <span>Environments:</span>
                <span class="metric-value">${this.data.environments.length}</span>
            </div>
            <div class="metric-item">
                <span>Cache Hit Ratio:</span>
                <span class="metric-value">
                    ${this.data.cacheStats?.enabled ?
                        ((this.data.cacheStats.stats.hits / (this.data.cacheStats.stats.hits + this.data.cacheStats.stats.misses || 1)) * 100).toFixed(1) + '%' :
                        'N/A'
                    }
                </span>
            </div>
        `;
    }

    filterApplications() {
        // Implementation for filtering applications by organization
        // This would filter the displayed applications table
    }

    filterEnvironments() {
        // Implementation for filtering environments by application
        // This would filter the displayed environments table
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new Dashboard();
});

// Global functions for CRUD operations
window.editOrganization = async (slug) => {
    // TODO: Implement edit organization modal
    alert('Edit organization functionality coming soon');
};

window.deleteOrganization = async (slug) => {
    if (!confirm(`Are you sure you want to delete organization "${slug}"? This action cannot be undone.`)) {
        return;
    }

    try {
        await API.deleteOrganization(slug);
        dashboard.showSuccess('Organization deleted successfully');
        await dashboard.loadOrganizations();
        if (dashboard.currentSection === 'organizations') {
            dashboard.renderOrganizations();
        }
    } catch (error) {
        dashboard.showError('Failed to delete organization: ' + APIUtils.formatError(error));
    }
};

window.editApplication = async (orgSlug, appSlug) => {
    // TODO: Implement edit application modal
    alert('Edit application functionality coming soon');
};

window.deleteApplication = async (orgSlug, appSlug) => {
    if (!confirm(`Are you sure you want to delete application "${appSlug}"? This action cannot be undone.`)) {
        return;
    }

    try {
        await API.deleteApplication(orgSlug, appSlug);
        dashboard.showSuccess('Application deleted successfully');
        await dashboard.loadApplications();
        if (dashboard.currentSection === 'applications') {
            dashboard.renderApplications();
        }
    } catch (error) {
        dashboard.showError('Failed to delete application: ' + APIUtils.formatError(error));
    }
};

window.editEnvironment = async (orgSlug, appSlug, envSlug) => {
    // TODO: Implement edit environment modal
    alert('Edit environment functionality coming soon');
};

window.deleteEnvironment = async (orgSlug, appSlug, envSlug) => {
    if (!confirm(`Are you sure you want to delete environment "${envSlug}"? This action cannot be undone.`)) {
        return;
    }

    try {
        await API.deleteEnvironment(orgSlug, appSlug, envSlug);
        dashboard.showSuccess('Environment deleted successfully');
        await dashboard.loadEnvironments();
        if (dashboard.currentSection === 'environments') {
            dashboard.renderEnvironments();
        }
    } catch (error) {
        dashboard.showError('Failed to delete environment: ' + APIUtils.formatError(error));
    }
};

// Add some CSS for stat rows and additional components
const style = document.createElement('style');
style.textContent = `
    .stat-row {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 0;
        border-bottom: 1px solid #e9ecef;
    }
    .stat-row:last-child {
        border-bottom: none;
    }
    .stat-value {
        font-weight: 600;
        color: #2c3e50;
    }
    .error {
        color: #dc3545;
        font-style: italic;
        text-align: center;
        padding: 20px;
    }
    .api-key {
        font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
        font-size: 12px;
        background: #f8f9fa;
        padding: 2px 4px;
        border-radius: 3px;
    }
    .cache-stats-grid, .sse-stats-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 20px;
    }
    .cache-stat-card, .sse-stat-card, .sse-clients-card {
        background: #f8f9fa;
        padding: 20px;
        border-radius: 8px;
        border: 1px solid #e9ecef;
    }
    .cache-stat-card h4, .sse-stat-card h4, .sse-clients-card h4 {
        margin: 0 0 15px 0;
        color: #2c3e50;
        font-size: 14px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    .clients-list {
        max-height: 300px;
        overflow-y: auto;
    }
    .client-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px;
        background: white;
        border-radius: 4px;
        margin-bottom: 8px;
        border: 1px solid #e9ecef;
    }
    .client-info strong {
        display: block;
        color: #2c3e50;
        font-size: 13px;
    }
    .client-info small {
        color: #6c757d;
        font-size: 11px;
    }
    .client-id code {
        font-size: 11px;
        background: #e9ecef;
        padding: 2px 4px;
        border-radius: 3px;
    }
    .no-clients {
        text-align: center;
        color: #6c757d;
        font-style: italic;
        padding: 20px;
    }
    .metric-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px 0;
        border-bottom: 1px solid #e9ecef;
    }
    .metric-item:last-child {
        border-bottom: none;
    }
    .metric-value {
        font-weight: 600;
        color: #2c3e50;
    }
`;
document.head.appendChild(style);
