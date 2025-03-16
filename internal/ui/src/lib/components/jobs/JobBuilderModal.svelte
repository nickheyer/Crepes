<script>
  import Button from "$lib/components/common/Button.svelte";
  import JobBuilder from "$lib/components/jobs/JobBuilder.svelte";
  import { jobsApi } from "$lib/utils/api";
  import { addToast } from "$lib/stores/uiStore.svelte";
  
  let { 
    isOpen = false,
    onClose = () => {},
    onJobCreated = () => {}
  } = $props();
  
  let job = $state({
    name: '',
    baseUrl: '',
    description: '',
    schedule: '',
    data: {
      pipeline: null,
      jobConfig: null
    }
  });
  let saving = $state(false);
  
  // RESET FORM WHEN MODAL OPENS
  $effect(() => {
    if (isOpen) {
      resetForm();
    }
  });
  
  function resetForm() {
    job = {
      name: '',
      baseUrl: '',
      description: '',
      schedule: '',
      data: {
        pipeline: null,
        jobConfig: null
      }
    };
    saving = false;
  }
  
  // HANDLE PIPELINE SAVE FROM JOBBUILDER
  function handlePipelineSave(pipelineData) {
    if (!pipelineData) return;
    
    job.data = {
      pipeline: JSON.stringify(pipelineData.pipeline),
      jobConfig: JSON.stringify(pipelineData.jobConfig)
    };
  }
  
  // SAVE THE NEW JOB
  async function saveJob() {
    if (!job.name) {
      addToast('Job name is required', 'error');
      return;
    }
    if (!job.baseUrl) {
      addToast('Base URL is required', 'error');
      return;
    }
    try {
      saving = true;
      const response = await jobsApi.create(job);
      addToast('Job created successfully', 'success');
      onJobCreated(response);
      resetForm();
    } catch (error) {
      addToast(`Failed to create job: ${error.message}`, 'error');
    } finally {
      saving = false;
    }
  }
</script>

{#if isOpen}
  <div class="modal modal-open fixed inset-0 z-50 overflow-y-auto bg-black bg-opacity-50 flex items-center justify-center">
    <div class="modal-box w-11/12 z-51 max-w-7xl h-[90vh] bg-base-200 flex flex-col rounded-lg shadow-xl">
      <!-- HEADER -->
      <div class="sticky top-0 z-10 bg-base-200 px-6 py-4 border-b border-base-300 flex justify-between items-center">
        <h3 class="text-xl font-bold">Create New Job</h3>
        <button class="btn btn-sm btn-circle" onclick={onClose}>✕</button>
      </div>
      
      <div class="flex-1 overflow-y-auto p-6">
        <div class="space-y-6">
          <!-- BASIC INFORMATION SECTION -->
          <div class="card bg-base-100 shadow-sm">
            <div class="card-body p-6">
              <h2 class="card-title text-lg font-bold mb-4">Basic Information</h2>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="form-control">
                  <label class="label" for="job-name">
                    <span class="label-text font-medium">Job Name <span class="text-error">*</span></span>
                  </label>
                  <input
                    id="job-name"
                    type="text"
                    bind:value={job.name}
                    placeholder="E.g., Product Scraper"
                    class="input input-bordered w-full"
                    required
                  />
                </div>
                <div class="form-control">
                  <label class="label" for="base-url">
                    <span class="label-text font-medium">Base URL <span class="text-error">*</span></span>
                  </label>
                  <input
                    id="base-url"
                    type="text"
                    bind:value={job.baseUrl}
                    placeholder="https://example.com"
                    class="input input-bordered w-full"
                    required
                  />
                </div>
                <div class="form-control md:col-span-2">
                  <label class="label" for="description">
                    <span class="label-text font-medium">Description</span>
                  </label>
                  <textarea
                    id="description"
                    bind:value={job.description}
                    placeholder="Describe the purpose of this job"
                    rows="3"
                    class="textarea textarea-bordered w-full"
                  ></textarea>
                </div>
                <div class="form-control">
                  <label class="label" for="schedule">
                    <span class="label-text font-medium">Schedule (CRON Expression)</span>
                    <span class="label-text-alt">Leave empty for manual execution only</span>
                  </label>
                  <input
                    id="schedule"
                    type="text"
                    bind:value={job.schedule}
                    placeholder="E.g., 0 0 * * * (daily at midnight)"
                    class="input input-bordered w-full"
                  />
                </div>
              </div>
            </div>
          </div>
          
          <!-- PIPELINE BUILDER SECTION -->
          <div class="card bg-base-100 shadow-sm">
            <div class="card-body p-0">
              <JobBuilder onSave={handlePipelineSave} />
            </div>
          </div>
        </div>
      </div>
      
      <!-- FOOTER WITH ACTIONS -->
      <div class="sticky bottom-0 z-10 bg-base-200 px-6 py-4 border-t border-base-300 flex justify-end space-x-4">
        <Button variant="ghost" onclick={onClose} disabled={saving}>Cancel</Button>
        <Button 
          variant="primary" 
          onclick={saveJob} 
          disabled={saving}
        >
          {#if saving}
            <span class="loading loading-spinner loading-xs mr-2"></span>
            Creating...
          {:else}
            Create Job
          {/if}
        </Button>
      </div>
    </div>
  </div>
{/if}
