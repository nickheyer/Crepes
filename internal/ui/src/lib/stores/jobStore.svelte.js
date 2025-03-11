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
  }
};

export const state = $state({
  jobs: [],
  selectedJob: null,
  jobsLoading: false,
  createJobModal: false,
  editJobModal: false,
  formData: structuredClone(formDataBase),
});

// DERIVED STORES FOR DIFFERENT JOB STATUSES
const runningJobsDer = $derived(() => 
  state.jobs.filter(job => job.status === 'running')
);

export const runningJobs = () => runningJobsDer();

const completedJobsDer = $derived(() => 
  state.jobs.filter(job => job.status === 'completed')
);

export const completedJobs = () => completedJobsDer();

const failedJobsDer = $derived(() => 
  state.jobs.filter(job => job.status === 'failed')
);

export const failedJobs = () => failedJobsDer();

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
    addToast(`FAILED TO LOAD JOBS: ${error.message}`, 'error');
    return [];
  } finally {
    state.jobsLoading = false;
  }
}

// LOAD JOB DETAILS INCLUDING STATISTICS
export async function loadJobDetails(jobId) {
  try {
    // GET JOB FROM EXISTING JOBS OR FETCH IT
    const existingJobs = state.jobs;
    const existingJob = existingJobs.find(j => j.id === jobId);
    let job = existingJob;
    
    if (!job) {
      const fetchedJob = await fetchJob(jobId);
      job = fetchedJob;
    }
    
    // GET STATISTICS IF JOB EXISTS
    if (job) {
      try {
        const stats = await fetchJobStatistics(jobId);
        if (stats.success && stats.data) {
          job.statistics = stats.data;
        }
      } catch (err) {
        console.error("FAILED TO LOAD JOB STATISTICS:", err);
      }
    }
    
    state.selectedJob = job;
    return job;
  } catch (error) {
    addToast(`FAILED TO LOAD JOB DETAILS: ${error.message}`, 'error');
    throw error;
  }
}

// CREATE A NEW JOB
export async function createNewJob(jobData) {
  try {
    // ENSURE ID IS SET BY BACKEND
    const dataToSend = { ...jobData };
    if (dataToSend.id) {
      delete dataToSend.id;
    }
    
    const newJob = await createJob(dataToSend);
    state.jobs = [newJob, ...state.jobs];

    addToast('JOB CREATED SUCCESSFULLY', 'success');
    return newJob;
  } catch (error) {
    addToast(`FAILED TO CREATE JOB: ${error.message}`, 'error');
    throw error;
  }
}

// UPDATE AN EXISTING JOB
export async function updateExistingJob(jobId, jobData) {
  try {
    const updatedJob = await updateJob(jobId, jobData);
    state.jobs = state.jobs.map(job => job.id === jobId ? {...job, ...updatedJob} : job);

    addToast('JOB UPDATED SUCCESSFULLY', 'success');
    return updatedJob;
  } catch (error) {
    addToast(`FAILED TO UPDATE JOB: ${error.message}`, 'error');
    throw error;
  }
}

// DELETE A JOB
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

// START A JOB
export async function startJobById(jobId) {
  try {
    await startJob(jobId);
    state.jobs = state.jobs.map(job => job.id === jobId ? {...job, status: 'running'} : job);

    addToast('JOB STARTED SUCCESSFULLY', 'success');
    return true;
  } catch (error) {
    addToast(`FAILED TO START JOB: ${error.message}`, 'error');
    throw error;
  }
}

// STOP A JOB
export async function stopJobById(jobId) {
  try {
    await stopJob(jobId);
    state.jobs = state.jobs.map(job => job.id === jobId ? {...job, status: 'stopped'} : job);

    addToast('JOB STOPPED SUCCESSFULLY', 'success');
    return true;
  } catch (error) {
    addToast(`FAILED TO STOP JOB: ${error.message}`, 'error');
    throw error;
  }
}

// UPDATE A SPECIFIC STEP IN THE JOB WIZARD
export function updateJobWizardStep(step, data) {
  if (!data) return;
  
  // CREATE DEEP CLONE OF CURRENT STATE FOR IMMUTABILITY
  const updatedFormData = state.formData;
  
  // MERGE THE NEW DATA AT THE TOP LEVEL
  for (const [key, value] of Object.entries(data)) {
    // HANDLE ARRAYS PROPERLY - REPLACE RATHER THAN MERGE
    if (Array.isArray(value)) {
      updatedFormData.data[key] = [...value];
    } 
    // HANDLE OBJECTS (EXCEPT ARRAYS) WITH DEEP MERGE
    else if (value && typeof value === 'object' && !Array.isArray(value)) {
      updatedFormData.data[key] = updatedFormData.data[key] || {};
      for (const [nestedKey, nestedValue] of Object.entries(value)) {
        updatedFormData.data[key][nestedKey] = nestedValue;
      }
    } 
    // HANDLE PRIMITIVES WITH DIRECT ASSIGNMENT
    else {
      updatedFormData.data[key] = value;
    }
  }

  state.formData = updatedFormData;
  
  // MARK STEP AS VALID
  if (step && step > 0 && step <= 6) {
    updatedFormData.stepValidity[step] = true;
  }
}

// SET STEP VALIDITY
export function setStepValidity(step, isValid) {
  if (step && step > 0 && step <= 6) {
    state.formData.stepValidity[step] = isValid;
  }
}

// RESET JOB WIZARD STATE
export function resetJobWizard() {
  state.formData = structuredClone(formDataBase);
}

// GO TO A SPECIFIC STEP WHILE PERSISTING DATA
export function setJobWizardStep(step) {
  if (step > 0 && step <= 6) {
    state.formData.step = step;
  }
}
