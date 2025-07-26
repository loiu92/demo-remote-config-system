// API Utility Class
class API {
    static baseURL = '';
    static defaultHeaders = {
        'Content-Type': 'application/json',
    };

    static async request(method, url, data = null, headers = {}) {
        const config = {
            method: method.toUpperCase(),
            headers: { ...this.defaultHeaders, ...headers },
        };

        if (data && (method.toUpperCase() === 'POST' || method.toUpperCase() === 'PUT')) {
            config.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(this.baseURL + url, config);
            
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new APIError(response.status, errorData.message || 'Request failed', errorData);
            }

            // Handle empty responses (like 204 No Content)
            if (response.status === 204 || response.headers.get('content-length') === '0') {
                return null;
            }

            return await response.json();
        } catch (error) {
            if (error instanceof APIError) {
                throw error;
            }
            throw new APIError(0, 'Network error', { originalError: error.message });
        }
    }

    static async get(url, headers = {}) {
        return this.request('GET', url, null, headers);
    }

    static async post(url, data, headers = {}) {
        return this.request('POST', url, data, headers);
    }

    static async put(url, data, headers = {}) {
        return this.request('PUT', url, data, headers);
    }

    static async delete(url, headers = {}) {
        return this.request('DELETE', url, null, headers);
    }

    // Organization API methods
    static async getOrganizations(params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const url = `/admin/orgs${queryString ? '?' + queryString : ''}`;
        return this.get(url);
    }

    static async getOrganization(slug) {
        return this.get(`/admin/orgs/${slug}`);
    }

    static async createOrganization(data) {
        return this.post('/admin/orgs', data);
    }

    static async updateOrganization(slug, data) {
        return this.put(`/admin/orgs/${slug}`, data);
    }

    static async deleteOrganization(slug) {
        return this.delete(`/admin/orgs/${slug}`);
    }

    // Application API methods
    static async getApplications(orgSlug, params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const url = `/admin/orgs/${orgSlug}/apps${queryString ? '?' + queryString : ''}`;
        return this.get(url);
    }

    static async getApplication(orgSlug, appSlug) {
        return this.get(`/admin/orgs/${orgSlug}/apps/${appSlug}`);
    }

    static async createApplication(orgSlug, data) {
        return this.post(`/admin/orgs/${orgSlug}/apps`, data);
    }

    static async updateApplication(orgSlug, appSlug, data) {
        return this.put(`/admin/orgs/${orgSlug}/apps/${appSlug}`, data);
    }

    static async deleteApplication(orgSlug, appSlug) {
        return this.delete(`/admin/orgs/${orgSlug}/apps/${appSlug}`);
    }

    // Environment API methods
    static async getEnvironments(orgSlug, appSlug, params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const url = `/admin/orgs/${orgSlug}/apps/${appSlug}/envs${queryString ? '?' + queryString : ''}`;
        return this.get(url);
    }

    static async getEnvironment(orgSlug, appSlug, envSlug) {
        return this.get(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}`);
    }

    static async createEnvironment(orgSlug, appSlug, data) {
        return this.post(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs`, data);
    }

    static async updateEnvironment(orgSlug, appSlug, envSlug, data) {
        return this.put(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}`, data);
    }

    static async deleteEnvironment(orgSlug, appSlug, envSlug) {
        return this.delete(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}`);
    }

    // Configuration API methods
    static async getConfiguration(orgSlug, appSlug, envSlug) {
        return this.get(`/config/${orgSlug}/${appSlug}/${envSlug}`);
    }

    static async updateConfiguration(orgSlug, appSlug, envSlug, data) {
        return this.put(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}/config`, data);
    }

    static async getConfigurationHistory(orgSlug, appSlug, envSlug, params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const url = `/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}/history${queryString ? '?' + queryString : ''}`;
        return this.get(url);
    }

    static async getConfigurationChanges(orgSlug, appSlug, envSlug, params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const url = `/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}/changes${queryString ? '?' + queryString : ''}`;
        return this.get(url);
    }

    static async rollbackConfiguration(orgSlug, appSlug, envSlug, data) {
        return this.post(`/admin/orgs/${orgSlug}/apps/${appSlug}/envs/${envSlug}/rollback`, data);
    }

    // Cache API methods
    static async getCacheStats() {
        return this.get('/admin/cache/stats');
    }

    static async warmCache() {
        return this.post('/admin/cache/warm');
    }

    static async clearCache() {
        return this.delete('/admin/cache');
    }

    // SSE API methods
    static async getSSEStats() {
        return this.get('/admin/sse/stats');
    }

    // Health check
    static async healthCheck() {
        return this.get('/health');
    }
}

// Custom Error Class
class APIError extends Error {
    constructor(status, message, data = {}) {
        super(message);
        this.name = 'APIError';
        this.status = status;
        this.data = data;
    }

    toString() {
        return `APIError ${this.status}: ${this.message}`;
    }
}

// Export for use in other modules
window.API = API;
window.APIError = APIError;

// Add request interceptor for loading states
const originalRequest = API.request;
API.request = async function(method, url, data, headers) {
    // Show loading indicator if available
    const loadingIndicator = document.querySelector('.loading-indicator');
    if (loadingIndicator) {
        loadingIndicator.style.display = 'block';
    }

    try {
        const result = await originalRequest.call(this, method, url, data, headers);
        return result;
    } catch (error) {
        // Log error for debugging
        console.error(`API Error: ${method} ${url}`, error);
        throw error;
    } finally {
        // Hide loading indicator
        if (loadingIndicator) {
            loadingIndicator.style.display = 'none';
        }
    }
};

// Utility functions for common operations
window.APIUtils = {
    // Format error message for display
    formatError(error) {
        if (error instanceof APIError) {
            return error.message || 'An error occurred';
        }
        return error.message || 'Unknown error';
    },

    // Handle common API responses
    handleResponse(response, successMessage = 'Operation completed successfully') {
        if (response === null) {
            // Handle 204 No Content responses
            return { success: true, message: successMessage };
        }
        return response;
    },

    // Validate required fields
    validateRequired(data, fields) {
        const missing = fields.filter(field => !data[field] || data[field].trim() === '');
        if (missing.length > 0) {
            throw new Error(`Missing required fields: ${missing.join(', ')}`);
        }
    },

    // Format date for display
    formatDate(dateString) {
        if (!dateString) return 'N/A';
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
    },

    // Format JSON for display
    formatJSON(obj, indent = 2) {
        try {
            return JSON.stringify(obj, null, indent);
        } catch (error) {
            return 'Invalid JSON';
        }
    },

    // Parse JSON safely
    parseJSON(str) {
        try {
            return JSON.parse(str);
        } catch (error) {
            throw new Error('Invalid JSON format');
        }
    },

    // Debounce function for search inputs
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    // Generate slug from name
    generateSlug(name) {
        return name
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-+|-+$/g, '');
    }
};
