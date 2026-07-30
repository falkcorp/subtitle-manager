// file: webui/src/services/api.js
/**
 * Unified API service for Subtitle Manager React frontend
 * Provides a consistent interface for all backend communication
 */

/**
 * API service configuration
 */
/**
 * Determine the base path for API requests by inspecting the current URL.
 * If the first path segment isn't one of the known application routes, it is
 * treated as an installation prefix specified via `base_url` on the server.
 *
 * @returns {string} Base path beginning with a slash or an empty string.
 */
export function getBasePath() {
  const known = new Set([
    '',
    'dashboard',
    'library',
    'wanted',
    'history',
    'settings',
    'system',
    'tools',
    'details',
    'offline-info',
    'setup',
  ]);
  const parts = window.location.pathname.split('/').filter(Boolean);
  if (parts.length > 0 && !known.has(parts[0])) {
    return `/${parts[0]}`;
  }
  return '';
}

const API_BASE_URL = import.meta?.env?.VITE_API_URL || getBasePath();

/**
 * Path-prefixing fetch for callers that need raw fetch semantics.
 *
 * Components that call `fetch('/api/...')` directly send a path with no
 * installation prefix, so under a server `base_url` of `/sm` the browser
 * requests `/api/tags` rather than `/sm/api/tags`. That path has no route, so
 * the single-page-app catch-all answers with `index.html` and a 200 — the
 * component then fails trying to parse HTML as JSON, or silently renders
 * nothing. The server-side path handling cannot help: the request never
 * reaches the right route to begin with.
 *
 * This is deliberately NOT apiClient. apiClient injects
 * `Content-Type: application/json`, which would corrupt the callers that post
 * FormData (file uploads) or expect a non-JSON response. apiFetch only fixes
 * the URL and otherwise behaves exactly like fetch, so it is a safe 1:1
 * substitution.
 *
 * @param {string} url - Path beginning with `/`, or an absolute URL.
 * @param {RequestInit} [options] - Passed through to fetch untouched.
 * @returns {Promise<Response>}
 */
export function apiFetch(url, options) {
  const fullUrl = url.startsWith('http') ? url : `${API_BASE_URL}${url}`;
  // Preserve fetch's call arity. Always passing a second argument would turn
  // fetch(url) into fetch(url, undefined), which is behaviourally identical but
  // changes what a spy records — enough to break callers' tests for no reason.
  return options === undefined ? fetch(fullUrl) : fetch(fullUrl, options);
}

/**
 * Enhanced fetch wrapper with error handling and logging
 * @param {string} url - The endpoint URL
 * @param {RequestInit} options - Fetch options
 * @returns {Promise<Response>} - Fetch response
 */
async function apiClient(url, options = {}) {
  const fullUrl = url.startsWith('http') ? url : `${API_BASE_URL}${url}`;

  // Only log in development mode
  if (import.meta.env.DEV) {
    console.log(`API Request: ${options.method || 'GET'} ${fullUrl}`);
  }

  const config = {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    ...options,
  };

  try {
    const response = await fetch(fullUrl, config);
    if (import.meta.env.DEV) {
      console.log(`API Response: ${response.status} ${fullUrl}`);
    }
    return response;
  } catch (error) {
    // Only log errors in development mode to avoid console errors in production/testing
    if (import.meta.env.DEV) {
      console.error('API Request Error:', error);

      if (error.name === 'TypeError' && error.message.includes('fetch')) {
        console.warn('Backend service appears to be offline');
      }
    }

    throw error;
  }
}

/**
 * API service object with common HTTP methods and semantic endpoints
 */
