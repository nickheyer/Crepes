<script>
    import { onMount, onDestroy } from "svelte";
    import { isValidUrl } from "$lib/utils/validation";
    import { addToast } from "$lib/stores/uiStore.svelte";
    
    // PROPS
    let {
        url = '',
        selectedElement = null,
        selectedElements = []
    } = $props();
    
    // INTERNAL STATE
    let iframeLoaded = $state(false);
    let iframeError = $state(false);
    let loading = $state(true);
    let iframe;
    let cssPath = $state("");
    
    // SELECTION TYPES 
    const selectionTypes = [
        { id: "assets", label: "Assets (images, videos)" },
        { id: "links", label: "Links (URLs to follow)" },
        { id: "pagination", label: "Pagination" },
        { id: "metadata", label: "Metadata" }
    ];
    let selectedType = $state("assets");
    
    onMount(() => {
        // AUTO-LOAD URL IF PROVIDED
        if (url) {
            loadIframe();
        }
    });
    
    function loadIframe() {
        if (!isValidUrl(url)) {
            loading = false;
            iframeError = true;
            addToast("PLEASE ENTER A VALID URL", "error");
            return;
        }
        
        loading = true;
        iframeError = false;
        iframeLoaded = false;
        
        if (iframe) {
            // USE BACKEND PROXY TO BYPASS CORS
            iframe.src = `/api/proxy?url=${encodeURIComponent(url)}`;
            
            iframe.onload = () => {
                setupIframeInteraction();
                loading = false;
                iframeLoaded = true;
                
                // ENSURE IFRAME CONTENTS ARE PROPERLY SCALED
                scaleIframeContents();
            };
            
            iframe.onerror = () => {
                loading = false;
                iframeError = true;
                addToast("FAILED TO LOAD THE WEBPAGE", "error");
            };
        }
    }
    
    // SCALE IFRAME CONTENTS TO FIT THE CONTAINER
    function scaleIframeContents() {
        try {
            const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
            const style = document.createElement('style');
            style.textContent = `
                html, body {
                    width: 100% !important;
                    height: 100% !important;
                    margin: 0 !important;
                    padding: 0 !important;
                    overflow-x: hidden !important;
                }
                
                body {
                    transform-origin: 0 0;
                    transform: scale(1);
                }
            `;
            iframeDoc.head.appendChild(style);
        } catch (e) {
            console.error("ERROR SCALING IFRAME:", e);
        }
    }
    
    function setupIframeInteraction() {
        try {
            const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
            
            // ADD CSS FOR HIGHLIGHTING
            const style = document.createElement('style');
            style.textContent = `
                .selector-hover {
                    outline: 2px solid #ff72c0 !important;
                    background-color: rgba(255, 114, 192, 0.1) !important;
                    cursor: pointer !important;
                    position: relative !important;
                    z-index: 9999 !important;
                }
                
                .selector-selected {
                    outline: 2px solid #42b983 !important;
                    background-color: rgba(66, 185, 131, 0.1) !important;
                    position: relative !important;
                    z-index: 9998 !important;
                }
            `;
            iframeDoc.head.appendChild(style);
            
            // DISABLE ALL LINKS AND HANDLE ELEMENT SELECTION
            iframeDoc.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                
                if (e.target) {
                    // CREATE ELEMENT DATA
                    const elementData = createElementData(e.target);
                    
                    // TOGGLE SELECTION VISUALLY
                    e.target.classList.toggle('selector-selected');
                    
                    if (e.target.classList.contains('selector-selected')) {
                        // ADD TO SELECTED ELEMENTS
                        selectedElements = [...selectedElements, elementData];
                        // UPDATE THE CURRENTLY SELECTED ELEMENT
                        selectedElement = elementData;
                    } else {
                        // REMOVE FROM SELECTED ELEMENTS
                        selectedElements = selectedElements.filter(el => el.cssPath !== elementData.cssPath);
                        // CLEAR SELECTED ELEMENT IF IT WAS THIS ONE
                        if (selectedElement && selectedElement.cssPath === elementData.cssPath) {
                            selectedElement = null;
                        }
                    }
                }
                return false;
            }, true);
            
            // HANDLE HOVER
            iframeDoc.addEventListener('mouseover', (e) => {
                if (e.target) {
                    e.target.classList.add('selector-hover');
                    cssPath = generateCssSelector(e.target);
                }
            }, true);
            
            iframeDoc.addEventListener('mouseout', (e) => {
                if (e.target) {
                    e.target.classList.remove('selector-hover');
                }
            }, true);
        } catch (e) {
            console.error("ERROR SETTING UP IFRAME:", e);
            loading = false;
            iframeError = true;
            addToast("CANNOT ACCESS PAGE CONTENT", "error");
        }
    }
    
    // CREATE ELEMENT DATA FROM DOM ELEMENT
    function createElementData(element) {
        return {
            tag: element.tagName.toLowerCase(),
            cssPath: generateCssSelector(element),
            attribute: getDefaultAttribute(element.tagName.toLowerCase(), selectedType),
            purpose: selectedType,
            text: element.textContent?.trim().substring(0, 50) || "",
            html: element.outerHTML?.substring(0, 100) || ""
        };
    }
    
    // GENERATE CSS SELECTOR FOR ELEMENT
    function generateCssSelector(element) {
        if (!element) return '';
        if (element.tagName.toLowerCase() === 'html') return 'html';
        if (element.tagName.toLowerCase() === 'body') return 'body';
        
        let path = [];
        let current = element;
        
        while (current && current.nodeType === Node.ELEMENT_NODE) {
            let selector = current.tagName.toLowerCase();
            
            // USE ID IF AVAILABLE
            if (current.id) {
                selector += '#' + current.id;
                path.unshift(selector);
                break;
            }
            
            // USE CLASSES
            const classes = Array.from(current.classList)
                .filter(c => !c.startsWith('selector-'))
                .join('.');
                
            if (classes) {
                selector += '.' + classes;
            }
            
            // IF THERE ARE SIBLINGS
            let siblingCount = 0;
            let siblingIndex = 0;
            let sibling = current;
            
            while (sibling) {
                if (sibling.tagName === current.tagName) {
                    siblingCount++;
                    if (sibling === current) {
                        siblingIndex = siblingCount;
                    }
                }
                sibling = sibling.previousElementSibling;
            }
            
            if (siblingCount > 1) {
                selector += ':nth-of-type(' + siblingIndex + ')';
            }
            
            path.unshift(selector);
            current = current.parentElement;
        }
        
        return path.join(' > ');
    }
    
    // GET DEFAULT ATTRIBUTE BASED ON ELEMENT TYPE AND PURPOSE
    function getDefaultAttribute(tagName, purpose) {
        if (purpose === 'assets') {
            if (tagName === 'img') return 'src';
            if (tagName === 'video') return 'src';
            if (tagName === 'audio') return 'src';
            return 'src';
        } else if (purpose === 'links' || purpose === 'pagination') {
            return 'href';
        } else if (purpose === 'metadata') {
            return 'text';
        }
        return 'src';
    }
    
    // CLEAR ALL SELECTIONS
    function clearSelections() {
        if (iframeLoaded && iframe?.contentDocument) {
            const elements = iframe.contentDocument.querySelectorAll('.selector-selected');
            elements.forEach(el => el.classList.remove('selector-selected'));
        }
        selectedElements = [];
        selectedElement = null;
    }
    
    // UPDATE PURPOSE FOR ALL SELECTED ELEMENTS
    function updateSelectionPurpose(newPurpose) {
        selectedType = newPurpose;
        
        if (selectedElements.length > 0) {
            // UPDATE ALL SELECTED ELEMENTS TO NEW TYPE
            selectedElements = selectedElements.map(el => ({
                ...el,
                purpose: newPurpose,
                attribute: getDefaultAttribute(el.tag, newPurpose)
            }));
        }
    }
    
    // REMOVE ELEMENT FROM SELECTION
    function removeElement(elementToRemove) {
        if (iframeLoaded && iframe?.contentDocument) {
            try {
                const element = iframe.contentDocument.querySelector(elementToRemove.cssPath);
                if (element) {
                    element.classList.remove('selector-selected');
                }
            } catch (e) {
                // IGNORE SELECTOR ERRORS
            }
        }
        
        selectedElements = selectedElements.filter(el => el.cssPath !== elementToRemove.cssPath);
        
        if (selectedElement && selectedElement.cssPath === elementToRemove.cssPath) {
            selectedElement = null;
        }
    }
    
    // AUTO-LOAD IFRAME WHEN URL CHANGES
    $effect(() => {
        if (url && !iframeLoaded) {
            loadIframe();
        }
    });
