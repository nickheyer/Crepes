import { writable, derived } from 'svelte/store';
import { addToast } from './uiStore';
import { fetchJobs, createJob, updateJob, deleteJob, startJob, stopJob } from '$lib/utils/api';

// Job store state
export const jobs = writable([]);
export const selectedJob = writable(null);
export const jobsLoading = writable(false);
export const createJobModal = writable(false);
export const jobWizardState = writable({
  step: 1,
  data: {
    name: '',
    baseUrl: '',
    description: '',
    selectors: [],
    filters: [],
    schedule: null,
    processing: {
      thumbnails: true,
      metadata: true
    }
  }
});

// Derived stores for different job statuses
export const runningJobs = derived(jobs, $jobs => 
  $jobs.filter(job => job.status === 'running')
);

export const completedJobs = derived(jobs, $jobs => 
  $jobs.filter(job => job.status === 'completed')
);

export const failedJobs = derived(jobs, $jobs => 
  $jobs.filter(job => job.status === 'failed')
);

// Load jobs from API
export async function loadJobs() {
  jobsLoading.set(true);
  
  try {
    const data = await fetchJobs();
    jobs.set(data);
    return data;
  } catch (error) {
    addToast(`Failed to load jobs: ${error.message}`, 'error');
    return [];
  } finally {
    jobsLoading.set(false);
  }
}

// Create a new job
export async function createNewJob(jobData) {
  try {
    const newJob = await createJob(jobData);
    jobs.update(allJobs => [newJob, ...allJobs]);
    addToast('Job created successfully', 'success');
    return newJob;
  } catch (error) {
    addToast(`Failed to create job: ${error.message}`, 'error');
    throw error;
  }
}

// Update an existing job
export async function updateExistingJob(jobId, jobData) {
  try {
    const updatedJob = await updateJob(jobId, jobData);
    jobs.update(allJobs => 
      allJobs.map(job => job.id === jobId ? {...job, ...updatedJob} : job)
    );
    addToast('Job updated successfully', 'success');
    return updatedJob;
  } catch (error) {
    addToast(`Failed to update job: ${error.message}`, 'error');
    throw error;
  }
}

// Delete a job
export async function removeJob(jobId) {
  try {
    await deleteJob(jobId);
    jobs.update(allJobs => allJobs.filter(job => job.id !== jobId));
    addToast('Job deleted successfully', 'success');
  } catch (error) {
    addToast(`Failed to delete job: ${error.message}`, 'error');
    throw error;
  }
}

// Start a job
export async function startJobById(jobId) {
  try {
    await startJob(jobId);
    jobs.update(allJobs => 
      allJobs.map(job => job.id === jobId ? {...job, status: 'running'} : job)
    );
    addToast('Job started successfully', 'success');
  } catch (error) {
    addToast(`Failed to start job: ${error.message}`, 'error');
    throw error;
  }
}

// Stop a job
export async function stopJobById(jobId) {
  try {
    await stopJob(jobId);
    jobs.update(allJobs => 
      allJobs.map(job => job.id === jobId ? {...job, status: 'stopped'} : job)
    );
    addToast('Job stopped successfully', 'success');
  } catch (error) {
    addToast(`Failed to stop job: ${error.message}`, 'error');
    throw error;
  }
}

// Reset job wizard state
export function resetJobWizard() {
  jobWizardState.set({
    step: 1,
    data: {
      name: '',
      baseUrl: '',
      description: '',
      selectors: [],
      filters: [],
      schedule: null,
      processing: {
        thumbnails: true,
        metadata: true
      }
    }
  });
}

// Update a specific step in the job wizard
export function updateJobWizardStep(step, data) {
  jobWizardState.update(state => ({
    ...state,
    data: {
      ...state.data,
      ...data
    }
  }));
}

// Go to a specific step in the job wizard
export function setJobWizardStep(step) {
  jobWizardState.update(state => ({
    ...state,
    step
  }));
}
