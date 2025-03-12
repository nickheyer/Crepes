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

// BASE FORM DATA SCHEMA WITH DEFAULT VALUES
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

// LOAD JOB DETAILS INCLUDING STATISTICS
export async function loadJobDetails(jobId) {
  try {
    // GET JOB FROM EXISTING JOBS OR FETCH IT
    const existingJob = state.jobs.find(j => j.id === jobId);
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
    throw error;
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

// STEP 1: BASIC INFO VALIDATION
function validateStep1() {
  const data = state.formData.data;
  const errors = {};
  
  // Validate job name
  const nameValidation = validateField(data.name, {
    required: true,
    minLength: 3,
    maxLength: 50,
  });
  
  if (!nameValidation.valid) {
    errors.name = nameValidation.message;
  }
  
  // Validate base URL
  if (!data.baseUrl) {
    errors.baseUrl = "Base URL is required";
  } else if (!isValidUrl(data.baseUrl)) {
    errors.baseUrl = "Please enter a valid URL";
  }
  
  // Validate description (optional)
  if (data.description && data.description.length > 500) {
    errors.description = "Description should be 500 characters or less";
  }
  
  // Update errors in store
  state.formData.errors = { ...state.formData.errors, step1: errors };
  
  // Step is valid if there are no errors
  const isValid = Object.keys(errors).length === 0;
  setStepValidity(1, isValid);
  
  return isValid;
}

// STEP 2: CONTENT SELECTION VALIDATION
function validateStep2() {
  const selectors = state.formData.data.selectors || [];
  const errors = {};
  
  // Check if we have at least one selector
  if (selectors.length === 0) {
    errors.general = "At least one selector is required";
  }
  
  // Check if we have both links and assets selectors
  const hasLinks = selectors.some(sel => sel.purpose === "links");
  const hasAssets = selectors.some(sel => sel.purpose === "assets");
  
  if (!hasLinks) {
    errors.links = "At least one 'links' selector is required";
  }
  
  if (!hasAssets) {
    errors.assets = "At least one 'assets' selector is required";
  }
  
  // Check individual selectors
  const selectorErrors = [];
  selectors.forEach((selector, index) => {
    const selectorError = {};
    
    if (!selector.name) {
      selectorError.name = "Selector name is required";
    }
    
    if (!selector.value) {
      selectorError.value = "Selector value is required";
    }
    
    if (!selector.purpose) {
      selectorError.purpose = "Selector purpose is required";
    }
    
    if (Object.keys(selectorError).length > 0) {
      selectorErrors[index] = selectorError;
    }
  });
  
  if (selectorErrors.length > 0) {
    errors.selectors = selectorErrors;
  }
  
  // Update errors in store
  state.formData.errors = { ...state.formData.errors, step2: errors };
  
  // Step is valid if there are links and assets selectors and no errors
  const isValid = hasLinks && hasAssets && Object.keys(errors).length === 0;
  setStepValidity(2, isValid);
  
  return isValid;
}

// STEP 3: FILTERING VALIDATION
function validateStep3() {
  const rules = state.formData.data.rules || {};
  const filters = state.formData.data.filters || [];
  const errors = {};
  
  // Validate URL patterns if provided
  if (rules.includeUrlPattern && !isValidRegex(rules.includeUrlPattern)) {
    errors.includeUrlPattern = "Invalid regex pattern";
  }
  
  if (rules.excludeUrlPattern && !isValidRegex(rules.excludeUrlPattern)) {
    errors.excludeUrlPattern = "Invalid regex pattern";
  }
  
  // Validate filters
  const filterErrors = [];
  filters.forEach((filter, index) => {
    const filterError = {};
    
    if (!filter.name) {
      filterError.name = "Filter name is required";
    }
    
    if (!filter.pattern) {
      filterError.pattern = "Filter pattern is required";
    } else if (!isValidRegex(filter.pattern)) {
      filterError.pattern = "Invalid regex pattern";
    }
    
    if (Object.keys(filterError).length > 0) {
      filterErrors[index] = filterError;
    }
  });
  
  if (filterErrors.length > 0) {
    errors.filters = filterErrors;
  }
  
  // Update errors in store
  state.formData.errors = { ...state.formData.errors, step3: errors };
  
  // Step is always valid unless there are specific errors
  const isValid = Object.keys(errors).length === 0;
  setStepValidity(3, isValid);
  
  return isValid;
}

// STEP 4: PROCESSING VALIDATION
function validateStep4() {
  const processing = state.formData.data.processing || {};
  const errors = {};
  
  // Validate image width if resize is enabled
  if (processing.imageResize) {
    const width = parseInt(processing.imageWidth);
    if (isNaN(width) || width < 100 || width > 10000) {
      errors.imageWidth = "Image width must be between 100 and 10000 pixels";
    }
  }
  
  // Update errors in store
  state.formData.errors = { ...state.formData.errors, step4: errors };
  
  // Step is valid if there are no errors
  const isValid = Object.keys(errors).length === 0;
  setStepValidity(4, isValid);
  
  return isValid;
}

// STEP 5: SCHEDULE VALIDATION
function validateStep5() {
  const schedule = state.formData.data.schedule;
  const errors = {};
  
  // Validate cron expression if provided
  if (schedule && !isValidCron(schedule)) {
    errors.schedule = "Invalid cron expression format";
  }
  
  // Update errors in store
  state.formData.errors = { ...state.formData.errors, step5: errors };
  
  // Step is valid if there are no errors
  const isValid = Object.keys(errors).length === 0;
  setStepValidity(5, isValid);
  
  return isValid;
}

// HELPER FUNCTION FOR REGEX VALIDATION
function isValidRegex(pattern) {
  if (!pattern) return true; // Empty pattern is valid
  
  try {
    new RegExp(pattern);
    return true;
  } catch (e) {
    return false;
  }
}

// UPDATE A SPECIFIC STEP IN THE JOB WIZARD
export function updateJobWizardStep(step, data) {
  if (!data || state.updating) return;
  
  try {
    // SET UPDATING FLAG TO PREVENT INFINITE LOOPS
    state.updating = true;
    
    // UPDATE THE FORM DATA WITH NEW VALUES
    const newFormData = { ...state.formData };
    
    // HANDLE EACH TOP-LEVEL FIELD
    for (const key in data) {
      if (data.hasOwnProperty(key)) {
        // HANDLE ARRAYS - REPLACE ENTIRELY
        if (Array.isArray(data[key])) {
          newFormData.data[key] = [...data[key]];
        }
        // HANDLE OBJECTS - MERGE
        else if (data[key] && typeof data[key] === 'object') {
          newFormData.data[key] = { ...newFormData.data[key], ...data[key] };
        }
        // HANDLE PRIMITIVES
        else {
          newFormData.data[key] = data[key];
        }
      }
    }
    
    // UPDATE STATE
    state.formData = newFormData;
    
    // VALIDATE CURRENT STEP
    validateCurrentStep(step);
  } finally {
    // CLEAR UPDATING FLAG
    state.updating = false;
  }
}

// VALIDATE THE CURRENT STEP
function validateCurrentStep(step) {
  switch (step) {
    case 1:
      validateStep1();
      break;
    case 2:
      validateStep2();
      break;
    case 3:
      validateStep3();
      break;
    case 4:
      validateStep4();
      break;
    case 5:
      validateStep5();
      break;
    case 6:
      // Summary step is always valid
      setStepValidity(6, true);
      break;
    default:
      break;
  }
}

// SET STEP VALIDITY
export function setStepValidity(step, isValid) {
  if (!state.updating && step && step > 0 && step <= 6) {
    state.formData.stepValidity = { ...state.formData.stepValidity, [step]: isValid };
  }
}

// GET ERRORS FOR A SPECIFIC STEP
export function getStepErrors(step) {
  return state.formData.errors[`step${step}`] || {};
}

// RESET JOB WIZARD STATE
export function resetJobWizard() {
  state.formData = structuredClone(formDataBase);
}

// GO TO A SPECIFIC STEP
export function setJobWizardStep(step) {
  if (step > 0 && step <= 6) {
    state.formData.step = step;
    validateCurrentStep(step);
  }
}

export function updateField(fieldPath, value) {
  if (state.updating) return;
  
  try {
    // SET UPDATING FLAG
    state.updating = true;
    
    // CREATE NEW STATE COPY
    const newFormData = { ...state.formData };
    
    // PARSE THE PATH AND UPDATE NESTED FIELD
    const pathParts = fieldPath.split('.');
    let current = newFormData.data;
    
    // NAVIGATE TO THE PARENT OBJECT
    for (let i = 0; i < pathParts.length - 1; i++) {
      const part = pathParts[i];
      
      // CREATE OBJECT IF IT DOESN'T EXIST
      if (!current[part]) {
        current[part] = {};
      }
      
      // HANDLE ARRAY INDICES
      if (Array.isArray(current[part]) && pathParts[i+1].match(/^\d+$/)) {
        const index = parseInt(pathParts[i+1]);
        if (index >= current[part].length) {
          current[part].push({});
        }
      }
      
      // MOVE TO NEXT LEVEL
      current = current[part];
    }
    
    // SET THE VALUE
    const lastPart = pathParts[pathParts.length - 1];
    current[lastPart] = value;
    
    // UPDATE STATE
    state.formData = newFormData;
    
    // VALIDATE CURRENT STEP
    validateCurrentStep(state.formData.step);
  } finally {
    // CLEAR UPDATING FLAG
    state.updating = false;
  }
}

export function addArrayItem(arrayPath, item) {
  if (state.updating) return;
  
  try {
    // SET UPDATING FLAG
    state.updating = true;
    
    // CREATE NEW STATE COPY
    const newFormData = { ...state.formData };
    
    // PARSE THE PATH AND FIND THE ARRAY
    const pathParts = arrayPath.split('.');
    let current = newFormData.data;
    
    // NAVIGATE TO THE ARRAY
    for (let i = 0; i < pathParts.length; i++) {
      const part = pathParts[i];
      
      // CREATE OBJECT/ARRAY IF IT DOESN'T EXIST
      if (!current[part]) {
        current[part] = [];
      }
      
      if (i === pathParts.length - 1) {
        // WE'VE REACHED THE ARRAY
        if (Array.isArray(current[part])) {
          current[part].push(item);
        }
      } else {
        // MOVE TO NEXT LEVEL
        current = current[part];
      }
    }
    
    // UPDATE STATE
    state.formData = newFormData;
    
    // VALIDATE CURRENT STEP
    validateCurrentStep(state.formData.step);
  } finally {
    // CLEAR UPDATING FLAG
    state.updating = false;
  }
}

export function removeArrayItem(arrayPath, index) {
  if (state.updating) return;
  
  try {
    // SET UPDATING FLAG
    state.updating = true;
    
    // CREATE NEW STATE COPY
    const newFormData = { ...state.formData };
    
    // PARSE THE PATH AND FIND THE ARRAY
    const pathParts = arrayPath.split('.');
    let current = newFormData.data;
    
    // NAVIGATE TO THE ARRAY
    for (let i = 0; i < pathParts.length; i++) {
      const part = pathParts[i];
      
      if (i === pathParts.length - 1) {
        // WE'VE REACHED THE ARRAY
        if (Array.isArray(current[part]) && index >= 0 && index < current[part].length) {
          current[part].splice(index, 1);
        }
      } else {
        // MOVE TO NEXT LEVEL
        current = current[part];
      }
    }
    
    // UPDATE STATE
    state.formData = newFormData;
    
    // VALIDATE CURRENT STEP
    validateCurrentStep(state.formData.step);
  } finally {
    // CLEAR UPDATING FLAG
    state.updating = false;
  }
}

export function updateArrayItem(arrayPath, index, item) {
  if (state.updating) return;
  
  try {
    // SET UPDATING FLAG
    state.updating = true;
    
    // CREATE NEW STATE COPY
    const newFormData = { ...state.formData };
    
    // PARSE THE PATH AND FIND THE ARRAY
    const pathParts = arrayPath.split('.');
    let current = newFormData.data;
    
    // NAVIGATE TO THE ARRAY
    for (let i = 0; i < pathParts.length; i++) {
      const part = pathParts[i];
      
      if (i === pathParts.length - 1) {
        // WE'VE REACHED THE ARRAY
        if (Array.isArray(current[part]) && index >= 0 && index < current[part].length) {
          current[part][index] = item;
        }
      } else {
        // MOVE TO NEXT LEVEL
        current = current[part];
      }
    }
    
    // UPDATE STATE
    state.formData = newFormData;
    
    // VALIDATE CURRENT STEP
    validateCurrentStep(state.formData.step);
  } finally {
    // CLEAR UPDATING FLAG
    state.updating = false;
  }
}