export const apiService = {
  /**
   * Perform GET request
   * @param {string} url - The endpoint URL
   * @param {RequestInit} config - Optional fetch config
   * @returns {Promise<Response>} - Promise with response
   */
  async get(url, config = {}) {
    return apiClient(url, { method: 'GET', ...config });
  },

  /**
   * Perform POST request
   * @param {string} url - The endpoint URL
   * @param {any} data - Request payload
   * @param {RequestInit} config - Optional fetch config
   * @returns {Promise<Response>} - Promise with response
   */
  async post(url, data = {}, config = {}) {
    return apiClient(url, {
      method: 'POST',
      body: JSON.stringify(data),
      ...config,
    });
  },

  /**
   * Perform POST request with FormData (for file uploads)
   * @param {string} url - The endpoint URL
   * @param {FormData} formData - FormData object containing files/fields
   * @param {RequestInit} config - Optional fetch config
   * @returns {Promise<Response>} - Promise with response
   */
  async postFormData(url, formData, config = {}) {
    const { headers, ...otherConfig } = config;
    // Remove Content-Type header to let browser set it with boundary for FormData
    const cleanHeaders = { ...headers };
    delete cleanHeaders['Content-Type'];

    return apiClient(url, {
      method: 'POST',
      body: formData,
      headers: cleanHeaders,
      ...otherConfig,
    });
  },

  /**
   * Perform PUT request
   * @param {string} url - The endpoint URL
   * @param {any} data - Request payload
   * @param {RequestInit} config - Optional fetch config
   * @returns {Promise<Response>} - Promise with response
   */
  async put(url, data = {}, config = {}) {
    return apiClient(url, {
      method: 'PUT',
      body: JSON.stringify(data),
      ...config,
    });
  },

  /**
   * Perform DELETE request
   * @param {string} url - The endpoint URL
   * @param {RequestInit} config - Optional fetch config
   * @returns {Promise<Response>} - Promise with response
   */
  async delete(url, config = {}) {
    return apiClient(url, { method: 'DELETE', ...config });
  },

  /**
   * Check if the backend is available
   * @returns {Promise<boolean>} - Whether backend is available
   */
  async checkBackendHealth() {
    try {
      // Use setup/status endpoint for health check as it's unauthenticated
      const response = await this.get('/api/setup/status');
      return response.ok;
    } catch (error) {
      if (import.meta.env.DEV) {
        console.warn('Backend health check failed:', error);
      }
      return false;
    }
  },

  // ==================================================
  // SEMANTIC API METHODS - Higher-level functionality
  // ==================================================

  /**
   * Authentication and Session Management
   */
  auth: {
    /**
     * Login with username and password
     * @param {string} username - Username
     * @param {string} password - Password
     * @returns {Promise<Response>} - Login response
     */
    async login(username, password) {
      const formData = new FormData();
      formData.append('username', username);
      formData.append('password', password);
      return apiService.postFormData('/api/login', formData);
    },

    /**
     * Logout current session
     * @returns {Promise<Response>} - Logout response
     */
    async logout() {
      return apiService.post('/api/logout');
    },
  },

  /**
   * OAuth Management
   */
  oauth: {
    /**
     * Generate GitHub OAuth configuration
     * @returns {Promise<Response>} - Generate OAuth response
     */
    async generateGitHub() {
      return apiService.post('/api/oauth/github/generate');
    },

    /**
     * Regenerate GitHub OAuth configuration
     * @returns {Promise<Response>} - Regenerate OAuth response
     */
    async regenerateGitHub() {
      return apiService.post('/api/oauth/github/regenerate');
    },

    /**
     * Reset GitHub OAuth configuration
     * @returns {Promise<Response>} - Reset OAuth response
     */
    async resetGitHub() {
      return apiService.post('/api/oauth/github/reset');
    },
  },

  /**
   * Configuration Management
   */
  config: {
    /**
     * Get current configuration
     * @returns {Promise<Response>} - Configuration response
     */
    async get() {
      return apiService.get('/api/config');
    },

    /**
     * Update configuration
     * @param {Object} configData - Configuration updates
     * @returns {Promise<Response>} - Update response
     */
    async update(configData) {
      return apiService.post('/api/config', configData);
    },
  },

  /**
   * Setup Management
   */
  setup: {
    /**
     * Check if setup is needed
     * @returns {Promise<Response>} - Setup status response
     */
    async getStatus() {
      return apiService.get('/api/setup/status');
    },

    /**
     * Perform initial setup
     * @param {Object} setupData - Setup configuration
     * @returns {Promise<Response>} - Setup response
     */
    async initialize(setupData) {
      return apiService.post('/api/setup', setupData);
    },

    /**
     * Import Bazarr configuration
     * @param {string} url - Bazarr URL
     * @param {string} apiKey - Bazarr API key
     * @returns {Promise<Response>} - Import response
     */
    async importBazarr(url, apiKey) {
      return apiService.post('/api/setup/bazarr', { url, api_key: apiKey });
    },

    /**
     * Upload Bazarr config file
     * @param {File} configFile - Bazarr config.ini file
     * @returns {Promise<Response>} - Upload response
     */
    async uploadBazarrConfig(configFile) {
      const formData = new FormData();
      formData.append('config', configFile);
      return apiService.postFormData('/api/setup/bazarr/upload', formData);
    },
  },

  /**
   * Provider Management
   */
  providers: {
    /**
     * Get list of all providers
     * @returns {Promise<Response>} - Providers list response
     */
    async list() {
      return apiService.get('/api/providers');
    },

    /**
     * Update provider configuration
     * @param {string} name - Provider name
     * @param {boolean} enabled - Whether provider is enabled
     * @param {Object} config - Provider configuration
     * @returns {Promise<Response>} - Update response
     */
    async update(name, enabled, config) {
      return apiService.post('/api/providers', { name, enabled, config });
    },

    /**
     * Get provider status
     * @returns {Promise<Response>} - Provider status response
     */
    async getStatus() {
      return apiService.get('/api/providers/status');
    },

    /**
     * Refresh provider status
     * @returns {Promise<Response>} - Refresh response
     */
    async refresh() {
      return apiService.post('/api/providers/refresh');
    },

    /**
     * Reset provider configuration
     * @returns {Promise<Response>} - Reset response
     */
    async reset() {
      return apiService.post('/api/providers/reset');
    },
  },

  /**
   * Subtitle Operations
   */
  subtitles: {
    /**
     * Convert subtitle file to SRT format
     * @param {File} file - Subtitle file to convert
     * @returns {Promise<Response>} - Converted file response (blob)
     */
    async convert(file) {
      const formData = new FormData();
      formData.append('file', file);
      return apiService.postFormData('/api/convert', formData);
    },

    /**
     * Extract subtitles from media file
     * @param {string} filePath - Path to media file
     * @returns {Promise<Response>} - Extracted subtitles response
     */
    async extract(filePath) {
      return apiService.post('/api/extract', { path: filePath });
    },

    /**
     * Translate subtitle file
     * @param {File} file - Subtitle file to translate
     * @param {string} targetLanguage - Target language code
     * @param {Object} options - Translation options
     * @returns {Promise<Response>} - Translated file response
     */
    async translate(file, targetLanguage, options = {}) {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('lang', targetLanguage); // Backend expects 'lang' parameter
      // Add other options as form fields
      Object.entries(options).forEach(([key, value]) => {
        formData.append(key, String(value));
      });
      return apiService.postFormData('/api/translate', formData);
    },

    /**
     * Download subtitle file
     * @param {string} downloadId - Download identifier
     * @returns {Promise<Response>} - Download response
     */
    async download(downloadId) {
      return apiService.get(
        `/api/download?id=${encodeURIComponent(downloadId)}`
      );
    },
  },

  /**
   * Library Management
   */
  library: {
    /**
     * Browse library directory
     * @param {string} path - Directory path to browse
     * @returns {Promise<Response>} - Directory contents response
     */
    async browse(path = '') {
      const query = path ? `?path=${encodeURIComponent(path)}` : '';
      return apiService.get(`/api/library/browse${query}`);
    },

    /**
     * Start library scan
     * @param {Object} scanOptions - Scan configuration
     * @returns {Promise<Response>} - Scan start response
     */
    async startScan(scanOptions = {}) {
      return apiService.post('/api/library/scan', scanOptions);
    },

    /**
     * Get library scan status
     * @returns {Promise<Response>} - Scan status response
     */
    async getScanStatus() {
      return apiService.get('/api/library/scan/status');
    },

    /**
     * Get library tags
     * @returns {Promise<Response>} - Tags response
     */
    async getTags() {
      return apiService.get('/api/library/tags');
    },
  },

  /**
   * History Management
   */
  history: {
    /**
     * Get operation history
     * @param {Object} filters - Optional filters (page, limit, etc.)
     * @returns {Promise<Response>} - History response
     */
    async get(filters = {}) {
      const query = new URLSearchParams(filters).toString();
      const queryString = query ? `?${query}` : '';
      return apiService.get(`/api/history${queryString}`);
    },
  },

  /**
   * System Management
   */
  system: {
    /**
     * Get system information
     * @returns {Promise<Response>} - System info response
     */
    async getInfo() {
      return apiService.get('/api/system');
    },

    /**
     * Get system logs
     * @returns {Promise<Response>} - Logs response
     */
    async getLogs() {
      return apiService.get('/api/logs');
    },

    /**
     * Get available releases
     * @returns {Promise<Response>} - Releases response
     */
    async getReleases() {
      return apiService.get('/api/releases');
    },

    /**
     * Get announcements
     * @returns {Promise<Response>} - Announcements response
     */
    async getAnnouncements() {
      return apiService.get('/api/announcements');
    },
  },

  /**
   * Database Management
   */
  database: {
    /**
     * Get database information
     * @returns {Promise<Response>} - Database info response
     */
    async getInfo() {
      return apiService.get('/api/database/info');
    },

    /**
     * Get database statistics
     * @returns {Promise<Response>} - Database stats response
     */
    async getStats() {
      return apiService.get('/api/database/stats');
    },

    /**
     * Create database backup
     * @returns {Promise<Response>} - Backup response
     */
    async backup() {
      return apiService.post('/api/database/backup');
    },

    /**
     * Optimize database
     * @returns {Promise<Response>} - Optimize response
     */
    async optimize() {
      return apiService.post('/api/database/optimize');
    },
  },

  /**
   * Backup Management
   */
  backups: {
    /**
     * List available backups
     * @returns {Promise<Response>} - Backups list response
     */
    async list() {
      return apiService.get('/api/backups');
    },

    /**
     * Create new backup
     * @returns {Promise<Response>} - Create backup response
     */
    async create() {
      return apiService.post('/api/backups/create');
    },

    /**
     * Restore from backup
     * @param {string} backupId - Backup identifier
     * @returns {Promise<Response>} - Restore response
     */
    async restore(backupId) {
      return apiService.post('/api/backups/restore', { backup_id: backupId });
    },
  },

  /**
   * Task Management
   */
  tasks: {
    /**
     * Get task list
     * @returns {Promise<Response>} - Tasks response
     */
    async list() {
      return apiService.get('/api/tasks');
    },

    /**
     * Start a task
     * @param {string} taskName - Name of task to start
     * @param {Object} params - Task parameters
     * @returns {Promise<Response>} - Start task response
     */
    async start(taskName, params = {}) {
      return apiService.post('/api/tasks/start', { task: taskName, ...params });
    },
  },

  /**
   * User Management
   */
  users: {
    /**
     * Get user list
     * @returns {Promise<Response>} - Users response
     */
    async list() {
      return apiService.get('/api/users/');
    },

    /**
     * Create new user
     * @param {Object} userData - User data
     * @returns {Promise<Response>} - Create user response
     */
    async create(userData) {
      return apiService.post('/api/users/', userData);
    },

    /**
     * Update user
     * @param {string} userId - User ID
     * @param {Object} userData - Updated user data
     * @returns {Promise<Response>} - Update user response
     */
    async update(userId, userData) {
      return apiService.put(`/api/users/${userId}`, userData);
    },

    /**
     * Delete user
     * @param {string} userId - User ID
     * @returns {Promise<Response>} - Delete user response
     */
    async delete(userId) {
      return apiService.delete(`/api/users/${userId}`);
    },

    /**
     * Reset user password
     * @param {string} userId - User ID
     * @returns {Promise<Response>} - Reset password response
     */
    async resetPassword(userId) {
      return apiService.post(`/api/users/${userId}/reset`);
    },
  },

  /**
   * Tag Management
   */
  tags: {
    /**
     * Get all tags
     * @returns {Promise<Response>} - Tags response
     */
    async list() {
      return apiService.get('/api/tags');
    },

    /**
     * Create new tag
     * @param {Object} tagData - Tag data
     * @returns {Promise<Response>} - Create tag response
     */
    async create(tagData) {
      return apiService.post('/api/tags', tagData);
    },

    /**
     * Update tag
     * @param {string} tagId - Tag ID
     * @param {Object} tagData - Updated tag data
     * @returns {Promise<Response>} - Update tag response
     */
    async update(tagId, tagData) {
      return apiService.put(`/api/tags/${tagId}`, tagData);
    },

    /**
     * Delete tag
     * @param {string} tagId - Tag ID
     * @returns {Promise<Response>} - Delete tag response
     */
    async delete(tagId) {
      return apiService.delete(`/api/tags/${tagId}`);
    },

    /**
     * Bulk update tags
     * @param {Object} bulkData - Bulk operation data
     * @returns {Promise<Response>} - Bulk update response
     */
    async bulkUpdate(bulkData) {
      return apiService.post('/api/tags/bulk', bulkData);
    },
  },

  /**
   * Widget Management
   */
  widgets: {
    /**
     * Get available widgets
     * @returns {Promise<Response>} - Widgets response
     */
    async list() {
      return apiService.get('/api/widgets');
    },

    /**
     * Get dashboard layout
     * @returns {Promise<Response>} - Layout response
     */
    async getLayout() {
      return apiService.get('/api/widgets/layout');
    },

    /**
     * Update dashboard layout
     * @param {Object} layoutData - Layout configuration
     * @returns {Promise<Response>} - Update layout response
     */
    async updateLayout(layoutData) {
      return apiService.post('/api/widgets/layout', layoutData);
    },
  },

  // ==================================================
  // UTILITY METHODS
  // ==================================================

  /**
   * Helper method to handle file downloads with proper naming
   * @param {string} url - Download URL
   * @param {string} filename - Suggested filename
   * @returns {Promise<void>}
   */
  async downloadFile(url, filename) {
    try {
      const response = await this.get(url);
      if (!response.ok) {
        throw new Error(`Download failed: ${response.status}`);
      }

      const blob = await response.blob();
      const downloadUrl = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = downloadUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(downloadUrl);
    } catch (error) {
      console.error('Download failed:', error);
      throw error;
    }
  },

  /**
   * Helper method to parse JSON response with error handling
   * @param {Response} response - Fetch response
   * @returns {Promise<any>} - Parsed JSON data
   */
  async parseJsonResponse(response) {
    if (!response.ok) {
      let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
      try {
        const errorData = await response.json();
        if (errorData.message) {
          errorMessage = errorData.message;
        } else if (errorData.error) {
          errorMessage = errorData.error;
        }
      } catch {
        // Fallback to status text if JSON parsing fails
      }
      throw new Error(errorMessage);
    }

    try {
      return await response.json();
    } catch (parseError) {
      console.error('JSON parsing error:', parseError);
      throw new Error('Invalid JSON response from server');
    }
  },

  /**
   * Helper method to handle form validation errors
   * @param {Response} response - Fetch response
   * @returns {Promise<Object>} - Validation errors object
   */
  async getValidationErrors(response) {
    if (response.status === 400) {
      try {
        const errorData = await response.json();
        return errorData.errors || {};
      } catch {
        return {};
      }
    }
    return {};
  },
};

export default apiService;
