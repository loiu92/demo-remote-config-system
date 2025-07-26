// Modal Management System
class ModalManager {
    constructor() {
        this.currentModal = null;
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Close modal when clicking outside
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal-overlay')) {
                this.closeModal();
            }
        });

        // Close modal with Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.currentModal) {
                this.closeModal();
            }
        });
    }

    showModal(content) {
        const modalContainer = document.getElementById('modal-container');
        modalContainer.innerHTML = `
            <div class="modal-overlay">
                <div class="modal">
                    ${content}
                </div>
            </div>
        `;
        this.currentModal = modalContainer.querySelector('.modal');
        document.body.style.overflow = 'hidden';
    }

    closeModal() {
        const modalContainer = document.getElementById('modal-container');
        modalContainer.innerHTML = '';
        this.currentModal = null;
        document.body.style.overflow = '';
    }

    showCreateOrgModal() {
        const content = `
            <div class="modal-header">
                <h3>Create Organization</h3>
                <button class="modal-close" onclick="modalManager.closeModal()">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <form class="modal-form" onsubmit="handleCreateOrg(event)">
                <div class="form-group">
                    <label for="org-name">Organization Name *</label>
                    <input type="text" id="org-name" name="name" required 
                           placeholder="Enter organization name">
                </div>
                <div class="form-group">
                    <label for="org-slug">Slug *</label>
                    <input type="text" id="org-slug" name="slug" required 
                           placeholder="organization-slug" pattern="[a-z0-9-]+">
                    <small>Lowercase letters, numbers, and hyphens only</small>
                </div>
                <div class="modal-actions">
                    <button type="button" class="btn btn-secondary" onclick="modalManager.closeModal()">
                        Cancel
                    </button>
                    <button type="submit" class="btn btn-primary">
                        <i class="fas fa-plus"></i> Create Organization
                    </button>
                </div>
            </form>
        `;
        this.showModal(content);

        // Auto-generate slug from name
        const nameInput = document.getElementById('org-name');
        const slugInput = document.getElementById('org-slug');
        nameInput.addEventListener('input', () => {
            slugInput.value = APIUtils.generateSlug(nameInput.value);
        });
    }

    showCreateAppModal() {
        if (dashboard.data.organizations.length === 0) {
            alert('Please create an organization first');
            return;
        }

        const orgOptions = dashboard.data.organizations.map(org => 
            `<option value="${org.slug}">${org.name}</option>`
        ).join('');

        const content = `
            <div class="modal-header">
                <h3>Create Application</h3>
                <button class="modal-close" onclick="modalManager.closeModal()">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <form class="modal-form" onsubmit="handleCreateApp(event)">
                <div class="form-group">
                    <label for="app-org">Organization *</label>
                    <select id="app-org" name="organization" required>
                        <option value="">Select Organization</option>
                        ${orgOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label for="app-name">Application Name *</label>
                    <input type="text" id="app-name" name="name" required 
                           placeholder="Enter application name">
                </div>
                <div class="form-group">
                    <label for="app-slug">Slug *</label>
                    <input type="text" id="app-slug" name="slug" required 
                           placeholder="application-slug" pattern="[a-z0-9-]+">
                    <small>Lowercase letters, numbers, and hyphens only</small>
                </div>
                <div class="form-group">
                    <label for="app-api-key">API Key (optional)</label>
                    <input type="text" id="app-api-key" name="api_key" 
                           placeholder="Leave empty to auto-generate">
                    <small>Leave empty to automatically generate a secure API key</small>
                </div>
                <div class="modal-actions">
                    <button type="button" class="btn btn-secondary" onclick="modalManager.closeModal()">
                        Cancel
                    </button>
                    <button type="submit" class="btn btn-primary">
                        <i class="fas fa-plus"></i> Create Application
                    </button>
                </div>
            </form>
        `;
        this.showModal(content);

        // Auto-generate slug from name
        const nameInput = document.getElementById('app-name');
        const slugInput = document.getElementById('app-slug');
        nameInput.addEventListener('input', () => {
            slugInput.value = APIUtils.generateSlug(nameInput.value);
        });
    }

    showCreateEnvModal() {
        if (dashboard.data.applications.length === 0) {
            alert('Please create an application first');
            return;
        }

        const appOptions = dashboard.data.applications.map(app => 
            `<option value="${app.organization.slug}/${app.slug}">${app.name} (${app.organization.name})</option>`
        ).join('');

        const content = `
            <div class="modal-header">
                <h3>Create Environment</h3>
                <button class="modal-close" onclick="modalManager.closeModal()">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <form class="modal-form" onsubmit="handleCreateEnv(event)">
                <div class="form-group">
                    <label for="env-app">Application *</label>
                    <select id="env-app" name="application" required>
                        <option value="">Select Application</option>
                        ${appOptions}
                    </select>
                </div>
                <div class="form-group">
                    <label for="env-name">Environment Name *</label>
                    <input type="text" id="env-name" name="name" required 
                           placeholder="Enter environment name">
                </div>
                <div class="form-group">
                    <label for="env-slug">Slug *</label>
                    <input type="text" id="env-slug" name="slug" required 
                           placeholder="environment-slug" pattern="[a-z0-9-]+">
                    <small>Lowercase letters, numbers, and hyphens only</small>
                </div>
                <div class="modal-actions">
                    <button type="button" class="btn btn-secondary" onclick="modalManager.closeModal()">
                        Cancel
                    </button>
                    <button type="submit" class="btn btn-primary">
                        <i class="fas fa-plus"></i> Create Environment
                    </button>
                </div>
            </form>
        `;
        this.showModal(content);

        // Auto-generate slug from name
        const nameInput = document.getElementById('env-name');
        const slugInput = document.getElementById('env-slug');
        nameInput.addEventListener('input', () => {
            slugInput.value = APIUtils.generateSlug(nameInput.value);
        });
    }

    showUpdateConfigModal() {
        const envFilter = document.getElementById('env-filter');
        const selectedEnv = envFilter.value;
        
        if (!selectedEnv) {
            alert('Please select an environment first');
            return;
        }

        const [orgSlug, appSlug, envSlug] = selectedEnv.split('/');
        
        const content = `
            <div class="modal-header">
                <h3>Update Configuration</h3>
                <button class="modal-close" onclick="modalManager.closeModal()">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <form class="modal-form" onsubmit="handleUpdateConfig(event, '${orgSlug}', '${appSlug}', '${envSlug}')">
                <div class="form-group">
                    <label for="config-json">Configuration JSON *</label>
                    <textarea id="config-json" name="config" required rows="15" 
                              placeholder='{"key": "value", "nested": {"key": "value"}}'></textarea>
                    <small>Enter valid JSON configuration</small>
                </div>
                <div class="form-group">
                    <label for="config-created-by">Created By</label>
                    <input type="text" id="config-created-by" name="created_by" 
                           placeholder="Enter your name or email">
                </div>
                <div class="modal-actions">
                    <button type="button" class="btn btn-secondary" onclick="modalManager.closeModal()">
                        Cancel
                    </button>
                    <button type="submit" class="btn btn-primary">
                        <i class="fas fa-save"></i> Update Configuration
                    </button>
                </div>
            </form>
        `;
        this.showModal(content);

        // Load current configuration
        this.loadCurrentConfig(orgSlug, appSlug, envSlug);
    }

    async loadCurrentConfig(orgSlug, appSlug, envSlug) {
        try {
            const config = await API.getConfiguration(orgSlug, appSlug, envSlug);
            const textarea = document.getElementById('config-json');
            if (config && config.config) {
                textarea.value = APIUtils.formatJSON(config.config);
            }
        } catch (error) {
            console.error('Failed to load current config:', error);
            // Don't show error, just leave textarea empty for new config
        }
    }
}

