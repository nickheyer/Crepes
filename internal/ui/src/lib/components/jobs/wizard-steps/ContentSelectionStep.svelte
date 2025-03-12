<script>
    import { onMount } from "svelte";
    import VisualSelector from "../VisualSelector.svelte";
    import { state as jobState, setStepValidity } from "$lib/stores/jobStore.svelte";
    
    let view = $state("list");
    let newSelector = $state({
        id: "",
        name: "",
        type: "css",
        value: "",
        attribute: "src",
        description: "",
        purpose: "assets",
        priority: 0,
        isOptional: false,
        urlPattern: "",
    });
    let editingIndex = $state(-1);
    let selectedVisualElement = $state(null);
    let selectedVisualElements = $state([]);
    let isValid = $state(false);
    
    onMount(() => {
        resetNewSelector();
    });
    
    $effect(() => {
        if (selectedVisualElement) {
            // POPULATE FORM WITH SELECTED ELEMENT DATA
            newSelector = {
                id: generateId(),
                name: `${selectedVisualElement.purpose} - ${selectedVisualElement.tag}`,
                type: "css",
                value: selectedVisualElement.cssPath,
                attribute: selectedVisualElement.attribute,
                purpose: selectedVisualElement.purpose,
                description: selectedVisualElement.text ? `Extracts: ${selectedVisualElement.text}` : "",
                priority: 0,
                isOptional: false,
                urlPattern: ""
            };
            
            // SWITCH TO LIST VIEW TO ALLOW EDITING
            view = "list";
        }
    });
    
    function getDefaultAttributeForPurpose(purpose) {
        switch(purpose) {
            case "links":
            case "pagination":
                return "href";
            case "assets":
                return "src";
            case "metadata":
                return "text";
            default:
                return "src";
        }
    }
    
    function handlePurposeChange() {
        newSelector.attribute = getDefaultAttributeForPurpose(newSelector.purpose);
    }
    
    function addSelector() {
        if (!newSelector.name || !newSelector.value) return;
        
        // GENERATE NEW ID IF ADDING
        if (editingIndex < 0) {
            newSelector.id = generateId();
        }
        
        if (editingIndex >= 0) {
            // UPDATE EXISTING SELECTOR
            jobState.formData.data.selectors[editingIndex] = { ...newSelector };
            editingIndex = -1;
        } else {
            // ADD NEW SELECTOR
            jobState.formData.data.selectors = [...jobState.formData.data.selectors, { ...newSelector }];
        }
        
        // RESET FORM
        resetNewSelector();
        validate();
    }
    
    function editSelector(index) {
        newSelector = { ...jobState.formData.data.selectors[index] };
        editingIndex = index;
        
        // SWITCH TO FORM VIEW IF IN VISUAL MODE
        if (view === "visual") {
            view = "list";
        }
    }
    
    function removeSelector(index) {
        jobState.formData.data.selectors = jobState.formData.data.selectors.filter((_, i) => i !== index);
        
        if (editingIndex === index) {
            resetNewSelector();
            editingIndex = -1;
        }
        
        validate();
    }
    
    function resetNewSelector() {
        newSelector = {
            id: generateId(),
            name: "",
            type: "css",
            value: "",
            attribute: "src",
            description: "",
            purpose: "assets",
            priority: 0,
            isOptional: false,
            urlPattern: "",
        };
        editingIndex = -1;
    }
    
    function generateId() {
        return "sel_" + Math.random().toString(36).substring(2, 11);
    }
    
    function switchView(newView) {
        view = newView;
    }
    
    function validate() {
        const hasLinks = jobState.formData.data.selectors.some((sel) => sel.purpose === "links");
        const hasAssets = jobState.formData.data.selectors.some((sel) => sel.purpose === "assets");
        isValid = jobState.formData.data.selectors.length > 0 && hasLinks && hasAssets;
    
        setStepValidity(2, isValid);
        return isValid;
    }
</script>

