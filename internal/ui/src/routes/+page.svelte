<script>
  import { buttonClasses, cardClasses, inputClasses, selectClasses, checkboxClasses, formatDate } from '$lib/components';
  
  let jobs = $state([]);
  let showNewJobForm = $state(false);
  let initialLoading = $state(true);
  let pollingPaused = $state(false);
  
  // FORM FIELDS
  let newJob = $state({
    baseUrl: "",
    selectors: [],
    rules: {
      maxDepth: 3,
      maxAssets: 100,
      includeUrlPattern: "",
      excludeUrlPattern: "",
      timeout: 60,
      requestDelay: 2000,
      randomizeDelay: true
    },
    schedule: ""
  });

  // INITIALIZE WITH DEFAULT SELECTORS
  $effect(() => {
    if (newJob.selectors.length === 0) {
      resetSelectors();
    }
  });

  // SELECTOR OPTIONS
  const selectorTypes = [
    { value: "css", label: "CSS" },
    { value: "xpath", label: "XPath" }
  ];
  
  const selectorPurposes = [
    { value: "links", label: "Links to Follow" },
    { value: "assets", label: "Assets to Download" },
    { value: "title", label: "Title Metadata" },
    { value: "description", label: "Description Metadata" },
    { value: "author", label: "Author Metadata" },
    { value: "date", label: "Date Metadata" },
    { value: "pagination", label: "Pagination Links" }
  ];

  // FETCH JOBS FROM BACKEND - INITIAL LOAD
  async function initialFetchJobs() {
    try {
      // Only called ONCE for the initial load
      const res = await fetch("/api/jobs");
      if (!res.ok) throw new Error(`Failed to fetch jobs: ${res.status}`);
      
      jobs = await res.json() || [];
      initialLoading = false; // Only set once, never again
    } catch (err) {
      console.error("Error fetching jobs:", err);
      if (window.showToast) window.showToast("Failed to fetch jobs", "error");
      initialLoading = false;
    }
  }
  
  // FETCH JOBS FROM BACKEND - BACKGROUND POLLING
  async function pollJobs() {
    // Never set loading state for background polling
    if (pollingPaused || showNewJobForm) return;
    
    try {
      const res = await fetch("/api/jobs");
      if (!res.ok) return; // Silently fail for polling
      
      const newJobs = await res.json() || [];
      
      // Only update if there are changes
      if (hasJobsChanged(jobs, newJobs)) {
        jobs = newJobs;
      }
    } catch (err) {
      console.error("Error polling jobs:", err);
      // Don't show error toast for background polling failures
    }
  }
  
  // Check if jobs data has actually changed
  function hasJobsChanged(oldJobs, newJobs) {
    if (oldJobs.length !== newJobs.length) return true;
    
    // Compare jobs by their IDs and status
    for (let i = 0; i < oldJobs.length; i++) {
      const oldJob = oldJobs[i];
      const newJob = newJobs.find(j => j.id === oldJob.id);
      
      if (!newJob) return true;
      if (oldJob.status !== newJob.status) return true;
      if ((oldJob.assets?.length || 0) !== (newJob.assets?.length || 0)) return true;
    }
    
    return false;
  }

  // RESET SELECTORS TO DEFAULT
  function resetSelectors() {
    newJob.selectors = [
      { type: "css", value: "", for: "links" },
      { type: "css", value: "", for: "assets" },
      { type: "css", value: "", for: "title" },
      { type: "css", value: "", for: "description" },
      { type: "css", value: "", for: "pagination" }
    ];
  }

  // ADD NEW SELECTOR
  function addSelector() {
    newJob.selectors = [...newJob.selectors, { type: "css", value: "", for: "links" }];
  }

  // REMOVE SELECTOR
  function removeSelector(index) {
    newJob.selectors = newJob.selectors.filter((_, i) => i !== index);
  }

  // CREATE A NEW JOB
  async function createJob(event) {
    event.preventDefault();
    
    try {
      // VALIDATE FORM
      if (!newJob.baseUrl) {
        if (window.showToast) window.showToast("Base URL is required", "error");
        return;
      }
      
      // VALIDATE PAGINATION SELECTOR
      let hasPaginationSelector = false;
      for (const selector of newJob.selectors) {
        if (selector.for === "pagination" && selector.value.trim() !== "") {
          hasPaginationSelector = true;
          break;
        }
      }
      
      // IF NOT FOUND, ADD A "NONE" VALUE TO PREVENT AUTO-PAGINATION
      if (!hasPaginationSelector) {
        newJob.selectors.push({
          type: "css",
          value: "none", // Special value to indicate no pagination
          for: "pagination"
        });
      }
      
      // CREATE A CLEAN OBJECT FOR SUBMISSION
      const jobData = {
        baseUrl: newJob.baseUrl,
        selectors: newJob.selectors
          .filter(s => s.value.trim())
          .map(s => ({
            type: s.type,
            value: s.value,
            for: s.for
          })),
        rules: {
          maxDepth: Number(newJob.rules.maxDepth) || 3,
          maxAssets: Number(newJob.rules.maxAssets) || 100,
          includeUrlPattern: newJob.rules.includeUrlPattern,
          excludeUrlPattern: newJob.rules.excludeUrlPattern,
          timeout: Number(newJob.rules.timeout) || 60,
          requestDelay: Number(newJob.rules.requestDelay) || 2000,
          randomizeDelay: Boolean(newJob.rules.randomizeDelay)
        },
        schedule: newJob.schedule
      };
      
      if (jobData.selectors.filter(s => s.for !== "pagination").length === 0) {
        if (window.showToast) window.showToast("At least one selector is required", "error");
        return;
      }

      if (!jobData.selectors.some(s => s.for === "assets")) {
        if (window.showToast) window.showToast("At least one asset selector is required", "error");
        return;
      }

      if (window.showToast) window.showToast("Creating job...", "info");
      
      // Temporarily pause polling during form submission
      pollingPaused = true;
      
      const resp = await fetch("/api/jobs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(jobData)
      });
      
      if (!resp.ok) {
        const data = await resp.json();
        throw new Error("Server error: " + (data.error || resp.statusText));
      }

      if (window.showToast) window.showToast("Job created successfully", "success");
      
      // Manually get the full set of jobs, don't update incrementally
      const jobsResp = await fetch("/api/jobs");
      if (jobsResp.ok) {
        jobs = await jobsResp.json() || [];
      }
      
      showNewJobForm = false;
      resetNewJobForm();
    } catch (err) {
      console.error("Error creating job:", err);
      if (window.showToast) window.showToast(`Error creating job: ${err.message}`, "error");
    } finally {
      // Resume polling
      pollingPaused = false;
    }
  }

  // EXECUTE ACTION WITHOUT FLICKERING
  async function executeAction(actionFn, successMsg) {
    pollingPaused = true;
    
    try {
      if (window.showToast) window.showToast(`${successMsg}...`, "info");
      await actionFn();
      
      // Direct fetch without conditional updates to prevent flickering
      const res = await fetch("/api/jobs");
      if (res.ok) {
        jobs = await res.json() || [];
      }
      
      if (window.showToast) window.showToast(successMsg, "success");
    } catch (err) {
      console.error(`Error: ${err}`);
      if (window.showToast) window.showToast(`Error: ${err.message}`, "error");
    } finally {
      pollingPaused = false;
    }
  }

  // START A JOB
  async function startJob(id) {
    executeAction(
      async () => {
        const resp = await fetch(`/api/jobs/${id}/start`, { method: "POST" });
        if (!resp.ok) throw new Error(`Failed to start job: ${resp.status}`);
      },
      "Job started successfully"
    );
  }

  // STOP A JOB
  async function stopJob(id) {
    executeAction(
      async () => {
        const resp = await fetch(`/api/jobs/${id}/stop`, { method: "POST" });
        if (!resp.ok) throw new Error(`Failed to stop job: ${resp.status}`);
      },
      "Job stopped successfully"
    );
  }

  // DELETE A JOB
  async function deleteJob(id) {
    if (!confirm("Are you sure you want to delete this job? All assets will be removed.")) return;
    
    executeAction(
      async () => {
        const resp = await fetch(`/api/jobs/${id}`, { method: "DELETE" });
        if (!resp.ok) throw new Error(`Failed to delete job: ${resp.status}`);
      },
      "Job deleted successfully"
    );
  }

  // RESET THE NEW-JOB FORM TO DEFAULTS
  function resetNewJobForm() {
    newJob = {
      baseUrl: "",
      selectors: [],
      rules: {
        maxDepth: 3,
        maxAssets: 100,
        includeUrlPattern: "",
        excludeUrlPattern: "",
        timeout: 60,
        requestDelay: 2000,
        randomizeDelay: true
      },
      schedule: ""
    };
    resetSelectors();
  }

  // FORMAT JOB DATA
  function getStatusBadgeClass(status) {
    switch (status) {
      case "idle": return "bg-gray-500";
      case "running": return "bg-green-500";
      case "completed": return "bg-blue-500";
      case "failed": return "bg-red-500";
      case "stopped": return "bg-yellow-500";
      default: return "bg-gray-500";
    }
  }
  
  function truncateUrl(url, maxLength = 30) {
    if (!url) return "";
    if (url.length <= maxLength) return url;
    
    return url.substring(0, maxLength) + "...";
  }

  // LIFECYCLE SETUP
  let refreshTimer;
  
  $effect(() => {
    // Initial load - separate from polling
    initialFetchJobs();
    
    // Setup polling on a timer
    refreshTimer = setInterval(() => {
      if (!pollingPaused && !showNewJobForm) {
        pollJobs();
      }
    }, 5000);
    
    // Cleanup on component unmount
    return () => clearInterval(refreshTimer);
  });
