<script>
  import { onMount } from 'svelte';
  import { formatRelativeTime } from '$lib/utils/formatters';
  import { state as jobState, loadJobs } from "$lib/stores/jobStore.svelte";
  
  // ACTIVITY STATE
  let activities = $state([]);
  let loading = $state(true);
  
  // GENERATE ACTIVITIES FROM JOBS
  $effect(() => {
    if (jobState.jobs.length > 0) {
      generateActivities();
      loading = false;
    }
  });
  
  onMount(async () => {
    if (jobState.jobs.length === 0) {
      await loadJobs();
    } else {
      generateActivities();
    }
    loading = false;
  });
  
  // GENERATE ACTIVITY ITEMS FROM JOBS
  function generateActivities() {
    const allActivities = [];
    
    // Add job status changes
    jobState.jobs.forEach(job => {
      if (job.lastRun) {
        allActivities.push({
          id: `job-run-${job.id}`,
          type: 'job-run',
          title: job.name || 'Unnamed Job',
          description: `Job was ${job.status}`,
          timestamp: new Date(job.lastRun),
          icon: getStatusIcon(job.status),
          iconColor: getStatusColor(job.status),
          url: `/jobs/${job.id}`
        });
      }
      
      // Add asset creation activities for jobs with assets
      if (job.assets && job.assets.length > 0) {
        // Get latest assets (up to 3)
        const latestAssets = [...job.assets]
          .sort((a, b) => new Date(b.date || 0) - new Date(a.date || 0))
          .slice(0, 3);
          
        latestAssets.forEach(asset => {
          if (asset.date) {
            allActivities.push({
              id: `asset-${asset.id}`,
              type: 'asset-created',
              title: asset.title || 'Untitled Asset',
              description: `New ${asset.type} asset downloaded`,
              timestamp: new Date(asset.date),
              icon: getAssetIcon(asset.type),
              iconColor: 'bg-info',
              url: `/assets?id=${asset.id}`,
              thumbnailUrl: asset.thumbnailPath ? `/thumbnails/${asset.thumbnailPath}` : null
            });
          }
        });
      }
    });
    
    activities = allActivities
      .sort((a, b) => b.timestamp - a.timestamp)
      .slice(0, 10);
  }
  
  function getStatusIcon(status) {
    switch(status) {
      case 'running':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
                </svg>`;
      case 'completed':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                </svg>`;
      case 'failed':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                </svg>`;
      case 'stopped':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
                </svg>`;
      default:
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 00-.867.5 1 1 0 11-1.731-1A3 3 0 0113 8a3.001 3.001 0 01-2 2.83V11a1 1 0 11-2 0v-1a1 1 0 011-1 1 1 0 100-2zm0 8a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
                </svg>`;
    }
  }
  
  function getStatusColor(status) {
    switch(status) {
      case 'running': return 'badge-success';
      case 'completed': return 'badge-info';
      case 'failed': return 'badge-error';
      case 'stopped': return 'badge-warning';
      default: return 'badge-ghost';
    }
  }
  
  function getAssetIcon(type) {
    switch(type) {
      case 'image':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z" clip-rule="evenodd" />
                </svg>`;
      case 'video':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M2 6a2 2 0 012-2h6a2 2 0 012 2v8a2 2 0 01-2 2H4a2 2 0 01-2-2V6zM14.553 7.106A1 1 0 0014 8v4a1 1 0 00.553.894l2 1A1 1 0 0018 13V7a1 1 0 00-1.447-.894l-2 1z" />
                </svg>`;
      case 'audio':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.707.707L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.707-3.707a1 1 0 011.09-.217zM14.657 2.929a1 1 0 011.414 0A9.972 9.972 0 0119 10a9.972 9.972 0 01-2.929 7.071 1 1 0 01-1.414-1.414A7.971 7.971 0 0017 10c0-2.21-.894-4.208-2.343-5.657a1 1 0 010-1.414zm-2.829 2.828a1 1 0 011.415 0A5.983 5.983 0 0115 10a5.984 5.984 0 01-1.757 4.243 1 1 0 01-1.415-1.415A3.984 3.984 0 0013 10a3.983 3.983 0 00-1.172-2.828 1 1 0 010-1.415z" clip-rule="evenodd" />
                </svg>`;
      case 'document':
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4zm2 6a1 1 0 011-1h6a1 1 0 110 2H7a1 1 0 01-1-1zm1 3a1 1 0 100 2h6a1 1 0 100-2H7z" clip-rule="evenodd" />
                </svg>`;
      default:
        return `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M8 4a3 3 0 00-3 3v4a3 3 0 006 0V7a1 1 0 112 0v4a5 5 0 01-10 0V7a5 5 0 0110 0v1h-2V7a3 3 0 00-3-3z" clip-rule="evenodd" />
                </svg>`;
    }
  }
</script>

{#if loading}
<div class="flex justify-center py-6">
  <span class="loading loading-spinner loading-md text-primary"></span>
</div>
{:else if activities.length === 0}
<div class="flex flex-col items-center justify-center py-6">
  <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-base-content opacity-40 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
  <p class="text-base-content opacity-60">No recent activity</p>
</div>
{:else}
<div class="space-y-4">
  {#each activities as activity, idx}
    <div class="flex">
      <div class="mr-4">
        <div class={`avatar placeholder ${activity.iconColor}`}>
          <div class="w-10 rounded-full text-white">
            {@html activity.icon}
          </div>
        </div>
        
        {#if idx !== activities.length - 1}
          <div class="flex justify-center">
            <div class="h-full w-0.5 bg-base-300 mt-1 mb-1"></div>
          </div>
        {/if}
      </div>
      
      <div class="min-w-0 flex-1 pb-5">
        <div>
          <div class="flex items-center justify-between mb-1">
            <a href={activity.url} class="link link-hover font-medium">
              {activity.title}
            </a>
            <span class="text-xs text-base-content opacity-60">
              {formatRelativeTime(activity.timestamp)}
            </span>
          </div>
        </div>
        <p class="text-sm text-base-content opacity-70">
          {activity.description}
        </p>
        {#if activity.thumbnailUrl}
          <div class="mt-2">
            <a href={activity.url} class="inline-block">
              <div class="avatar">
                <div class="w-16 h-16 rounded object-cover">
                  <img src={activity.thumbnailUrl} alt={activity.title} />
                </div>
              </div>
            </a>
          </div>
        {/if}
      </div>
    </div>
  {/each}
</div>
{/if}