<div>
    <h2 class="text-xl font-semibold mb-4">Content Selection</h2>
    <p class="text-dark-300 mb-6">
        Define what content to extract with CSS or XPath selectors
    </p>
    
    <!-- VIEW SWITCHER -->
    <div class="flex border border-dark-700 rounded-lg mb-6 overflow-hidden">
        <button
            class={`flex-1 py-3 px-4 focus:outline-none ${view === "list" ? "bg-base-700 text-white" : "bg-base-800 text-dark-300 hover:bg-base-750"}`}
            onclick={() => switchView("list")}
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 inline-block mr-2"
                viewBox="0 0 20 20"
                fill="currentColor"
            >
                <path
                    fill-rule="evenodd"
                    d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
                    clip-rule="evenodd"
                />
            </svg>
            List View
        </button>
        <button
            class={`flex-1 py-3 px-4 focus:outline-none ${view === "visual" ? "bg-base-700 text-white" : "bg-base-800 text-dark-300 hover:bg-base-750"}`}
            onclick={() => switchView("visual")}
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-5 w-5 inline-block mr-2"
                viewBox="0 0 20 20"
                fill="currentColor"
            >
                <path
                    fill-rule="evenodd"
                    d="M3 5a2 2 0 012-2h10a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2V5zm11 1H6a1 1 0 00-1 1v6a1 1 0 001 1h8a1 1 0 001-1V7a1 1 0 00-1-1z"
                    clip-rule="evenodd"
                />
            </svg>
            Visual Selector
        </button>
    </div>

    {#if view === "list"}
        <!-- SELECTOR LIST VIEW -->
        <div>
            <!-- EXISTING SELECTORS -->
            {#if jobState.formData.data.selectors && jobState.formData.data.selectors.length > 0}
                <div class="mb-6">
                    <h3 class="text-sm font-medium text-dark-300 mb-3">
                        Current Selectors
                    </h3>
                    <div class="bg-base-800 rounded-lg overflow-hidden">
                        <table class="min-w-full divide-y divide-dark-700">
                            <thead class="bg-base-750">
                                <tr>
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Name</th
                                    >
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Purpose</th
                                    >
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-left text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Selector</th
                                    >
                                    <th
                                        scope="col"
                                        class="px-4 py-3 text-right text-xs font-medium text-dark-300 uppercase tracking-wider"
                                        >Actions</th
                                    >
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-dark-700">
                                {#each jobState.formData.data.selectors as selector, i}
                                    <tr class="hover:bg-base-750">
                                        <td class="px-4 py-3 whitespace-nowrap">
                                            <div class="text-sm font-medium">
                                                {selector.name}
                                            </div>
                                            {#if selector.description}
                                                <div
                                                    class="text-xs text-dark-400"
                                                >
                                                    {selector.description}
                                                </div>
                                            {/if}
                                        </td>
                                        <td class="px-4 py-3 whitespace-nowrap">
                                            <span
                                                class={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full 
                                                ${selector.purpose === "assets" ? "bg-blue-500 text-blue-100" : 
                                                  selector.purpose === "links" ? "bg-green-500 text-green-100" : 
                                                  selector.purpose === "pagination" ? "bg-yellow-500 text-yellow-100" : 
                                                  "bg-purple-500 text-purple-100"}`}
                                            >
                                                {selector.purpose}
                                            </span>
                                        </td>
                                        <td class="px-4 py-3">
                                            <div
                                                class="text-xs font-mono text-dark-300 break-all"
                                            >
                                                {selector.value}
                                            </div>
                                        </td>
                                        <td
                                            class="px-4 py-3 whitespace-nowrap text-right text-sm font-medium"
                                        >
                                            <button
                                                class="text-primary-400 hover:text-primary-300 mr-3"
                                                onclick={() => editSelector(i)}
                                            >
                                                Edit
                                            </button>
                                            <button
                                                class="text-danger-400 hover:text-danger-300"
                                                onclick={() => removeSelector(i)}
                                            >
                                                Delete
                                            </button>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                </div>
            {/if}

            <!-- SELECTOR FORM -->
            <div class="bg-base-800 rounded-lg p-4 mb-6">
                <h3 class="text-sm font-medium mb-4">
                    {editingIndex >= 0 ? "Edit Selector" : "Add New Selector"}
                </h3>
                <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                        <label
                            for="selector-name"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Name <span class="text-danger-500">*</span>
                        </label>
                        <input
                            id="selector-name"
                            type="text"
                            bind:value={newSelector.name}
                            placeholder="E.g., Product Images"
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        />
                    </div>

                    <div>
                        <label
                            for="selector-purpose"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Purpose <span class="text-danger-500">*</span>
                        </label>
                        <select
                            id="selector-purpose"
                            bind:value={newSelector.purpose}
                            onchange={handlePurposeChange}
                            class="select select-bordered w-full"
                        >
                            <option value="assets">Assets (images, videos)</option>
                            <option value="links">Links (URLs to follow)</option>
                            <option value="pagination">Pagination</option>
                            <option value="metadata">Metadata</option>
                        </select>
                    </div>

                    <div>
                        <label
                            for="selector-type"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Selector Type
                        </label>
                        <select
                            id="selector-type"
                            bind:value={newSelector.type}
                            class="select select-bordered w-full"
                        >
                            <option value="css">CSS Selector</option>
                            <option value="xpath">XPath</option>
                        </select>
                    </div>

                    <div>
                        <label
                            for="selector-attribute"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Attribute to Extract
                        </label>
                        <select
                            id="selector-attribute"
                            bind:value={newSelector.attribute}
                            class="select select-bordered w-full"
                        >
                            <option value="src">src (for images, videos)</option>
                            <option value="href">href (for links)</option>
                            <option value="text">text content</option>
                            <option value="html">HTML content</option>
                            <option value="data-src">data-src (lazy loading)</option>
                            <option value="alt">alt (image description)</option>
                            <option value="title">title attribute</option>
                        </select>
                    </div>

                    <div class="sm:col-span-2">
                        <label
                            for="selector-value"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Selector Value <span class="text-danger-500">*</span>
                        </label>
                        <input
                            id="selector-value"
                            type="text"
                            bind:value={newSelector.value}
                            placeholder="E.g., .product-image img"
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        />
                    </div>

                    <div class="sm:col-span-2">
                        <label
                            for="selector-description"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Description
                        </label>
                        <textarea
                            id="selector-description"
                            bind:value={newSelector.description}
                            rows="2"
                            placeholder="Describe what this selector does"
                            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                        ></textarea>
                    </div>

                    <div class="sm:col-span-2 flex items-center">
                        <input
                            id="selector-optional"
                            type="checkbox"
                            bind:checked={newSelector.isOptional}
                            class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-dark-500 rounded"
                        />
                        <label
                            for="selector-optional"
                            class="ml-2 block text-sm text-dark-300"
                        >
                            Optional (continue even if selector doesn't match)
                        </label>
                    </div>
                </div>

                <div class="mt-4 flex justify-end space-x-3">
                    {#if editingIndex >= 0}
                        <button
                            class="px-3 py-1.5 text-sm border border-dark-600 rounded-md focus:outline-none hover:bg-base-700"
                            onclick={resetNewSelector}
                        >
                            Cancel
                        </button>
                    {/if}
                    <button
                        class="px-3 py-1.5 text-sm bg-primary-600 text-white rounded-md focus:outline-none hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        onclick={addSelector}
                        disabled={!newSelector.name || !newSelector.value}
                    >
                        {editingIndex >= 0 ? "Update Selector" : "Add Selector"}
                    </button>
                </div>
            </div>

            <!-- HELP SECTION -->
            <div class="bg-base-850 rounded-lg p-4">
                <h4 class="text-sm font-medium mb-2">Selector Tips</h4>
                <ul class="text-xs text-dark-300 list-disc pl-5 space-y-1">
                    <li>
                        You need at least one <strong>links</strong> selector
                        (to find URLs to crawl) and one <strong>assets</strong> selector
                        (to find content to download)
                    </li>
                    <li>
                        Try switching to the <strong>Visual Selector</strong> tab to easily pick elements from the page
                    </li>
                    <li>
                        For images, use <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >img</code
                        >
                        with
                        <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >src</code
                        > attribute
                    </li>
                    <li>
                        For links, use <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >a</code
                        >
                        with
                        <code
                            class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                            >href</code
                        > attribute
                    </li>
                </ul>
            </div>
        </div>
    {:else}
        <!-- VISUAL SELECTOR VIEW -->
        <div class="bg-base-800 rounded-lg overflow-hidden">
            <VisualSelector
                url={jobState.formData.data.baseUrl}
                bind:selectedElement={selectedVisualElement}
                bind:selectedElements={selectedVisualElements}
            />
        </div>
    {/if}
</div>