</script>

<!-- HEADER -->
<header class="bg-gray-800 shadow">
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8 flex justify-between items-center">
    <div class="flex items-center space-x-4">
      <h1 class="text-3xl font-bold text-white">Crepes</h1>
      <span class="text-indigo-400 text-sm">Asset Scraper</span>
    </div>
    
    <div class="flex space-x-3">
      <a href="/gallery" class={buttonClasses.secondary} aria-label="View gallery">
        Gallery
      </a>
      <button
        onclick={() => (showNewJobForm = true)}
        class={buttonClasses.primary}
        aria-label="Create new job"
      >
        New Job
      </button>
    </div>
  </div>
</header>

<!-- MAIN CONTENT -->
<main class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8 w-full">
  <!-- DASHBOARD STATS -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
    <div class={cardClasses + " flex flex-col"}>
      <h3 class="text-lg font-medium text-gray-300 mb-2">Total Jobs</h3>
      <p class="text-3xl font-bold">{jobs.length}</p>
    </div>
    
    <div class={cardClasses + " flex flex-col"}>
      <h3 class="text-lg font-medium text-gray-300 mb-2">Running Jobs</h3>
      <p class="text-3xl font-bold text-green-400">
        {jobs.filter(job => job.status === "running").length}
      </p>
    </div>
    
    <div class={cardClasses + " flex flex-col"}>
      <h3 class="text-lg font-medium text-gray-300 mb-2">Total Assets</h3>
      <p class="text-3xl font-bold text-indigo-400">
        {jobs.reduce((total, job) => total + (job.assets?.length || 0), 0)}
      </p>
    </div>
  </div>

  <!-- JOBS TABLE -->
  <div class={cardClasses + " mb-6"}>
    <h2 class="text-xl font-semibold mb-4">Jobs</h2>

    {#if initialLoading}
      <div class="flex justify-center my-12">
        <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-indigo-500" role="status" aria-label="Loading jobs"></div>
      </div>
    {:else if jobs.length === 0}
      <div class="text-center my-12 py-8 bg-gray-700 rounded-lg">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto text-gray-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
        <p class="text-gray-400 text-lg">No jobs found</p>
        <p class="text-gray-500 mt-2">Create a new job to get started</p>
        <button
          onclick={() => (showNewJobForm = true)}
          class={buttonClasses.primary + " mt-4"}
          aria-label="Create first job"
        >
          Create Job
        </button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-700">
          <thead>
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">
                URL
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">
                Status
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">
                Last Run
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">
                Assets
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider">
                Schedule
              </th>
              <th class="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-700">
            {#each jobs as job (job.id)}
              <tr class="hover:bg-gray-700 transition">
                <td class="px-6 py-4 whitespace-nowrap">
                  <a
                    href="/jobs/{job.id}"
                    class="text-indigo-400 hover:text-indigo-300 transition flex items-center"
                    aria-label={`View job details for ${job.baseUrl}`}
                  >
                    <span class="truncate max-w-xs inline-block">{job.baseUrl}</span>
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 ml-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                    </svg>
                  </a>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span
                    class={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusBadgeClass(job.status)}`}
                  >
                    {job.status}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  {formatDate(job.lastRun)}
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <a href="/jobs/{job.id}" class="hover:text-indigo-400 transition">
                    {job.assets ? job.assets.length : 0}
                  </a>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                  {job.schedule || "Manual"}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  {#if job.status !== "running"}
                    <button
                      onclick={() => startJob(job.id)}
                      class="text-green-400 hover:text-green-300 transition mr-3"
                      aria-label={`Start job ${job.baseUrl}`}
                    >
                      Start
                    </button>
                  {:else}
                    <button
                      onclick={() => stopJob(job.id)}
                      class="text-yellow-400 hover:text-yellow-300 transition mr-3"
                      aria-label={`Stop job ${job.baseUrl}`}
                    >
                      Stop
                    </button>
                  {/if}
                  <button
                    onclick={() => deleteJob(job.id)}
                    class="text-red-400 hover:text-red-300 transition"
                    aria-label={`Delete job ${job.baseUrl}`}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</main>

<!-- CREATE NEW JOB MODAL -->
{#if showNewJobForm}
  <div
    class="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center p-4 z-50"
    role="dialog"
    aria-modal="true"
    aria-labelledby="job-form-title"
  >
    <div
      class="bg-gray-800 rounded-lg max-w-4xl w-full max-h-screen overflow-y-auto"
    >
      <!-- HEADER -->
      <div class="sticky top-0 bg-gray-800 z-10 px-6 py-4 border-b border-gray-700 flex justify-between items-center">
        <h3 id="job-form-title" class="text-lg font-medium">Create New Scraping Job</h3>
        <button
          onclick={() => (showNewJobForm = false)}
          class="text-gray-400 hover:text-white transition text-2xl focus:outline-none"
          aria-label="Close form"
        >
          &times;
        </button>
      </div>

      <!-- FORM -->
      <div class="p-6">
        <form onsubmit={createJob} class="space-y-6">
          <!-- BASE URL -->
          <div>
            <label for="baseUrlInput" class="block text-sm font-medium mb-1">
              Base URL <span class="text-red-500">*</span>
            </label>
            <input
              id="baseUrlInput"
              type="url"
              placeholder="https://example.com"
              required
              class={inputClasses}
              bind:value={newJob.baseUrl}
            />
            <p class="mt-1 text-xs text-gray-400">
              The starting URL for the scraper
            </p>
          </div>

          <!-- SELECTORS -->
          <div>
            <fieldset>
              <legend class="block text-sm font-medium mb-1">
                Selectors <span class="text-red-500">*</span>
              </legend>
              <div class="bg-gray-750 rounded-md p-4 space-y-3">
                {#each newJob.selectors as selector, i (i)}
                  <div class="grid grid-cols-12 gap-2">
                    <select
                      class={selectClasses + " col-span-2"}
                      bind:value={selector.type}
                      aria-label={`Selector ${i+1} type`}
                    >
                      {#each selectorTypes as type}
                        <option value={type.value}>{type.label}</option>
                      {/each}
                    </select>

                    <select
                      class={selectClasses + " col-span-3"}
                      bind:value={selector.for}
                      aria-label={`Selector ${i+1} purpose`}
                    >
                      {#each selectorPurposes as purpose}
                        <option value={purpose.value}>{purpose.label}</option>
                      {/each}
                    </select>

                    <input
                      type="text"
                      placeholder="Enter selector value (e.g. img.product-image)"
                      class={inputClasses + " col-span-6"}
                      bind:value={selector.value}
                      aria-label={`Selector ${i+1} value`}
                    />

                    <button
                      type="button"
                      class="col-span-1 px-3 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition focus:outline-none"
                      onclick={() => removeSelector(i)}
                      aria-label={`Remove selector ${i+1}`}
                    >
                      &times;
                    </button>
                  </div>
                {/each}
                
                <div class="flex justify-between mt-2">
                  <button
                    type="button"
                    class={buttonClasses.success}
                    onclick={addSelector}
                    aria-label="Add selector"
                  >
                    + Add Selector
                  </button>
                  
                  <button
                    type="button"
                    class={buttonClasses.secondary}
                    onclick={resetSelectors}
                    aria-label="Reset selectors"
                  >
                    Reset to Defaults
                  </button>
                </div>
                
                <div class="mt-2 text-xs text-gray-400 bg-gray-700 p-3 rounded-md">
                  <p class="font-semibold mb-1">Selector Examples:</p>
                  <ul class="list-disc pl-5 space-y-1">
                    <li><strong>Links:</strong> a.product-link</li>
                    <li><strong>Assets:</strong> img.product-image, video.media-player</li> 
                    <li><strong>Title:</strong> h1.product-title</li>
                    <li><strong>Description:</strong> div.product-description</li>
                    <li><strong>Pagination:</strong> a.next-page, a[rel="next"]</li>
                  </ul>
                </div>
              </div>
            </fieldset>
          </div>

          <!-- RULES SECTION -->
          <div>
            <fieldset>
              <legend class="block text-sm font-medium mb-1">
                Scraping Rules
              </legend>
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <!-- MAX DEPTH -->
                <div>
                  <label
                    for="maxDepthInput"
                    class="block text-xs mb-1 text-gray-400"
                  >
                    Max Link Depth
                  </label>
                  <input
                    id="maxDepthInput"
                    type="number"
                    min="0"
                    class={inputClasses}
                    bind:value={newJob.rules.maxDepth}
                    aria-label="Maximum link depth"
                  />
                  <p class="mt-1 text-xs text-gray-500">
                    How deep to follow links (0 = no limit)
                  </p>
                </div>
                
                <!-- MAX ASSETS -->
                <div>
                  <label
                    for="maxAssetsInput"
                    class="block text-xs mb-1 text-gray-400"
                  >
                    Max Assets
                  </label>
                  <input
                    id="maxAssetsInput"
                    type="number"
                    min="0"
                    class={inputClasses}
                    bind:value={newJob.rules.maxAssets}
                    aria-label="Maximum assets to collect"
                  />
                  <p class="mt-1 text-xs text-gray-500">
                    Maximum assets to download (0 = no limit)
                  </p>
                </div>
                
                <!-- INCLUDE PATTERN -->
                <div>
                  <label
                    for="includePatternInput"
                    class="block text-xs mb-1 text-gray-400"
                  >
                    Include URL Pattern
                  </label>
                  <input
                    id="includePatternInput"
                    type="text"
                    placeholder="Regular expression"
                    class={inputClasses}
                    bind:value={newJob.rules.includeUrlPattern}
                    aria-label="URL pattern to include"
                  />
                  <p class="mt-1 text-xs text-gray-500">
                    Only follow URLs matching this pattern
                  </p>
                </div>
                
                <!-- EXCLUDE PATTERN -->
                <div>
                  <label
                    for="excludePatternInput"
                    class="block text-xs mb-1 text-gray-400"
                  >
                    Exclude URL Pattern
                  </label>
                  <input
                    id="excludePatternInput"
                    type="text"
                    placeholder="Regular expression"
                    class={inputClasses}
                    bind:value={newJob.rules.excludeUrlPattern}
                    aria-label="URL pattern to exclude"
                  />
                  <p class="mt-1 text-xs text-gray-500">
                    Skip URLs matching this pattern
                  </p>
                </div>
                
                <!-- TIMEOUT -->
                <div>
                  <label
                    for="timeoutInput"
                    class="block text-xs mb-1 text-gray-400"
                  >
                    Timeout (seconds)
                  </label>
                  <input
                    id="timeoutInput"
                    type="number"
                    min="1"
                    class={inputClasses}
                    bind:value={newJob.rules.timeout}
                    aria-label="Request timeout in seconds"
                  />
                  <p class="mt-1 text-xs text-gray-500">
                    Maximum time per page request
                  </p>
                </div>
                
                <!-- REQUEST DELAY -->
                <div>
                  <label
                    for="requestDelayInput"
                    class="block text-xs mb-1 text-gray-400"
                  >
                    Request Delay (ms)
                  </label>
                  <input
                    id="requestDelayInput"
                    type="number"
                    min="0"
                    class={inputClasses}
                    bind:value={newJob.rules.requestDelay}
                    aria-label="Delay between requests in milliseconds"
                  />
                  <p class="mt-1 text-xs text-gray-500">
                    Delay between requests to avoid rate limiting
                  </p>
                </div>
                
                <!-- RANDOMIZE DELAY -->
                <div class="flex items-center space-x-2 mt-2">
                  <input
                    id="randomizeDelayCheck"
                    type="checkbox"
                    class={checkboxClasses}
                    bind:checked={newJob.rules.randomizeDelay}
                    aria-label="Randomize delay between requests"
                  />
                  <label
                    for="randomizeDelayCheck"
                    class="text-sm text-gray-400"
                  >
                    Randomize Delay (adds variation to avoid detection)
                  </label>
                </div>
              </div>
            </fieldset>
          </div>

          <!-- SCHEDULE -->
          <div>
            <label for="scheduleInput" class="block text-sm font-medium mb-1">
              Schedule (Cron Expression)
            </label>
            <input
              id="scheduleInput"
              type="text"
              class={inputClasses}
              bind:value={newJob.schedule}
              placeholder="e.g. 0 0 * * * (daily at midnight)"
              aria-label="Cron schedule expression"
            />
            <p class="mt-1 text-xs text-gray-400">
              Leave empty for manual execution only. Examples:
              <span class="font-mono">0 * * * *</span> (hourly),
              <span class="font-mono">0 0 * * *</span> (daily),
              <span class="font-mono">0 0 * * 0</span> (weekly on Sunday)
            </p>
          </div>

          <!-- FORM BUTTONS -->
          <div class="flex justify-end space-x-3 pt-4 border-t border-gray-700">
            <button
              type="button"
              class={buttonClasses.secondary}
              onclick={() => (showNewJobForm = false)}
              aria-label="Cancel creating job"
            >
              Cancel
            </button>
            <button
              type="submit"
              class={buttonClasses.primary}
              aria-label="Create new job"
            >
              Create Job
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
{/if}