// Initialize modal manager
const modalManager = new ModalManager();

// Global modal functions
window.showCreateOrgModal = () => modalManager.showCreateOrgModal();
window.showCreateAppModal = () => modalManager.showCreateAppModal();
window.showCreateEnvModal = () => modalManager.showCreateEnvModal();
window.showUpdateConfigModal = () => modalManager.showUpdateConfigModal();

// Form handlers
window.handleCreateOrg = async (event) => {
    event.preventDefault();
    const formData = new FormData(event.target);
    const data = Object.fromEntries(formData);

    try {
        APIUtils.validateRequired(data, ['name', 'slug']);
        await API.createOrganization(data);
        modalManager.closeModal();
        dashboard.showSuccess('Organization created successfully');
        await dashboard.loadOrganizations();
        if (dashboard.currentSection === 'organizations') {
            dashboard.renderOrganizations();
        }
    } catch (error) {
        alert('Error: ' + APIUtils.formatError(error));
    }
};

window.handleCreateApp = async (event) => {
    event.preventDefault();
    const formData = new FormData(event.target);
    const data = Object.fromEntries(formData);
    const orgSlug = data.organization;
    delete data.organization;

    try {
        APIUtils.validateRequired(data, ['name', 'slug']);
        await API.createApplication(orgSlug, data);
        modalManager.closeModal();
        dashboard.showSuccess('Application created successfully');
        await dashboard.loadApplications();
        if (dashboard.currentSection === 'applications') {
            dashboard.renderApplications();
        }
    } catch (error) {
        alert('Error: ' + APIUtils.formatError(error));
    }
};

