<script>
  import { state as jobState } from "$lib/stores/jobStore.svelte";

  let newTag = $state("");

  function addTag() {
    if (newTag && !jobState.formData.data.tags.includes(newTag)) {
      jobState.formData.data.tags = [...jobState.formData.data.tags, newTag];
      newTag = "";
    }
  }

  function removeTag(tag) {
    jobState.formData.data.tags = jobState.formData.data.tags.filter((t) => t !== tag);
  }
</script>

<div>
  <h2 class="text-xl font-semibold mb-4">Basic Information</h2>
  <p class="text-dark-300 mb-6">
    Provide the basic details for your scraping job
  </p>
  <div class="space-y-6">
    <!-- JOB NAME INPUT -->
    <div>
      <label for="job-name" class="block text-sm font-medium text-white mb-1">
        Job Name <span class="text-danger-500">*</span>
      </label>
      <input
        id="job-name"
        type="text"
        bind:value={jobState.formData.data.name}
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
        placeholder="my scraping job"
      />
    </div>
    
    <!-- BASE URL INPUT -->
    <div>
      <label for="base-url" class="block text-sm font-medium text-white mb-1">
        Base URL <span class="text-danger-500">*</span>
      </label>
      <input
        id="base-url"
        type="url"
        bind:value={jobState.formData.data.baseUrl}
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
        placeholder="https://example.com"
      />
      <p class="mt-1 text-sm text-dark-400">
        This is the starting URL for your scraping job
      </p>
    </div>
    
    <!-- DESCRIPTION TEXTAREA -->
    <div>
      <label for="description" class="block text-sm font-medium text-white mb-1">
        Description
      </label>
      <textarea
        id="description"
        bind:value={jobState.formData.data.description}
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
        rows="3"
        placeholder="describe what this job does and what assets you want to collect"
      ></textarea>
      <p class="mt-1 text-sm text-dark-400">
        {jobState.formData.data.description?.length || 0}/500 characters
      </p>
    </div>
    
    <!-- TAGS INPUT -->
    <div>
      <label for="tags" class="block text-sm font-medium text-white mb-1">
        Tags
      </label>
      <div class="flex">
        <input
          id="tags"
          type="text"
          bind:value={newTag}
          class="flex-1 px-3 py-2 bg-base-700 border border-dark-600 rounded-l-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          placeholder="Add tag"
          onkeydown={(e) => e.key === "Enter" && addTag()}
        />
        <button
          type="button"
          class="px-4 py-2 bg-base-600 hover:bg-base-500 rounded-r-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
          onclick={addTag}
        >
          Add
        </button>
      </div>
      {#if jobState.formData.data.tags.length > 0}
        <div class="mt-2 flex flex-wrap gap-2">
          {#each jobState.formData.data.tags as tag}
            <div
              class="flex items-center bg-base-700 text-sm rounded-full px-3 py-1"
            >
              {tag}
              <button
                type="button"
                class="ml-1.5 text-dark-400 hover:text-white focus:outline-none"
                onclick={() => removeTag(tag)}
                aria-label="removetag"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-4 w-4"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                >
                  <path
                    fill-rule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clip-rule="evenodd"
                  />
                </svg>
              </button>
            </div>
          {/each}
        </div>
      {/if}
      <p class="mt-1 text-sm text-dark-400">
        Tags help you organize and filter your jobs
      </p>
    </div>
  </div>
</div>
