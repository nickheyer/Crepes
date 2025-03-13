// API BASE URL - WOULD TYPICALLY COME FROM ENVIRONMENT VARIABLES
const API_BASE_URL = '/api';

// GENERIC FETCH WRAPPER WITH ERROR HANDLING
async function apiFetch(endpoint, options = {}) {
  try {
    const url = `${API_BASE_URL}${endpoint}`;
    const defaultOptions = {
      headers: {
        'Content-Type': 'application/json',
      },
    };
    const response = await fetch(url, { ...defaultOptions, ...options });
    
    // HANDLE NON-SUCCESS RESPONSES
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
    
    // CHECK IF RESPONSE IS EMPTY
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return await response.json();
    }
    return await response.text();
  } catch (error) {
    console.error('API REQUEST FAILED:', error);
    throw error;
  }
}

// JOB ENDPOINTS
// FETCH ALL JOBS
export async function fetchJobs() {
  return apiFetch('/jobs');
}

// FETCH JOB BY ID
export async function fetchJob(jobId) {
  return apiFetch(`/jobs/${jobId}`);
}

// CREATE A NEW JOB
export async function createJob(jobData) {
  return apiFetch('/jobs', {
    method: 'POST',
    body: JSON.stringify(jobData),
  });
}

// UPDATE AN EXISTING JOB
export async function updateJob(jobId, jobData) {
  return apiFetch(`/jobs/${jobId}`, {
    method: 'PUT',
    body: JSON.stringify(jobData),
  });
}

// DELETE A JOB
export async function deleteJob(jobId) {
  return apiFetch(`/jobs/${jobId}`, {
    method: 'DELETE',
  });
}

// START A JOB
export async function startJob(jobId) {
  return apiFetch(`/jobs/${jobId}/start`, {
    method: 'POST',
  });
}

// STOP A JOB
export async function stopJob(jobId) {
  return apiFetch(`/jobs/${jobId}/stop`, {
    method: 'POST',
  });
}

// GET JOB ASSETS
export async function fetchJobAssets(jobId) {
  return apiFetch(`/jobs/${jobId}/assets`);
}

// GET JOB STATISTICS
export async function fetchJobStatistics(jobId) {
  return apiFetch(`/jobs/${jobId}/statistics`);
}

// ASSET ENDPOINTS
// FETCH ALL ASSETS (WITH OPTIONAL FILTERS)
export async function fetchAssets(filters = {}) {
  // CONVERT FILTERS TO QUERY STRING
  const queryParams = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value) {
      queryParams.append(key, value);
    }
  }
  const queryString = queryParams.toString();
  const endpoint = queryString ? `/assets?${queryString}` : '/assets';
  const result = await apiFetch(endpoint);
  
  // HANDLE BACKEND RESPONSE FORMAT (ASSETS ARE NOW NESTED)
  if (result && result.assets) {
    return result;
  }
  return { assets: result, counts: {} };
}

// FETCH ASSET DETAILS
export async function fetchAssetDetails(assetId) {
  return apiFetch(`/assets/${assetId}`);
}

// DELETE AN ASSET
export async function deleteAsset(assetId) {
  return apiFetch(`/assets/${assetId}`, {
    method: 'DELETE',
  });
}

// REGENERATE THUMBNAIL
export async function regenerateThumbnail(assetId) {
  return apiFetch(`/assets/${assetId}/regenerate-thumbnail`, {
    method: 'POST',
  });
}

// SETTINGS ENDPOINTS
// GET ALL SETTINGS
export async function fetchSettings() {
  return apiFetch('/settings');
}

// UPDATE SETTINGS
export async function updateSettings(settingsData) {
  return apiFetch('/settings', {
    method: 'PUT',
    body: JSON.stringify(settingsData),
  });
}

// CLEAR CACHE
export async function clearCache() {
  return apiFetch('/cache/clear', {
    method: 'POST',
  });
}

// STORAGE INFO
export async function fetchStorageInfo() {
  return apiFetch('/storage/info');
}
