<script>
  import { onMount } from 'svelte';
  import {
      state as jobState,
      runningJobs,
      completedJobs,
      failedJobs,
      loadJobs,
      startJobById,
      stopJobById,
      removeJob,
      openCreateJobModal
  } from "$lib/stores/jobStore.svelte.js";
  import Card from '$lib/components/common/Card.svelte';
  import Button from '$lib/components/common/Button.svelte';
  import JobBuilderModal from '$lib/components/jobs/JobBuilderModal.svelte';
  import { formatDate, formatRelativeTime } from '$lib/utils/formatters';
  import { addToast } from '$lib/stores/uiStore.svelte.js';
  import { jobsApi } from '$lib/utils/api.js';
  import Loading from "$lib/components/common/Loading.svelte";
  
  // LOCAL STATE
  let loading = $state(true);
  let jobFilter = $state('');
  let statusFilter = $state('');
  let confirmingDelete = $state(null);
  
  // LOAD JOBS ON MOUNT
  onMount(async () => {
    try {
      await loadJobs();
    } catch (error) {
      console.error('Error loading jobs:', error);
    } finally {
      loading = false;
    }
  });
  
  // HANDLE JOB MODAL
  function handleOpenNewJobModal() {
    jobState.createJobModal = true;
  }
  
  function handleCloseModal() {
    jobState.createJobModal = false;
  }
  
  async function handleJobCreated(newJob) {
    await loadJobs();
    jobState.createJobModal = false;
  }
  
  // HANDLE JOB ACTIONS
  async function handleStartJob(id) {
    try {
      await startJobById(id);
      addToast('Job started successfully', 'success');
    } catch (error) {
      console.error('Error starting job:', error);
      addToast(`Failed to start job: ${error.message}`, 'error');
    }
  }
  
  async function handleStopJob(id) {
    try {
      await stopJobById(id);
      addToast('Job stopped successfully', 'success');
    } catch (error) {
      console.error('Error stopping job:', error);
      addToast(`Failed to stop job: ${error.message}`, 'error');
    }
  }
  
  function confirmDelete(id) {
    confirmingDelete = id;
  }
  
  async function handleDeleteJob(id) {
    try {
      await removeJob(id);
      confirmingDelete = null;
      addToast('Job deleted successfully', 'success');
    } catch (error) {
      console.error('Error deleting job:', error);
      addToast(`Failed to delete job: ${error.message}`, 'error');
    }
  }
  
  function cancelDelete() {
    confirmingDelete = null;
  }
  
  // FILTER JOBS
  let filteredJobs = $derived(jobState.jobs.filter((job) => {
      // SEARCH
      const searchMatch = (!jobFilter || jobFilter.length === 0) || 
                            (job.name && job.name.toLowerCase().includes(jobFilter.toLowerCase())) || 
                            job.baseUrl.toLowerCase().includes(jobFilter.toLowerCase());
      // STATUS
      const statusMatch = (!statusFilter || statusFilter.length === 0) || job.status === statusFilter;
      return searchMatch && statusMatch;
  }));
</script>

<svelte:head>
  <title>Jobs | Crepes</title>
</svelte:head>