</script>

<div class="flex flex-col h-full space-y-4">
    <!-- BROWSER UI -->
    <div class="mockup-browser border border-base-300 w-full">
        <div class="mockup-browser-toolbar">
            <div class="input flex-1 join">
                <input
                    type="url"
                    bind:value={url}
                    placeholder="https://example.com"
                    class="join-item input input-bordered w-full"
                />
                <button class="btn btn-primary join-item" onclick={loadIframe}>
                    Load
                </button>
            </div>
        </div>
        
        <!-- SELECTOR CONTROLS -->
        <div class="bg-base-200 p-2 border-t border-base-300 flex flex-wrap justify-between items-center gap-2">
            <select 
                bind:value={selectedType}
                onchange={() => updateSelectionPurpose(selectedType)}
                class="select select-sm select-bordered"
            >
                {#each selectionTypes as type}
                    <option value={type.id}>{type.label}</option>
                {/each}
            </select>
            
            <button class="btn btn-sm btn-outline" onclick={clearSelections}>
                Clear
            </button>
            
            {#if cssPath}
                <div class="w-full mt-1 px-2 py-1 bg-base-300 rounded text-xs font-mono overflow-x-auto whitespace-nowrap">
                    {cssPath}
                </div>
            {/if}
        </div>
        
        <!-- IFRAME CONTAINER WITH FIXED HEIGHT -->
        <div 
            class="mockup-browser-content bg-base-200 relative overflow-hidden"
            style="height: 400px"
        >
            {#if loading}
                <div class="absolute inset-0 flex items-center justify-center bg-base-100 bg-opacity-75 z-10">
                    <span class="loading loading-spinner loading-lg text-primary"></span>
                </div>
            {/if}
            
            <iframe
                bind:this={iframe}
                title="Web Page Preview"
                class="w-full h-full border-0"
                sandbox="allow-same-origin allow-scripts"
                style="background: white; max-height: 400px; overflow-y: auto;"
            ></iframe>
            
            {#if iframeError && !loading}
                <div class="absolute inset-0 flex flex-col items-center justify-center bg-base-100">
                    <h3 class="text-lg font-medium mb-2">Unable to load page content</h3>
                    <p class="text-center max-w-md mb-4">
                        The page content could not be loaded. Please check the URL and try again.
                    </p>
                </div>
            {/if}
        </div>
    </div>
    
    <!-- SELECTED ELEMENTS TABLE -->
    {#if selectedElements.length > 0}
        <div class="bg-base-200 rounded-lg p-4">
            <h3 class="font-medium mb-3">Selected Elements ({selectedElements.length})</h3>
            <div class="overflow-x-auto">
                <table class="table table-compact w-full">
                    <thead>
                        <tr>
                            <th>Element</th>
                            <th>Selector</th>
                            <th>Purpose</th>
                            <th>Attribute</th>
                            <th>Action</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each selectedElements as element}
                            <tr class={element === selectedElement ? "bg-base-300" : ""}>
                                <td class="font-mono text-xs">{element.tag}</td>
                                <td class="font-mono text-xs truncate max-w-xs">{element.cssPath}</td>
                                <td>
                                    <select 
                                        class="select select-xs select-bordered w-full max-w-xs"
                                        bind:value={element.purpose}
                                        onchange={() => {
                                            element.attribute = getDefaultAttribute(element.tag, element.purpose);
                                        }}
                                    >
                                        {#each selectionTypes as type}
                                            <option value={type.id}>{type.label}</option>
                                        {/each}
                                    </select>
                                </td>
                                <td>
                                    <select 
                                        class="select select-xs select-bordered w-full max-w-xs"
                                        bind:value={element.attribute}
                                    >
                                        <option value="src">src</option>
                                        <option value="href">href</option>
                                        <option value="text">text content</option>
                                        <option value="html">HTML content</option>
                                        <option value="data-src">data-src</option>
                                        <option value="alt">alt</option>
                                        <option value="title">title</option>
                                    </select>
                                </td>
                                <td>
                                    <button 
                                        class="btn btn-xs btn-error"
                                        onclick={() => removeElement(element)}
                                    >
                                        Remove
                                    </button>
                                </td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        </div>
    {/if}
</div>
