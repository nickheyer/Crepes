import { addToast } from './uiStore.svelte';
import { jobsApi } from '$lib/utils/api';

const baseForm = {
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
    visualSelections: [],
    pipeline: null,
    jobConfig: null
  },
  errors: {}
};

export const state = $state({
  jobs: [],
  selectedJob: null,
  jobsLoading: false,
  createJobModal: false,
  editJobModal: false,
  formData: Object.assign({}, baseForm),
  updating: false
});

// COMPUTED PROPERTIES FOR FILTERED JOBS
const runningJobsFiltered = $derived(
  state.jobs.filter(job => job.status === 'running')
);
const completedJobsFiltered = $derived(
  state.jobs.filter(job => job.status === 'completed')
);
const failedJobsFiltered = $derived(
  state.jobs.filter(job => job.status === 'failed')
);

// EXPORT COMPUTED PROPERTIES
export const runningJobs = () => runningJobsFiltered;
export const completedJobs = () => completedJobsFiltered;
export const failedJobs = () => failedJobsFiltered;

// LOAD JOBS FROM API
export async function loadJobs() {
  state.jobsLoading = true;
  try {
    const data = await jobsApi.getAll();
    // NORMALIZE JOB DATA
    const jobArray = Array.isArray(data) ? data : [];
    const normalizedJobs = jobArray.map(job => ({
      ...job,
      // ENSURE ARRAYS AND OBJECTS
      selectors: Array.isArray(job.selectors) ? job.selectors : [],
      filters: Array.isArray(job.filters) ? job.filters : [],
      rules: job.rules || {},
      processing: job.processing || state.formData.data.processing,
      tags: Array.isArray(job.tags) ? job.tags : []
    }));
    state.jobs = normalizedJobs;
    return normalizedJobs;
  } catch (error) {
    console.error("Error loading jobs:", error);
    return [];
  } finally {
    state.jobsLoading = false;
  }
}

// LOAD JOB BY ID
export async function loadJob(jobId) {
  state.jobsLoading = true;
  try {
    const job = await jobsApi.getById(jobId);
    state.selectedJob = job;
    return job;
  } catch (error) {
    addToast(`Failed to load job: ${error.message}`, 'error');
    return null;
  } finally {
    state.jobsLoading = false;
  }
}

// CREATE NEW JOB
export async function createNewJob(jobData) {
  try {
    // ENSURE ID IS SET BY BACKEND
    const dataToSend = { ...jobData };
    if (dataToSend.id) {
      delete dataToSend.id;
    }
    const newJob = await jobsApi.create(dataToSend);
    state.jobs = [newJob, ...state.jobs];
    return newJob;
  } catch (error) {
    console.error("Error creating job:", error);
    throw error;
  }
}

// UPDATE EXISTING JOB
export async function updateExistingJob(jobId, jobData) {
  try {
    const updatedJob = await jobsApi.update(jobId, jobData);
    state.jobs = state.jobs.map(job => 
      job.id === jobId ? {...job, ...updatedJob} : job
    );
    if (state.selectedJob && state.selectedJob.id === jobId) {
      state.selectedJob = {...state.selectedJob, ...updatedJob};
    }
    return updatedJob;
  } catch (error) {
    console.error("Error updating job:", error);
    throw error;
  }
}

// DELETE JOB
export async function removeJob(jobId) {
  try {
    await jobsApi.delete(jobId);
    state.jobs = state.jobs.filter(job => job.id !== jobId);
    if (state.selectedJob && state.selectedJob.id === jobId) {
      state.selectedJob = null;
    }
    addToast('Job deleted successfully', 'success');
  } catch (error) {
    addToast(`Failed to delete job: ${error.message}`, 'error');
    throw error;
  }
}

// START JOB
export async function startJobById(jobId) {
  try {
    await jobsApi.start(jobId);
    state.jobs = state.jobs.map(job => 
      job.id === jobId ? {...job, status: 'running'} : job
    );
    if (state.selectedJob && state.selectedJob.id === jobId) {
      state.selectedJob = {...state.selectedJob, status: 'running'};
    }
    addToast('Job started successfully', 'success');
    return true;
  } catch (error) {
    addToast(`Failed to start job: ${error.message}`, 'error');
    throw error;
  }
}

// STOP JOB
export async function stopJobById(jobId) {
  try {
    await jobsApi.stop(jobId);
    state.jobs = state.jobs.map(job => 
      job.id === jobId ? {...job, status: 'stopped'} : job
    );
    if (state.selectedJob && state.selectedJob.id === jobId) {
      state.selectedJob = {...state.selectedJob, status: 'stopped'};
    }
    addToast('Job stopped successfully', 'success');
    return true;
  } catch (error) {
    addToast(`Failed to stop job: ${error.message}`, 'error');
    throw error;
  }
}

// UPDATE JOB PIPELINE
export async function updateJobPipeline(jobId, pipeline, jobConfig) {
  try {
    state.updating = true;
    const job = state.jobs.find(j => j.id === jobId);
    if (!job) throw new Error('Job not found');
    const updatedJob = {
      ...job,
      data: {
        ...job.data,
        pipeline: JSON.stringify(pipeline),
        jobConfig: JSON.stringify(jobConfig)
      }
    };
    const result = await jobsApi.update(jobId, updatedJob);
    state.jobs = state.jobs.map(j => 
      j.id === jobId ? {...j, ...result} : j
    );
    if (state.selectedJob && state.selectedJob.id === jobId) {
      state.selectedJob = {...state.selectedJob, ...result};
    }
    addToast('Pipeline updated successfully', 'success');
    return result;
  } catch (error) {
    addToast(`Failed to update pipeline: ${error.message}`, 'error');
    throw error;
  } finally {
    state.updating = false;
  }
}

// OPEN CREATE JOB MODAL
export function openCreateJobModal() {
  // RESET FORM DATA
  state.formData = Object.assign({}, baseForm);
  state.createJobModal = true;
}

// CLOSE CREATE JOB MODAL
export function closeCreateJobModal() {
  state.createJobModal = false;
}

// OPEN EDIT JOB MODAL
export function openEditJobModal(job) {
  state.selectedJob = job;
  state.formData.data = {...job};
  state.editJobModal = true;
}

// CLOSE EDIT JOB MODAL
export function closeEditJobModal() {
  state.editJobModal = false;
}
