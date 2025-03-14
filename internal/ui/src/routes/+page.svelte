<script>
    import { onMount } from "svelte";

    import {
        state as jobState,
        runningJobs,
        completedJobs,
        failedJobs,
        loadJobs,
        startJobById,
        stopJobById
    } from "$lib/stores/jobStore.svelte.js";
    import { 
        state as assetState
    } from "$lib/stores/assetStore.svelte.js";
    import { loadAssets } from "$lib/stores/assetStore.svelte.js";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import StatCard from "$lib/components/dashboard/StatCard.svelte";
    import ActivityFeed from "$lib/components/dashboard/ActivityFeed.svelte";
    import QuickActions from "$lib/components/dashboard/QuickActions.svelte";
    import JobBuilder from "$lib/components/jobs/JobBuilder.svelte";
    import { formatDate, formatRelativeTime } from "$lib/utils/formatters";
    import { createJob } from "$lib/utils/api.js";
    import { addToast } from "$lib/stores/uiStore.svelte.js";

    // LOCAL STATE
    let loading = $state(true);
    let newJobModalOpen = $state(false);
    let newJob = $state({
      name: '',
      baseUrl: '',
      description: '',
      schedule: '',
      data: {}
    });
    let savingJob = $state(false);

    // LOAD DATA ON MOUNT
    onMount(async () => {
        try {
            await loadAssets();
            await loadJobs();
        } catch (error) {
            console.error("Error loading dashboard data:", error);
        } finally {
            loading = false;
        }
    });

    // CREATE NEW JOB
    function openNewJobModal() {
        // RESET NEW JOB OBJECT
        newJob = {
          name: '',
          baseUrl: '',
          description: '',
          schedule: '',
          data: {
            pipeline: null,
            jobConfig: null
          }
        };
        newJobModalOpen = true;
    }
    
    // SAVE NEW JOB
    async function handleSaveNewJob() {
        if (!newJob.name || !newJob.baseUrl) {
          addToast('Name and Base URL are required', 'error');
          return;
        }
        
        try {
          savingJob = true;
          const response = await createJob(newJob);
          newJobModalOpen = false;
          await loadJobs();
          addToast('Job created successfully', 'success');
        } catch (error) {
          console.error('Error creating job:', error);
          addToast(`Failed to create job: ${error.message}`, 'error');
        } finally {
          savingJob = false;
        }
    }
    
    // HANDLE JOB BUILDER SAVE
    function handleJobBuilderSave(event) {
        // GET THE PIPELINE AND CONFIG FROM THE BUILDER
        const { pipeline, jobConfig } = event.detail;
        
        // UPDATE THE NEW JOB OBJECT
        newJob.data = {
          ...newJob.data,
          pipeline: JSON.stringify(pipeline),
          jobConfig: JSON.stringify(jobConfig)
        };
    }
</script>

<svelte:head>
    <title>Dashboard | Crepes</title>
</svelte:head>

