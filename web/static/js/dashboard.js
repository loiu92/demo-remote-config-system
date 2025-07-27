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
            this.showLoadingState('Loading dashboard data...');

            // Load organizations first (required for other data)
            await this.loadOrganizations();

            // Then load applications and environments sequentially
            if (this.data.organizations.length > 0) {
                await this.loadApplications();
                if (this.data.applications.length > 0) {
                    await this.loadEnvironments();
                }
            }

            // Load stats in parallel (independent of org data)
            await Promise.all([
                this.loadCacheStats(),
                this.loadSSEStats()
            ]);

            this.updateDashboardStats();
            this.renderDashboardStats();
            this.loadRecentActivity();
            this.hideLoadingState();
        } catch (error) {
            console.error('Failed to load initial data:', error);
            this.hideLoadingState();
            this.showError('Failed to load initial data. Please refresh the page.');
        }
    }

    async loadSectionData(section) {
        try {
            switch (section) {
                case 'dashboard':
                    this.resetDashboardToLoading();
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
        } catch (error) {
            console.error(`Failed to load section data for ${section}:`, error);
        }
    }

    async loadDashboardData() {
        try {
            // Load dashboard-specific data in parallel
            await Promise.all([
                this.loadCacheStats(),
                this.loadSSEStats()
            ]);
            this.updateDashboardStats();
            this.renderDashboardStats();
            this.loadRecentActivity();
        } catch (error) {
            console.error('Failed to load dashboard data:', error);
            this.showDashboardError(error);
        }
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
            // Ensure organizations are loaded first
            if (this.data.organizations.length === 0) {
                console.warn('No organizations loaded, skipping applications');
                this.data.applications = [];
                return;
            }

            // Load applications for all organizations
            const apps = [];
            const loadPromises = this.data.organizations.map(async (org) => {
                try {
                    const response = await API.get(`/admin/orgs/${org.slug}/apps`);
                    const orgApps = (response.data || []).map(app => ({
                        ...app,
                        organization: org
                    }));
                    return orgApps;
                } catch (error) {
                    console.error(`Failed to load apps for org ${org.slug}:`, error);
                    return [];
                }
            });

            const results = await Promise.all(loadPromises);
            results.forEach(orgApps => apps.push(...orgApps));

            this.data.applications = apps;
            this.updateAppFilter();
        } catch (error) {
            console.error('Failed to load applications:', error);
            this.data.applications = [];
        }
    }

    async loadEnvironments() {
        try {
            // Ensure applications are loaded first
            if (this.data.applications.length === 0) {
                console.warn('No applications loaded, skipping environments');
                this.data.environments = [];
                return;
            }

            // Load environments for all applications
            const envs = [];
            const loadPromises = this.data.applications.map(async (app) => {
                try {
                    const response = await API.get(`/admin/orgs/${app.organization.slug}/apps/${app.slug}/envs`);
                    const appEnvs = (response.data || []).map(env => ({
                        ...env,
                        application: app,
                        organization: app.organization
                    }));
                    return appEnvs;
                } catch (error) {
                    console.error(`Failed to load envs for app ${app.slug}:`, error);
                    return [];
                }
            });

            const results = await Promise.all(loadPromises);
            results.forEach(appEnvs => envs.push(...appEnvs));

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
            const totalKeys = this.data.cacheStats.total_keys; // total_keys is at root level
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
                    <span class="stat-value">${totalKeys}</span>
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
        if (this.data.sseStats && this.data.sseStats.stats) {
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

    resetDashboardToLoading() {
        // Reset dashboard elements to loading state
        const cacheStatsEl = document.getElementById('cache-stats');
        if (cacheStatsEl) {
            cacheStatsEl.innerHTML = '<div class="loading-spinner">Loading...</div>';
        }

        const sseStatsEl = document.getElementById('sse-stats');
        if (sseStatsEl) {
            sseStatsEl.innerHTML = '<div class="loading-spinner">Loading...</div>';
        }

        const activityEl = document.getElementById('recent-activity');
        if (activityEl) {
            activityEl.innerHTML = '<div class="loading-spinner">Loading...</div>';
        }

        // Reset stats to loading state
        document.getElementById('total-orgs').textContent = '-';
        document.getElementById('total-apps').textContent = '-';
        document.getElementById('total-envs').textContent = '-';
        document.getElementById('active-connections').textContent = '-';
    }



    showDashboardError(error) {
        // Show error in cache stats
        const cacheStatsEl = document.getElementById('cache-stats');
        if (cacheStatsEl) {
            cacheStatsEl.innerHTML = '<div class="error">Failed to load cache stats</div>';
        }

        // Show error in SSE stats
        const sseStatsEl = document.getElementById('sse-stats');
        if (sseStatsEl) {
            sseStatsEl.innerHTML = '<div class="error">Failed to load SSE stats</div>';
        }

        // Show error in recent activity
        const activityEl = document.getElementById('recent-activity');
        if (activityEl) {
            activityEl.innerHTML = '<div class="error">Failed to load recent activity</div>';
        }
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
                            <div class="config-actions">
                                <button class="btn btn-secondary" onclick="showConfigHistory('${orgSlug}', '${appSlug}', '${envSlug}')">
                                    <i class="fas fa-history"></i> History
                                </button>
                            </div>
                        </div>
                        <div class="config-tabs">
                            <button class="tab-btn active" onclick="switchConfigTab('current')">Current</button>
                            <button class="tab-btn" onclick="switchConfigTab('history')">History</button>
                        </div>
                        <div class="config-content">
                            <div id="current-config" class="tab-content active">
                                <pre class="config-json">${APIUtils.formatJSON(config.config)}</pre>
                            </div>
                            <div id="history-config" class="tab-content">
                                <div class="loading">Loading history...</div>
                            </div>
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
        if (!refreshBtn) return;

        const originalText = refreshBtn.innerHTML;
        refreshBtn.innerHTML = '<i class="fas fa-sync-alt fa-spin"></i> Refreshing...';
        refreshBtn.disabled = true;

        try {
            // For dashboard section, reload all data sequentially
            if (this.currentSection === 'dashboard') {
                await this.loadInitialData();
            } else {
                await this.loadSectionData(this.currentSection);
            }
            this.showSuccess('Data refreshed successfully');
        } catch (error) {
            console.error('Refresh failed:', error);
            this.showError('Failed to refresh data: ' + (error.message || 'Unknown error'));
        } finally {
            refreshBtn.innerHTML = originalText;
            refreshBtn.disabled = false;
        }
    }

    showLoadingState(message = 'Loading...') {
        let loadingDiv = document.getElementById('loading-overlay');
        if (!loadingDiv) {
            loadingDiv = document.createElement('div');
            loadingDiv.id = 'loading-overlay';
            loadingDiv.className = 'loading-overlay';
            loadingDiv.innerHTML = `
                <div class="loading-content">
                    <div class="loading-spinner"></div>
                    <div class="loading-message">${message}</div>
                </div>
            `;
            document.body.appendChild(loadingDiv);
        } else {
            loadingDiv.querySelector('.loading-message').textContent = message;
        }
        loadingDiv.style.display = 'flex';
    }

    hideLoadingState() {
        const loadingDiv = document.getElementById('loading-overlay');
        if (loadingDiv) {
            loadingDiv.style.display = 'none';
        }
    }

    showError(message) {
        console.error('Dashboard Error:', message);

        // Create or update error notification
        let errorDiv = document.getElementById('error-notification');
        if (!errorDiv) {
            errorDiv = document.createElement('div');
            errorDiv.id = 'error-notification';
            errorDiv.className = 'error-notification';
            document.body.appendChild(errorDiv);
        }

        errorDiv.innerHTML = `
            <div class="error-content">
                <i class="fas fa-exclamation-triangle"></i>
                <span>${message}</span>
                <button onclick="this.parentElement.parentElement.remove()" class="close-btn">×</button>
            </div>
        `;

        errorDiv.style.display = 'block';

        // Auto-hide after 8 seconds
        setTimeout(() => {
            if (errorDiv && errorDiv.parentNode) {
                errorDiv.remove();
            }
        }, 8000);
    }

    showSuccess(message) {
        console.log('Dashboard Success:', message);

        // Create or update success notification
        let successDiv = document.getElementById('success-notification');
        if (!successDiv) {
            successDiv = document.createElement('div');
            successDiv.id = 'success-notification';
            successDiv.className = 'success-notification';
            document.body.appendChild(successDiv);
        }

        successDiv.innerHTML = `
            <div class="success-content">
                <i class="fas fa-check-circle"></i>
                <span>${message}</span>
                <button onclick="this.parentElement.parentElement.remove()" class="close-btn">×</button>
            </div>
        `;

        successDiv.style.display = 'block';

        // Auto-hide after 4 seconds
        setTimeout(() => {
            if (successDiv && successDiv.parentNode) {
                successDiv.remove();
            }
        }, 4000);
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
    /* Loading overlay */
    .loading-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        display: none;
        justify-content: center;
        align-items: center;
        z-index: 10000;
    }
    .loading-content {
        background: white;
        padding: 30px;
        border-radius: 8px;
        text-align: center;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
    }
    .loading-spinner {
        width: 40px;
        height: 40px;
        border: 4px solid #f3f3f3;
        border-top: 4px solid #007bff;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin: 0 auto 15px;
    }
    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
    .loading-message {
        color: #333;
        font-size: 16px;
        font-weight: 500;
    }

    /* Error notification */
    .error-notification {
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 9999;
        max-width: 400px;
    }
    .error-content {
        background: #f8d7da;
        color: #721c24;
        border: 1px solid #f5c6cb;
        border-radius: 6px;
        padding: 12px 16px;
        display: flex;
        align-items: center;
        gap: 10px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    }
    .error-content i {
        color: #dc3545;
        font-size: 18px;
    }
    .error-content .close-btn {
        background: none;
        border: none;
        color: #721c24;
        font-size: 20px;
        cursor: pointer;
        margin-left: auto;
        padding: 0;
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .error-content .close-btn:hover {
        background: rgba(0, 0, 0, 0.1);
        border-radius: 50%;
    }

    /* Success notification */
    .success-notification {
        position: fixed;
        top: 20px;
        right: 20px;
        z-index: 9999;
        max-width: 400px;
    }
    .success-content {
        background: #d4edda;
        color: #155724;
        border: 1px solid #c3e6cb;
        border-radius: 6px;
        padding: 12px 16px;
        display: flex;
        align-items: center;
        gap: 10px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    }
    .success-content i {
        color: #28a745;
        font-size: 18px;
    }
    .success-content .close-btn {
        background: none;
        border: none;
        color: #155724;
        font-size: 20px;
        cursor: pointer;
        margin-left: auto;
        padding: 0;
        width: 24px;
        height: 24px;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .success-content .close-btn:hover {
        background: rgba(0, 0, 0, 0.1);
        border-radius: 50%;
    }

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
    /* Configuration History Styles */
    .config-tabs {
        display: flex;
        border-bottom: 1px solid #dee2e6;
        margin-bottom: 20px;
    }
    .tab-btn {
        background: none;
        border: none;
        padding: 12px 20px;
        cursor: pointer;
        border-bottom: 2px solid transparent;
        color: #6c757d;
        font-weight: 500;
    }
    .tab-btn.active {
        color: #007bff;
        border-bottom-color: #007bff;
    }
    .tab-btn:hover {
        color: #007bff;
    }
    .tab-content {
        display: none;
    }
    .tab-content.active {
        display: block;
    }
    .history-list {
        max-height: 600px;
        overflow-y: auto;
    }
    .history-item {
        border: 1px solid #dee2e6;
        border-radius: 6px;
        margin-bottom: 15px;
        background: white;
    }
    .history-item.active {
        border-color: #28a745;
        background: #f8fff9;
    }
    .history-header {
        padding: 15px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom: 1px solid #dee2e6;
    }
    .version-info {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    .version-number {
        font-weight: 600;
        color: #2c3e50;
    }
    .active-badge {
        background: #28a745;
        color: white;
        padding: 2px 8px;
        border-radius: 12px;
        font-size: 11px;
        font-weight: 600;
    }
    .version-meta {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
    }
    .created-date {
        color: #6c757d;
        font-size: 14px;
    }
    .created-by {
        color: #6c757d;
        font-size: 12px;
    }
    .version-actions {
        display: flex;
        gap: 8px;
    }
    .version-config {
        padding: 15px;
        background: #f8f9fa;
        border-top: 1px solid #dee2e6;
    }
    .config-actions {
        display: flex;
        gap: 10px;
    }
    .warning-text {
        color: #856404;
        background: #fff3cd;
        border: 1px solid #ffeaa7;
        border-radius: 4px;
        padding: 10px;
        margin: 10px 0;
    }
    .warning-text i {
        margin-right: 8px;
    }
    .info-state {
        text-align: center;
        padding: 20px;
        background: #f8f9fa;
        border: 1px solid #dee2e6;
        border-radius: 6px;
        color: #495057;
    }
    .info-state i {
        color: #17a2b8;
        font-size: 24px;
        margin-bottom: 10px;
    }
    .info-state h4 {
        color: #2c3e50;
        margin: 10px 0;
    }
    .info-state .note {
        background: #e7f3ff;
        border: 1px solid #b8daff;
        border-radius: 4px;
        padding: 10px;
        margin: 15px 0;
        font-size: 14px;
    }
    .info-state .suggestion {
        background: #f8f9fa;
        border: 1px solid #dee2e6;
        border-radius: 4px;
        padding: 10px;
        margin: 15px 0;
        font-size: 13px;
        text-align: left;
    }
    .info-state code {
        background: #f1f3f4;
        padding: 2px 4px;
        border-radius: 3px;
        font-family: 'Courier New', monospace;
        font-size: 12px;
    }
    .error-state {
        text-align: center;
        padding: 20px;
        background: #f8d7da;
        border: 1px solid #f5c6cb;
        border-radius: 6px;
        color: #721c24;
    }
    .error-state i {
        color: #dc3545;
        font-size: 24px;
        margin-bottom: 10px;
    }
    .error-details {
        font-size: 14px;
        color: #6c757d;
        margin-top: 10px;
    }
`;
document.head.appendChild(style);

// Configuration History and Rollback Functions
window.switchConfigTab = function(tab) {
    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelector(`.tab-btn[onclick="switchConfigTab('${tab}')"]`).classList.add('active');

    // Update tab content
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    document.getElementById(`${tab}-config`).classList.add('active');

    // Load history if switching to history tab
    if (tab === 'history') {
        const envFilter = document.getElementById('env-filter');
        const selectedEnv = envFilter.value;
        if (selectedEnv) {
            const [orgSlug, appSlug, envSlug] = selectedEnv.split('/');
            loadConfigHistory(orgSlug, appSlug, envSlug);
        }
    }
};

window.showConfigHistory = function(orgSlug, appSlug, envSlug) {
    switchConfigTab('history');
    loadConfigHistory(orgSlug, appSlug, envSlug);
};

async function loadConfigHistory(orgSlug, appSlug, envSlug) {
    const historyContent = document.getElementById('history-config');

    try {
        historyContent.innerHTML = '<div class="loading">Loading configuration history...</div>';

        const response = await API.get(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}/history`);
        const history = response.data || [];

        // Clear any cached version data
        window.configVersionCache = {};

        if (history.length === 0) {
            historyContent.innerHTML = `
                <div class="no-data">
                    <i class="fas fa-history"></i>
                    <p>No configuration history found</p>
                </div>
            `;
            return;
        }

        historyContent.innerHTML = `
            <div class="history-list">
                ${history.map(version => `
                    <div class="history-item ${version.is_active ? 'active' : ''}">
                        <div class="history-header">
                            <div class="version-info">
                                <span class="version-number">Version ${version.version}</span>
                                ${version.is_active ? '<span class="active-badge">ACTIVE</span>' : ''}
                            </div>
                            <div class="version-meta">
                                <span class="created-date">${APIUtils.formatDate(version.created_at)}</span>
                                ${version.created_by ? `<span class="created-by">by ${version.created_by}</span>` : ''}
                            </div>
                            <div class="version-actions">
                                <button class="btn btn-sm btn-secondary" onclick="toggleVersionConfig('${orgSlug}', '${appSlug}', '${envSlug}', ${version.version})">
                                    <i class="fas fa-eye"></i> View
                                </button>
                                ${!version.is_active ? `
                                    <button class="btn btn-sm btn-warning" onclick="rollbackToVersion('${orgSlug}', '${appSlug}', '${envSlug}', ${version.version})">
                                        <i class="fas fa-undo"></i> Rollback
                                    </button>
                                ` : ''}
                            </div>
                        </div>
                        <div id="version-config-${version.version}" class="version-config" style="display: none;">
                            <div class="config-placeholder">Click "View" to load configuration</div>
                        </div>
                    </div>
                `).join('')}
            </div>
        `;
    } catch (error) {
        console.error('Failed to load config history:', error);
        historyContent.innerHTML = `
            <div class="error-state">
                <i class="fas fa-exclamation-triangle"></i>
                <p>Failed to load configuration history</p>
                <button class="btn btn-secondary" onclick="loadConfigHistory('${orgSlug}', '${appSlug}', '${envSlug}')">
                    <i class="fas fa-retry"></i> Retry
                </button>
            </div>
        `;
    }
}

window.toggleVersionConfig = async function(orgSlug, appSlug, envSlug, version) {
    const configDiv = document.getElementById(`version-config-${version}`);
    const isVisible = configDiv.style.display !== 'none';

    // Hide all other version configs
    document.querySelectorAll('.version-config').forEach(div => {
        div.style.display = 'none';
    });

    // If it was visible, just hide it
    if (isVisible) {
        configDiv.style.display = 'none';
        return;
    }

    // Show the config div
    configDiv.style.display = 'block';

    // Check if we already have the config loaded
    const existingContent = configDiv.querySelector('.config-json');
    if (existingContent) {
        // Already loaded, just show it
        return;
    }

    // Check cache first
    const cacheKey = `${orgSlug}/${appSlug}/${envSlug}/${version}`;
    if (window.configVersionCache && window.configVersionCache[cacheKey]) {
        configDiv.innerHTML = `<pre class="config-json">${APIUtils.formatJSON(window.configVersionCache[cacheKey])}</pre>`;
        return;
    }

    // Load the configuration data from API
    try {
        configDiv.innerHTML = '<div class="loading">Loading configuration...</div>';

        // Use the new dedicated endpoint for getting specific version configuration
        const configResponse = await API.get(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}/history/${version}`);

        if (configResponse && configResponse.config) {
            // Cache the result
            if (!window.configVersionCache) {
                window.configVersionCache = {};
            }
            window.configVersionCache[cacheKey] = configResponse.config;

            // Display the configuration
            configDiv.innerHTML = `<pre class="config-json">${APIUtils.formatJSON(configResponse.config)}</pre>`;
        } else {
            configDiv.innerHTML = `
                <div class="error-state">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>No configuration data found for version ${version}</p>
                </div>
            `;
        }
    } catch (error) {
        console.error('Failed to load version config:', error);

        // Check if it's a 404 (version not found)
        if (error.response && error.response.status === 404) {
            configDiv.innerHTML = `
                <div class="error-state">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>Configuration version ${version} not found</p>
                    <p class="error-details">This version may have been deleted or never existed.</p>
                </div>
            `;
        } else {
            configDiv.innerHTML = `
                <div class="error-state">
                    <i class="fas fa-exclamation-triangle"></i>
                    <p>Failed to load configuration for version ${version}</p>
                    <button class="btn btn-sm btn-secondary" onclick="toggleVersionConfig('${orgSlug}', '${appSlug}', '${envSlug}', ${version})">
                        <i class="fas fa-retry"></i> Retry
                    </button>
                </div>
            `;
        }
    }
};

window.rollbackToVersion = function(orgSlug, appSlug, envSlug, toVersion) {
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
        <div class="modal">
            <div class="modal-header">
                <h3>Confirm Rollback</h3>
                <button class="modal-close" onclick="this.closest('.modal-overlay').remove()">×</button>
            </div>
            <div class="modal-body">
                <p>Are you sure you want to rollback to <strong>Version ${toVersion}</strong>?</p>
                <p class="warning-text">
                    <i class="fas fa-exclamation-triangle"></i>
                    This will create a new version with the configuration from Version ${toVersion} and make it active.
                </p>
                <div class="form-group">
                    <label for="rollback-reason">Reason (optional):</label>
                    <input type="text" id="rollback-reason" placeholder="e.g., Reverting due to production issue">
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" onclick="this.closest('.modal-overlay').remove()">Cancel</button>
                <button class="btn btn-warning" onclick="confirmRollback('${orgSlug}', '${appSlug}', '${envSlug}', ${toVersion})">
                    <i class="fas fa-undo"></i> Rollback
                </button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
};

window.confirmRollback = async function(orgSlug, appSlug, envSlug, toVersion) {
    const modal = document.querySelector('.modal-overlay');
    const reason = document.getElementById('rollback-reason').value;

    try {
        const rollbackBtn = modal.querySelector('.btn-warning');
        rollbackBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Rolling back...';
        rollbackBtn.disabled = true;

        await API.post(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}/rollback`, {
            to_version: toVersion,
            created_by: reason || 'Dashboard User'
        });

        modal.remove();
        dashboard.showSuccess(`Successfully rolled back to Version ${toVersion}`);

        // Refresh the configuration display
        dashboard.filterConfigurations();

    } catch (error) {
        console.error('Rollback failed:', error);
        dashboard.showError('Failed to rollback configuration: ' + APIUtils.formatError(error));

        // Reset button
        const rollbackBtn = modal.querySelector('.btn-warning');
        rollbackBtn.innerHTML = '<i class="fas fa-undo"></i> Rollback';
        rollbackBtn.disabled = false;
    }
};

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    window.dashboard = new Dashboard();
});