<section>
  <div class="flex justify-between items-center mb-6">
    <div>
      <h1 class="text-2xl font-bold mb-2">Jobs</h1>
      <p class="text-dark-300">Manage your scraping jobs</p>
    </div>
    <Button variant="primary" onclick={handleOpenNewJobModal}>
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-2" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd" />
      </svg>
      Create New Job
    </Button>
  </div>
  
  <!-- FILTERS -->
  <Card class="mb-6">
    <div class="flex flex-col md:flex-row md:items-center gap-4">
      <div class="flex-1">
        <label for="job-search" class="sr-only">Search jobs</label>
        <div class="relative">
          <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-dark-400" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
            </svg>
          </div>
          <input
            id="job-search"
            type="text"
            bind:value={jobFilter}
            placeholder="Search jobs by name or URL..."
            class="pl-10 pr-4 py-2 w-full rounded-md bg-base-700 border border-dark-600 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
        </div>
      </div>
      <div class="w-full md:w-auto">
        <select
            id="status-filter"
            bind:value={statusFilter}
            class="select select-bordered w-full md:w-auto"
        >
            <option value="">All Statuses</option>
            <option value="idle">Idle</option>
            <option value="running">Running</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
            <option value="stopped">Stopped</option>
        </select>
      </div>
    </div>
  </Card>
  
  {#if loading}
    <Loading size="lg" />
  {:else if jobState.jobs.length === 0}
    <Card class="text-center py-12">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto text-dark-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
      <h3 class="text-lg font-medium mb-2">No jobs found</h3>
      <p class="text-dark-400 mb-4">Start by creating your first job</p>
      <Button variant="primary" onclick={handleOpenNewJobModal}>Create New Job</Button>
    </Card>
  {:else if filteredJobs.length === 0}
    <Card class="text-center py-12">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto text-dark-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <h3 class="text-lg font-medium mb-2">No matching jobs</h3>
      <p class="text-dark-400 mb-4">Try adjusting your filters</p>
      <Button variant="outline" onclick={() => { jobFilter = ''; statusFilter = ''; }}>Clear Filters</Button>
    </Card>
  {:else}
    <div class="space-y-6">
      {#each filteredJobs as job (job.id)}
        <Card class="hover:shadow-lg transition-shadow">
          <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <div class="flex-1">
              <div class="flex flex-col md:flex-row md:items-center gap-3">
                <div>
                  <span class={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium
                    ${job.status === 'running' ? 'bg-green-600 text-green-100' : 
                      job.status === 'completed' ? 'bg-blue-600 text-blue-100' :
                      job.status === 'failed' ? 'bg-red-600 text-red-100' :
                      job.status === 'stopped' ? 'bg-yellow-600 text-yellow-100' :
                      'bg-gray-600 text-gray-100'}`}
                  >
                    {job.status || 'idle'}
                  </span>
                </div>
                <div class="flex-1">
                  <h3 class="font-medium">
                    <a href={`/jobs/${job.id}`} class="hover:text-primary-400">
                      {job.name || 'Unnamed Job'}
                    </a>
                  </h3>
                  <p class="text-sm text-dark-300 truncate">{job.baseUrl}</p>
                </div>
              </div>
              <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-y-2 gap-x-4 mt-3">
                <div>
                  <p class="text-xs text-dark-400">Last Run</p>
                  <p class="text-sm">{job.lastRun ? formatRelativeTime(job.lastRun) : 'Never'}</p>
                </div>
                <div>
                  <p class="text-xs text-dark-400">Next Run</p>
                  <p class="text-sm">{job.nextRun ? formatRelativeTime(job.nextRun) : 'Not scheduled'}</p>
                </div>
                <div>
                  <p class="text-xs text-dark-400">Assets</p>
                  <p class="text-sm">{job.assets?.length || 0}</p>
                </div>
                {#if job.data?.pipeline}
                  <div>
                    <p class="text-xs text-dark-400">Pipeline</p>
                    <p class="text-sm">
                      {#if job.data.pipeline}
                        {@const pipelineData = JSON.parse(job.data.pipeline)}
                        {pipelineData.length} stages, {pipelineData.reduce((total, stage) => total + stage.tasks.length, 0)} tasks
                      {:else}
                        Defined
                      {/if}
                    </p>
                  </div>
                {/if}
              </div>
            </div>
            <div class="flex flex-wrap gap-2">
              {#if job.status === 'running'}
                <Button 
                  variant="warning" 
                  size="sm" 
                  onclick={() => handleStopJob(job.id)}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
                  </svg>
                  Stop
                </Button>
              {:else}
                <Button 
                  variant="success" 
                  size="sm" 
                  onclick={() => handleStartJob(job.id)}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
                  </svg>
                  Start
                </Button>
              {/if}
              <Button 
                variant="primary" 
                size="sm" 
                onclick={() => window.location.href = `/jobs/${job.id}`}
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M10 12a2 2 0 100-4 2 2 0 000 4z" />
                  <path fill-rule="evenodd" d="M.458 10C1.732 5.943 5.522 3 10 3s8.268 2.943 9.542 7c-1.274 4.057-5.064 7-9.542 7S1.732 14.057.458 10zM14 10a4 4 0 11-8 0 4 4 0 018 0z" clip-rule="evenodd" />
                </svg>
                View
              </Button>
              {#if confirmingDelete === job.id}
                <div class="flex items-center space-x-2">
                  <span class="text-sm text-danger-400">Confirm?</span>
                  <Button 
                    variant="danger" 
                    size="sm" 
                    onclick={() => handleDeleteJob(job.id)}
                  >
                    Yes
                  </Button>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    onclick={cancelDelete}
                  >
                    No
                  </Button>
                </div>
              {:else}
                <Button 
                  variant="outline" 
                  size="sm" 
                  onclick={() => confirmDelete(job.id)}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1 text-danger-400" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
                  </svg>
                  Delete
                </Button>
              {/if}
            </div>
          </div>
        </Card>
      {/each}
    </div>
  {/if}
</section>

<!-- CREATE NEW JOB MODAL -->
<JobBuilderModal 
  isOpen={jobState.createJobModal} 
  onClose={handleCloseModal}
  onJobCreated={handleJobCreated}
/>