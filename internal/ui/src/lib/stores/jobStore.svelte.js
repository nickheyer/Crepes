import { addToast } from './uiStore.svelte';
import { 
  fetchJobs, 
  createJob, 
  updateJob, 
  deleteJob, 
  startJob, 
  stopJob, 
  fetchJobStatistics 
} from '$lib/utils/api';
import { isValidUrl, isValidCron, validateField } from '$lib/utils/validation';

// THIS IS COMPLETELY OUT OF DATE. THE JOB STRUCTURE HAS ENTIRELY CHANGED WTF AM I DOING
export const formDataBase = {
  step: 1,
  data: {
    name: '',
    baseUrl: '',
    description: '',
    selectors: [],
    filters: [],
    rules: {
      maxDepth: 3,
      maxAssets: 0,
      maxPages: 0,
      maxConcurrent: 5,
      requestDelay: 0,
      randomizeDelay: false
    },
    schedule: '',
    processing: {
      thumbnails: true,
      metadata: true,
      deduplication: true,
      headless: true,
      imageResize: false,
      imageWidth: 1280,
      videoConvert: false,
      videoFormat: "mp4",
      extractText: false
    },
    tags: [],
    visualSelections: []
  },
  stepValidity: {
    1: false,
    2: false, 
    3: false,
    4: false,
    5: false,
    6: true
  },
  // TRACK VALIDATION ERRORS
  errors: {}
};

export const state = $state({
  jobs: [],
  selectedJob: null,
  jobsLoading: false,
  createJobModal: false,
  editJobModal: false,
  formData: Object.assign({}, formDataBase),
  updating: false
});

const runningJobsDer = $derived(
  state.jobs.filter(job => job.status === 'running')
);

export const runningJobs = () => runningJobsDer;

const completedJobsDer = $derived(
  state.jobs.filter(job => job.status === 'completed')
);

export const completedJobs = () => completedJobsDer;

const failedJobsDer = $derived(
  state.jobs.filter(job => job.status === 'failed')
);

export const failedJobs = () => failedJobsDer;

// LOAD JOBS FROM API
export async function loadJobs() {
  state.jobsLoading = true;
  try {
    const data = await fetchJobs();
    
    // ENSURE PROPER DATA STRUCTURE
    const jobArray = Array.isArray(data) ? data : [];
    
    // NORMALIZE JOB DATA
    const normalizedJobs = jobArray.map(job => ({
      ...job,
      // ENSURE ARRAYS AND OBJECTS
      selectors: Array.isArray(job.selectors) ? job.selectors : [],
      filters: Array.isArray(job.filters) ? job.filters : [],
      rules: job.rules || {},
      processing: job.processing || formDataBase.data.processing,
      tags: Array.isArray(job.tags) ? job.tags : []
    }));
    
    state.jobs = normalizedJobs;
    return normalizedJobs;
  } catch (error) {
    return [];
  } finally {
    state.jobsLoading = false;
  }
}


export async function createNewJob(jobData) {
  try {
    // ENSURE ID IS SET BY BACKEND
    const dataToSend = { ...jobData };
    if (dataToSend.id) {
      delete dataToSend.id;
    }
    
    const newJob = await createJob(dataToSend);
    state.jobs = [newJob, ...state.jobs];
    return newJob;
  } catch (error) {
    throw error;
  }
}

export async function updateExistingJob(jobId, jobData) {
  try {
    const updatedJob = await updateJob(jobId, jobData);
    state.jobs = state.jobs.map(job => job.id === jobId ? {...job, ...updatedJob} : job);
    return updatedJob;
  } catch (error) {
    throw error;
  }
}

export async function removeJob(jobId) {
  try {
    await deleteJob(jobId);
    state.jobs = state.jobs.filter(job => job.id !== jobId);
    addToast('JOB DELETED SUCCESSFULLY', 'success');
  } catch (error) {
    addToast(`FAILED TO DELETE JOB: ${error.message}`, 'error');
    throw error;
  }
}

export async function startJobById(jobId) {
  try {
    await startJob(jobId);
    state.jobs = state.jobs.map(job => job.id === jobId ? {...job, status: 'running'} : job);
    return true;
  } catch (error) {
    throw error;
  }
}

export async function stopJobById(jobId) {
  try {
    await stopJob(jobId);
    state.jobs = state.jobs.map(job => job.id === jobId ? {...job, status: 'stopped'} : job);
    return true;
  } catch (error) {
    throw error;
  }
}