<div class="container mx-auto px-4">
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard
            title="Total Jobs"
            value={jobState.jobs.length}
            icon="briefcase"
            trend={null}
            href="/jobs"
        />

        <StatCard
            title="Running Jobs"
            value={runningJobs().length}
            icon="play"
            color="success"
            href="/jobs"
        />

        <StatCard
            title="Total Assets"
            value={assetState.assetCounts.total || 0}
            icon="photo"
            color="primary"
            trend={+15}
            href="/assets"
        />

        <StatCard
            title="Storage Used"
            value="1.2 GB"
            icon="database"
            trend={+5.4}
            href="/settings"
        />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <!-- RECENT JOBS -->
        <Card title="Recent Jobs" class="lg:col-span-2">
            {#if loading}
                <div class="flex justify-center items-center py-20">
                    <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary-500"></div>
                </div>
            {:else if jobState.jobs.length === 0}
                <div class="py-12 text-center">
                    <div class="flex justify-center mb-4">
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            class="h-16 w-16 text-dark-400"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                        >
                            <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                            />
                        </svg>
                    </div>
                    <h3 class="text-lg font-medium mb-2">No jobs found</h3>
                    <p class="text-dark-400 mb-4">
                        Start by creating your first job
                    </p>
                    <Button variant="primary" onclick={openNewJobModal}>Create New Job</Button>
                </div>
            {:else}
                <div class="overflow-x-auto">
                    <table class="w-full border-collapse">
                        <thead class="bg-base-800">
                            <tr>
                                <th class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">
                                    Name
                                </th>
                                <th class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">
                                    Status
                                </th>
                                <th class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">
                                    Last Run
                                </th>
                                <th class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider">
                                    Assets
                                </th>
                                <th class="px-4 py-3 text-right text-xs font-medium text-dark-300 uppercase tracking-wider">
                                    Actions
                                </th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-base-700">
                            {#each jobState.jobs.slice(0, 5) as job}
                                <tr>
                                    <td class="px-4 py-3">
                                        <a
                                            href={`/jobs/${job.id}`}
                                            class="font-medium hover:text-primary-400"
                                        >
                                            {job.name || "Unnamed Job"}
                                        </a>
                                        <div
                                            class="text-xs text-dark-400 truncate max-w-xs"
                                        >
                                            {job.baseUrl}
                                        </div>
                                    </td>
                                    <td class="px-4 py-3">
                                        <span
                                            class={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium
                                            ${job.status === "running" ? "bg-green-600 text-green-100" : 
                                              job.status === "completed" ? "bg-blue-600 text-blue-100" : 
                                              job.status === "failed" ? "bg-red-600 text-red-100" : 
                                              job.status === "stopped" ? "bg-yellow-600 text-yellow-100" : 
                                              "bg-gray-600 text-gray-100"}`}
                                        >
                                            {job.status || "idle"}
                                        </span>
                                    </td>
                                    <td class="px-4 py-3">
                                        {job.lastRun
                                            ? formatRelativeTime(job.lastRun)
                                            : "Never"}
                                    </td>
                                    <td class="px-4 py-3">
                                        {job.assets?.length || 0}
                                    </td>
                                    <td class="px-4 py-3 text-right">
                                        {#if job.status === "running"}
                                            <Button
                                                variant="warning"
                                                size="sm"
                                                onclick={() => stopJobById(job.id)}
                                            >
                                                Stop
                                            </Button>
                                        {:else}
                                            <Button
                                                variant="success"
                                                size="sm"
                                                onclick={() => startJobById(job.id)}
                                            >
                                                Start
                                            </Button>
                                        {/if}
                                        <a
                                            href={`/jobs/${job.id}`}
                                            class="inline-flex items-center px-2.5 py-1.5 ml-2 border border-transparent text-xs font-medium rounded
                                                  bg-base-800 hover:bg-base-700 text-dark-100"
                                        >
                                            View
                                        </a>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>

                <div class="flex justify-end mt-4">
                    <a
                        href="/jobs"
                        class="text-primary-400 hover:text-primary-300 flex items-center"
                    >
                        View All Jobs →
                    </a>
                </div>
            {/if}
        </Card>

        <!-- QUICK ACTIONS -->
        <Card title="Quick Actions">
            <QuickActions />
        </Card>

        <!-- ACTIVITY FEED -->
        <Card title="Recent Activity" class="lg:col-span-2">
            <ActivityFeed />
        </Card>

        <!-- ASSET STATS -->
        <Card title="Asset Distribution">
            {#if loading}
                <div class="flex justify-center items-center py-20">
                    <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary-500"></div>
                </div>
            {:else}
                <div class="space-y-3">
                    <!-- Images -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Images</span>
                            <span class="text-dark-400"
                                >{assetState.assetCounts.image || 0}</span
                            >
                        </div>
                        <div class="w-full bg-base-700 rounded-full h-2">
                            <div 
                                class="bg-blue-600 h-2 rounded-full" 
                                style={`width: ${assetState.assetCounts.total > 0 ? ((assetState.assetCounts.image || 0) / assetState.assetCounts.total) * 100 : 0}%`}
                            ></div>
                        </div>
                    </div>

                    <!-- Videos -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Videos</span>
                            <span class="text-dark-400"
                                >{assetState.assetCounts.video || 0}</span
                            >
                        </div>
                        <div class="w-full bg-base-700 rounded-full h-2">
                            <div 
                                class="bg-red-600 h-2 rounded-full" 
                                style={`width: ${assetState.assetCounts.total > 0 ? ((assetState.assetCounts.video || 0) / assetState.assetCounts.total) * 100 : 0}%`}
                            ></div>
                        </div>
                    </div>

                    <!-- Audio -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Audio</span>
                            <span class="text-dark-400"
                                >{assetState.assetCounts.audio || 0}</span
                            >
                        </div>
                        <div class="w-full bg-base-700 rounded-full h-2">
                            <div 
                                class="bg-green-600 h-2 rounded-full" 
                                style={`width: ${assetState.assetCounts.total > 0 ? ((assetState.assetCounts.audio || 0) / assetState.assetCounts.total) * 100 : 0}%`}
                            ></div>
                        </div>
                    </div>

                    <!-- Documents -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Documents</span>
                            <span class="text-dark-400"
                                >{assetState.assetCounts.document || 0}</span
                            >
                        </div>
                        <div class="w-full bg-base-700 rounded-full h-2">
                            <div 
                                class="bg-yellow-600 h-2 rounded-full" 
                                style={`width: ${assetState.assetCounts.total > 0 ? ((assetState.assetCounts.document || 0) / assetState.assetCounts.total) * 100 : 0}%`}
                            ></div>
                        </div>
                    </div>

                    <!-- Other -->
                    <div>
                        <div class="flex justify-between items-center mb-1">
                            <span class="font-medium">Other</span>
                            <span class="text-dark-400"
                                >{assetState.assetCounts.unknown || 0}</span
                            >
                        </div>
                        <div class="w-full bg-base-700 rounded-full h-2">
                            <div 
                                class="bg-purple-600 h-2 rounded-full" 
                                style={`width: ${assetState.assetCounts.total > 0 ? ((assetState.assetCounts.unknown || 0) / assetState.assetCounts.total) * 100 : 0}%`}
                            ></div>
                        </div>
                    </div>
                </div>

                <div class="flex justify-end mt-4 pt-3 border-t border-base-700">
                    <a
                        href="/assets"
                        class="text-primary-400 hover:text-primary-300 flex items-center"
                    >
                        View All Assets →
                    </a>
                </div>
            {/if}
        </Card>
    </div>
</div>

<!-- CREATE NEW JOB MODAL -->
{#if newJobModalOpen}
  <div class="modal modal-open">
    <div class="modal-box max-w-7xl w-full h-[90vh] overflow-y-auto">
      <div class="sticky top-0 bg-base-800 py-2 z-10 flex justify-between items-center">
        <h3 class="font-bold text-lg">Create New Job</h3>
        <button 
          onclick={() => newJobModalOpen = false} 
          class="bg-base-700 hover:bg-base-600 p-2 rounded-full"
          aria-label="createnewjobmodal"
        >
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      
      <div class="space-y-6 mt-4">
        <!-- BASIC INFO SECTION -->
        <Card title="Basic Information">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label for="job-name" class="block text-sm font-medium text-dark-300 mb-1">
                Job Name <span class="text-danger-500">*</span>
              </label>
              <input
                id="job-name"
                type="text"
                bind:value={newJob.name}
                placeholder="E.g., Product Scraper"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>
            
            <div>
              <label for="base-url" class="block text-sm font-medium text-dark-300 mb-1">
                Base URL <span class="text-danger-500">*</span>
              </label>
              <input
                id="base-url"
                type="text"
                bind:value={newJob.baseUrl}
                placeholder="https://example.com"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
            </div>
            
            <div class="md:col-span-2">
              <label for="description" class="block text-sm font-medium text-dark-300 mb-1">
                Description
              </label>
              <textarea
                id="description"
                bind:value={newJob.description}
                placeholder="Describe the purpose of this job..."
                rows="3"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              ></textarea>
            </div>
            
            <div>
              <label for="schedule" class="block text-sm font-medium text-dark-300 mb-1">
                Schedule (CRON Expression)
              </label>
              <input
                id="schedule"
                type="text"
                bind:value={newJob.schedule}
                placeholder="E.g., 0 0 * * * (daily at midnight)"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              />
              <p class="text-xs text-dark-400 mt-1">
                Leave empty for manual execution only
              </p>
            </div>
          </div>
        </Card>
        
        <!-- PIPELINE BUILDER SECTION -->
        <Card title="Pipeline Builder">
          <JobBuilder on:save={handleJobBuilderSave} />
        </Card>
        
        <!-- ACTIONS -->
        <div class="flex justify-end space-x-3">
          <Button variant="outline" onclick={() => newJobModalOpen = false}>
            Cancel
          </Button>
          <Button variant="primary" onclick={handleSaveNewJob} disabled={savingJob}>
            {#if savingJob}
              <div class="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-white mr-2"></div>
              Creating...
            {:else}
              Create Job
            {/if}
          </Button>
        </div>
      </div>
    </div>
    <div
      class="modal-backdrop"
      onclick={() => newJobModalOpen = false}
      onkeydown={() => {}}
      role="button"
      aria-label="Close modal"
      tabindex="0"
    >
      <button>close</button>
    </div>
  </div>
{/if}