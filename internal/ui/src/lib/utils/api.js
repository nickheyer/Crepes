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
    console.error('API request failed:', error);
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
  
  return apiFetch(endpoint);
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

// TEMPLATE ENDPOINTS

// THESE ENDPOINTS WOULD BE IMPLEMENTED ON THE BACKEND

// FETCH ALL TEMPLATES
export async function fetchTemplates() {
  return apiFetch('/templates');
}

// CREATE A TEMPLATE
export async function createTemplate(templateData) {
  return apiFetch('/templates', {
    method: 'POST',
    body: JSON.stringify(templateData),
  });
}

// UPDATE A TEMPLATE
export async function updateTemplate(templateId, templateData) {
  return apiFetch(`/templates/${templateId}`, {
    method: 'PUT',
    body: JSON.stringify(templateData),
  });
}

// DELETE A TEMPLATE
export async function deleteTemplate(templateId) {
  return apiFetch(`/templates/${templateId}`, {
    method: 'DELETE',
  });
}