window.handleCreateEnv = async (event) => {
    event.preventDefault();
    const formData = new FormData(event.target);
    const data = Object.fromEntries(formData);
    const [orgSlug, appSlug] = data.application.split('/');
    delete data.application;

    try {
        APIUtils.validateRequired(data, ['name', 'slug']);
        await API.createEnvironment(orgSlug, appSlug, data);
        modalManager.closeModal();
        dashboard.showSuccess('Environment created successfully');
        await dashboard.loadEnvironments();
        if (dashboard.currentSection === 'environments') {
            dashboard.renderEnvironments();
        }
    } catch (error) {
        alert('Error: ' + APIUtils.formatError(error));
    }
};

window.handleUpdateConfig = async (event, orgSlug, appSlug, envSlug) => {
    event.preventDefault();
    const formData = new FormData(event.target);
    const data = Object.fromEntries(formData);

    try {
        // Validate JSON
        const config = APIUtils.parseJSON(data.config);
        const payload = {
            config: config,
            created_by: data.created_by || 'Dashboard User'
        };

        await API.updateConfiguration(orgSlug, appSlug, envSlug, payload);
        modalManager.closeModal();
        dashboard.showSuccess('Configuration updated successfully');
        
        // Refresh configuration display if on configurations section
        if (dashboard.currentSection === 'configurations') {
            dashboard.filterConfigurations();
        }
    } catch (error) {
        alert('Error: ' + APIUtils.formatError(error));
    }
};

// Add modal styles
const modalStyles = `
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 2000;
    }

    .modal {
        background: white;
        border-radius: 8px;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
        max-width: 500px;
        width: 90%;
        max-height: 90vh;
        overflow-y: auto;
    }

    .modal-header {
        padding: 20px;
        border-bottom: 1px solid #e9ecef;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }

    .modal-header h3 {
        margin: 0;
        color: #2c3e50;
    }

    .modal-close {
        background: none;
        border: none;
        font-size: 18px;
        cursor: pointer;
        color: #6c757d;
        padding: 5px;
    }

    .modal-close:hover {
        color: #dc3545;
    }

    .modal-form {
        padding: 20px;
    }

    .form-group {
        margin-bottom: 20px;
    }

    .form-group label {
        display: block;
        margin-bottom: 5px;
        font-weight: 500;
        color: #2c3e50;
    }

    .form-group input,
    .form-group select,
    .form-group textarea {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid #ced4da;
        border-radius: 4px;
        font-size: 14px;
        font-family: inherit;
    }

    .form-group textarea {
        resize: vertical;
        font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    }

    .form-group small {
        display: block;
        margin-top: 5px;
        color: #6c757d;
        font-size: 12px;
    }

    .modal-actions {
        display: flex;
        justify-content: flex-end;
        gap: 10px;
        margin-top: 30px;
        padding-top: 20px;
        border-top: 1px solid #e9ecef;
    }
`;

const styleSheet = document.createElement('style');
styleSheet.textContent = modalStyles;
document.head.appendChild(styleSheet);
