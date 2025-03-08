<script>
  import { onMount } from "svelte";
  import { createEventDispatcher } from "svelte";
  import { isValidUrl, validateField } from "$lib/utils/validation";
  
  // Create dispatch function
  const dispatch = createEventDispatcher();
  
  // Props
  let { formData = {} } = $props();
  
  // Local state
  let jobName = $state(formData.name || "");
  let baseUrl = $state(formData.baseUrl || "");
  let description = $state(formData.description || "");
  let tags = $state(formData.tags || []);
  let newTag = $state("");
  
  // Validation state
  let errors = $state({
    jobName: "",
    baseUrl: "",
    description: "",
  });
  
  // Validate step
  function validate() {
    const newErrors = {
      jobName: "",
      baseUrl: "",
      description: "",
    };
    // Validate job name
    const nameValidation = validateField(jobName, {
      required: true,
      minLength: 3,
      maxLength: 50,
    });
    if (!nameValidation.valid) {
      newErrors.jobName = nameValidation.message;
    }
    // Validate base URL
    if (!baseUrl) {
      newErrors.baseUrl = "Base URL is required";
    } else if (!isValidUrl(baseUrl)) {
      newErrors.baseUrl = "Please enter a valid URL";
    }
    // Validate description (optional)
    if (description && description.length > 500) {
      newErrors.description = "Description should be 500 characters or less";
    }
    errors = newErrors;
    // Step is valid if there are no errors
    const isValid = !Object.values(newErrors).some((error) => error);
    // Emit validation result
    dispatch("validate", isValid);
    return isValid;
  }
  
  // Update formData and validate
  function updateFormData() {
    const updatedData = {
      name: jobName,
      baseUrl,
      description,
      tags: [...tags],
    };
    const isValid = validate();
    if (isValid) {
      dispatch("update", updatedData);
    }
  }
  
  function addTag() {
    if (newTag && !tags.includes(newTag)) {
      tags = [...tags, newTag];
      newTag = "";
      updateFormData();
    }
  }
  
  function removeTag(tag) {
    tags = tags.filter((t) => t !== tag);
    updateFormData();
  }
  
  // Initialize validation on mount and any input change
  onMount(() => {
    validate();
  });
  
  // Watch for input changes
  $effect(() => {
    updateFormData();
  });
</script>
<div>
  <h2 class="text-xl font-semibold mb-4">Basic Information</h2>
  <p class="text-dark-300 mb-6">
    Provide the basic details for your scraping job
  </p>
  <div class="space-y-6">
    <!-- Job Name Input -->
    <div>
      <label for="job-name" class="block text-sm font-medium text-white mb-1"
        >Job Name <span class="text-danger-500">*</span></label
      >
      <input
        id="job-name"
        type="text"
        bind:value={jobName}
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 {errors.jobName
          ? 'border-danger-500'
          : ''}"
        placeholder="My Scraping Job"
      />
      {#if errors.jobName}
        <p class="mt-1 text-sm text-danger-500">{errors.jobName}</p>
      {/if}
    </div>
    <!-- Base URL Input -->
    <div>
      <label for="base-url" class="block text-sm font-medium text-white mb-1"
        >Base URL <span class="text-danger-500">*</span></label
      >
      <input
        id="base-url"
        type="url"
        bind:value={baseUrl}
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 {errors.baseUrl
          ? 'border-danger-500'
          : ''}"
        placeholder="https://example.com"
      />
      {#if errors.baseUrl}
        <p class="mt-1 text-sm text-danger-500">{errors.baseUrl}</p>
      {:else}
        <p class="mt-1 text-sm text-dark-400">
          This is the starting URL for your scraping job
        </p>
      {/if}
    </div>
    <!-- Description Textarea -->
    <div>
      <label for="description" class="block text-sm font-medium text-white mb-1"
        >Description</label
      >
      <textarea
        id="description"
        bind:value={description}
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 {errors.description
          ? 'border-danger-500'
          : ''}"
        rows="3"
        placeholder="Describe what this job does and what assets you want to collect"
      ></textarea>
      {#if errors.description}
        <p class="mt-1 text-sm text-danger-500">{errors.description}</p>
      {:else}
        <p class="mt-1 text-sm text-dark-400">
          {description.length}/500 characters
        </p>
      {/if}
    </div>
    <!-- Tags Input -->
    <div>
      <label for="tags" class="block text-sm font-medium text-white mb-1"
        >Tags</label
      >
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
      {#if tags.length > 0}
        <div class="mt-2 flex flex-wrap gap-2">
          {#each tags as tag}
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
