<script>
    import { onMount } from "svelte";
    import { createEventDispatcher } from "svelte";
    import { getCronDescription } from "$lib/utils/formatters";

    const dispatch = createEventDispatcher();


    // Props
    let { formData = {} } = $props();

    // Initialize
    onMount(() => {
        validate();
    });

    // Validate the step (always valid as it's just a summary)
    function validate() {
        dispatch("validate", true);
        return true;
    }
</script>

<div>
    <h2 class="text-xl font-semibold mb-4">Review & Confirm</h2>
    <p class="text-dark-300 mb-6">
        Review your job configuration before creation
    </p>

    <div class="space-y-6">
        <!-- Basic Information -->
        <div class="bg-base-800 rounded-lg p-4">
            <h3 class="text-sm font-medium text-primary-400 mb-3">
                Basic Information
            </h3>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <p class="text-xs text-dark-400">Job Name</p>
                    <p class="text-sm">{formData.name || "Unnamed Job"}</p>
                </div>

                <div>
                    <p class="text-xs text-dark-400">Base URL</p>
                    <p class="text-sm break-all">
                        {formData.baseUrl || "No URL specified"}
                    </p>
                </div>

                {#if formData.description}
                    <div class="md:col-span-2">
                        <p class="text-xs text-dark-400">Description</p>
                        <p class="text-sm">{formData.description}</p>
                    </div>
                {/if}
            </div>
        </div>

        <!-- Selectors -->
        <div class="bg-base-800 rounded-lg p-4">
            <h3 class="text-sm font-medium text-primary-400 mb-3">
                Content Selectors
            </h3>

            {#if formData.selectors && formData.selectors.length > 0}
                <div class="overflow-x-auto">
                    <table class="min-w-full divide-y divide-dark-700">
                        <thead>
                            <tr>
                                <th
                                    class="px-3 py-2 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                    >Name</th
                                >
                                <th
                                    class="px-3 py-2 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                    >Purpose</th
                                >
                                <th
                                    class="px-3 py-2 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                    >Type</th
                                >
                                <th
                                    class="px-3 py-2 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                    >Value</th
                                >
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-dark-700">
                            {#each formData.selectors as selector}
                                <tr class="hover:bg-base-750">
                                    <td
                                        class="px-3 py-2 whitespace-nowrap text-sm"
                                        >{selector.name}</td
                                    >
                                    <td
                                        class="px-3 py-2 whitespace-nowrap text-sm"
                                    >
                                        <span
                                            class={`px-2 py-0.5 inline-flex text-xs leading-5 font-semibold rounded-full 
                        ${
                            selector.purpose === "assets"
                                ? "bg-blue-500 text-blue-100"
                                : selector.purpose === "links"
                                  ? "bg-green-500 text-green-100"
                                  : selector.purpose === "pagination"
                                    ? "bg-yellow-500 text-yellow-100"
                                    : "bg-purple-500 text-purple-100"
                        }`}
                                        >
                                            {selector.purpose}
                                        </span>
                                    </td>
                                    <td
                                        class="px-3 py-2 whitespace-nowrap text-sm"
                                        >{selector.type}</td
                                    >
                                    <td
                                        class="px-3 py-2 text-sm font-mono text-dark-300 truncate max-w-xs"
                                    >
                                        {selector.value}
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            {:else}
                <p class="text-sm text-dark-400">No selectors configured</p>
            {/if}
        </div>

        <!-- Filtering & Limits -->
        <div class="bg-base-800 rounded-lg p-4">
            <h3 class="text-sm font-medium text-primary-400 mb-3">
                Filtering & Limits
            </h3>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                {#if formData.rules}
                    <div>
                        <p class="text-xs text-dark-400">Maximum Depth</p>
                        <p class="text-sm">
                            {formData.rules.maxDepth || "0"}
                            {parseInt(formData.rules.maxDepth) === 0
                                ? "(unlimited)"
                                : ""}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">Maximum Assets</p>
                        <p class="text-sm">
                            {formData.rules.maxAssets || "0"}
                            {parseInt(formData.rules.maxAssets) === 0
                                ? "(unlimited)"
                                : ""}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">Maximum Pages</p>
                        <p class="text-sm">
                            {formData.rules.maxPages || "0"}
                            {parseInt(formData.rules.maxPages) === 0
                                ? "(unlimited)"
                                : ""}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">
                            Concurrent Connections
                        </p>
                        <p class="text-sm">
                            {formData.rules.maxConcurrent || "5"}
                        </p>
                    </div>

                    {#if formData.rules.includeUrlPattern}
                        <div>
                            <p class="text-xs text-dark-400">
                                Include URLs (regex)
                            </p>
                            <p class="text-sm font-mono">
                                {formData.rules.includeUrlPattern}
                            </p>
                        </div>
                    {/if}

                    {#if formData.rules.excludeUrlPattern}
                        <div>
                            <p class="text-xs text-dark-400">
                                Exclude URLs (regex)
                            </p>
                            <p class="text-sm font-mono">
                                {formData.rules.excludeUrlPattern}
                            </p>
                        </div>
                    {/if}
                {:else}
                    <div class="md:col-span-2">
                        <p class="text-sm text-dark-400">
                            No filtering rules configured
                        </p>
                    </div>
                {/if}
            </div>

            {#if formData.filters && formData.filters.length > 0}
                <div class="mt-4">
                    <p class="text-xs text-dark-400 mb-2">Custom Filters</p>
                    <ul class="space-y-1 text-sm">
                        {#each formData.filters as filter}
                            <li class="flex items-center">
                                <span
                                    class={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium 
                    ${filter.action === "include" ? "bg-green-500 text-green-100" : "bg-red-500 text-red-100"} mr-2`}
                                >
                                    {filter.action}
                                </span>
                                <span class="text-white">{filter.name}:</span>
                                <span class="text-dark-300 ml-1 font-mono"
                                    >{filter.pattern}</span
                                >
                            </li>
                        {/each}
                    </ul>
                </div>
            {/if}
        </div>

        <!-- Processing Options -->
        {#if formData.processing}
            <div class="bg-base-800 rounded-lg p-4">
                <h3 class="text-sm font-medium text-primary-400 mb-3">
                    Processing Options
                </h3>

                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div>
                        <p class="text-xs text-dark-400">Generate Thumbnails</p>
                        <p class="text-sm">
                            {formData.processing.thumbnails ? "Yes" : "No"}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">Extract Metadata</p>
                        <p class="text-sm">
                            {formData.processing.metadata ? "Yes" : "No"}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">
                            Enable Deduplication
                        </p>
                        <p class="text-sm">
                            {formData.processing.deduplication ? "Yes" : "No"}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">Resize Images</p>
                        <p class="text-sm">
                            {#if formData.processing.imageResize}
                                Yes (max width: {formData.processing
                                    .imageWidth}px)
                            {:else}
                                No
                            {/if}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">Convert Videos</p>
                        <p class="text-sm">
                            {#if formData.processing.videoConvert}
                                Yes (to {formData.processing.videoFormat.toUpperCase()})
                            {:else}
                                No
                            {/if}
                        </p>
                    </div>

                    <div>
                        <p class="text-xs text-dark-400">
                            Extract Text from Documents
                        </p>
                        <p class="text-sm">
                            {formData.processing.extractText ? "Yes" : "No"}
                        </p>
                    </div>
                </div>
            </div>
        {/if}

        <!-- Schedule -->
        <div class="bg-base-800 rounded-lg p-4">
            <h3 class="text-sm font-medium text-primary-400 mb-3">Schedule</h3>

            {#if formData.schedule}
                <div>
                    <p class="text-xs text-dark-400">Cron Schedule</p>
                    <p class="text-sm font-mono">{formData.schedule}</p>
                    <p class="text-xs text-dark-300 mt-1">
                        {getCronDescription(formData.schedule)}
                    </p>
                </div>
            {:else}
                <p class="text-sm text-dark-400">
                    Manual execution only (no schedule)
                </p>
            {/if}
        </div>
    </div>

    <div class="mt-6 bg-base-850 rounded-lg p-4">
        <div class="flex">
            <div class="flex-shrink-0">
                <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="h-6 w-6 text-primary-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                </svg>
            </div>
            <div class="ml-3">
                <p class="text-sm text-dark-300">
                    Review your job configuration carefully. Once created, the
                    job will be ready to run but won't start automatically
                    unless scheduled.
                </p>
            </div>
        </div>
    </div>
</div>
