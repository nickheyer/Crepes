import { addToast } from '$lib/stores/uiStore.svelte';

const API_BASE_URL = '/api';

export async function apiRequest(endpoint, options = {}, showToasts = true) {
  const url = `${API_BASE_URL}${endpoint}`;
  const defaultOptions = {
    headers: {
      'Content-Type': 'application/json',
    },
  };
  
  try {
    const response = await fetch(url, { ...defaultOptions, ...options });
    
    if (!response.ok) {
      let errorMessage;
      try {
        const errorData = await response.json();
        errorMessage = errorData.error || `HTTP error ${response.status}`;
      } catch (e) {
        errorMessage = `HTTP error ${response.status}`;
      }
      throw new Error(errorMessage);
    }
    
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      const data = await response.json();
      return data;
    }
    
    return await response.text();
  } catch (error) {
    console.error('API request failed:', error.message);
    if (showToasts) {
      addToast(error.message, 'error');
    }
    throw error;
  }
}

// JOBS API
export const jobsApi = {
  getAll: () => apiRequest('/jobs'),
  getById: (id) => apiRequest(`/jobs/${id}`),
  create: (jobData) => apiRequest('/jobs', {
    method: 'POST',
    body: JSON.stringify(jobData),
  }),
  update: (id, jobData) => apiRequest(`/jobs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(jobData),
  }),
  delete: (id) => apiRequest(`/jobs/${id}`, {
    method: 'DELETE',
  }),
  start: (id) => apiRequest(`/jobs/${id}/start`, {
    method: 'POST',
  }),
  stop: (id) => apiRequest(`/jobs/${id}/stop`, {
    method: 'POST',
  }),
  getStatistics: (id) => apiRequest(`/jobs/${id}/statistics`),
  getAssets: (id) => apiRequest(`/jobs/${id}/assets`),
};

// ASSETS API
export const assetsApi = {
  getAll: (filters = {}) => {
    const queryParams = new URLSearchParams();
    for (const [key, value] of Object.entries(filters)) {
      if (value) {
        queryParams.append(key, value);
      }
    }
    const queryString = queryParams.toString();
    const endpoint = queryString ? `/assets?${queryString}` : '/assets';
    
    return apiRequest(endpoint).then((result) => {
      if (result && result.assets) {
        return result;
      }
      return { assets: result, counts: {} };
    });
  },
  getById: (id) => apiRequest(`/assets/${id}`),
  delete: (id) => apiRequest(`/assets/${id}`, {
    method: 'DELETE',
  }),
  regenerateThumbnail: (id) => apiRequest(`/assets/${id}/regenerate-thumbnail`, {
    method: 'POST',
  }),
};

// SETTINGS API
export const settingsApi = {
  getAll: () => apiRequest('/settings'),
  update: (settingsData) => apiRequest('/settings', {
    method: 'PUT',
    body: JSON.stringify(settingsData),
  }),
  clearCache: () => apiRequest('/cache/clear', {
    method: 'POST',
  }),
  getStorageInfo: () => apiRequest('/storage/info'),
};
