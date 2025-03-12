<script>
    import { onMount } from "svelte";
    import { isValidUrl } from "$lib/utils/validation";
    import { state as jobState } from "$lib/stores/jobStore.svelte";
    import { addToast } from "$lib/stores/uiStore.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import { Eye, MousePointer, RefreshCw, XCircle } from "lucide-svelte";

    let iframeLoaded = $state(false);
    let iframeError = $state(false);
    let loading = $state(true);
    let iframe;
    let currentSelector = $state("");
    let inspectMode = $state(true);

    const selectionTypes = [
        { id: "assets", label: "Assets (images, videos)" },
        { id: "links", label: "Links (URLs to follow)" },
        { id: "pagination", label: "Pagination" },
        { id: "metadata", label: "Metadata" },
    ];
    let selectedType = $state(
        jobState.formData.data.visualSelectionType || "assets",
    );

    onMount(() => {
        if (jobState.formData.data.baseUrl) {
            loadIframe();
        }
    });

    function loadIframe() {
        if (!isValidUrl(jobState.formData.data.baseUrl)) {
            loading = false;
            iframeError = true;
            addToast("please enter a valid URL", "error");
            return;
        }

        loading = true;
        iframeError = false;
        iframeLoaded = false;

        if (iframe) {
            // USE BACKEND PROXY TO BYPASS CORS
            iframe.src = `/api/proxy?url=${encodeURIComponent(jobState.formData.data.baseUrl)}`;

            iframe.onload = () => {
                setupIframeInteraction();
                loading = false;
                iframeLoaded = true;
            };

            iframe.onerror = () => {
                loading = false;
                iframeError = true;
                addToast("failed to load the webpage", "error");
            };
        }
    }

    function setupIframeInteraction() {
        try {
            const doc = iframe.contentDocument || iframe.contentWindow.document;

            // ADD CSS FOR HIGHLIGHTING
            const style = document.createElement("style");
            style.textContent = `
                .__vs_hover {
                    outline: 2px solid #ff72c0 !important;
                    background-color: rgba(255, 114, 192, 0.1) !important;
                    cursor: pointer !important;
                    position: relative !important;
                    z-index: 9999 !important;
                }
                .__vs_selected {
                    outline: 2px solid #42b983 !important;
                    background-color: rgba(66, 185, 131, 0.1) !important;
                    position: relative !important;
                    z-index: 9998 !important;
                }
            `;
            doc.head.appendChild(style);

            // ADD EVENT LISTENERS
            doc.addEventListener("mouseover", handleMouseOver, true);
            doc.addEventListener("mouseout", handleMouseOut, true);
            doc.addEventListener("click", handleClick, true);

            // DISABLE ALL LINK CLICKS
            const allLinks = doc.querySelectorAll("a");
            allLinks.forEach((link) => {
                link.addEventListener("click", (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    return false;
                });
            });

            // HIGHLIGHT PREVIOUSLY SELECTED ELEMENTS
            if (jobState.formData.data.visualSelections?.length) {
                jobState.formData.data.visualSelections.forEach((sel) => {
                    try {
                        const el = doc.querySelector(sel.cssPath);
                        if (el) {
                            el.classList.add("__vs_selected");
                        }
                    } catch (e) {
                        // IGNORE SELECTOR ERRORS
                    }
                });
            }

            // SCALE CONTENT TO FIT
            scaleIframeContents();
        } catch (e) {
            console.error("error setting up iframe:", e);
            loading = false;
            iframeError = true;
            addToast("cannot access page content", "error");
        }
    }

    function scaleIframeContents() {
        try {
            const doc = iframe.contentDocument || iframe.contentWindow.document;
            const style = document.createElement("style");
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
            doc.head.appendChild(style);
        } catch (e) {
            console.error("error scaling iframe:", e);
        }
    }

    function handleMouseOver(e) {
        if (!inspectMode) return;

        if (e.target && e.target.nodeType === 1) {
            // DON'T HIGHLIGHT OUR OWN ELEMENTS
            if (
                e.target.classList.contains("__vs_hover") ||
                e.target.classList.contains("__vs_selected")
            ) {
                return;
            }

            // HIGHLIGHT ELEMENT
            e.target.classList.add("__vs_hover");

            // GENERATE SELECTOR AND UPDATE UI
            currentSelector = generateCssPath(e.target);
        }
    }

    function handleMouseOut(e) {
        if (!inspectMode) return;

        if (e.target && e.target.nodeType === 1) {
            e.target.classList.remove("__vs_hover");
        }
    }

    function handleClick(e) {
        if (!inspectMode) return;

        e.preventDefault();
        e.stopPropagation();

        if (e.target && e.target.nodeType === 1) {
            // HANDLE SELECTION
            const element = e.target;
            const elementData = createElementData(element);

            // TOGGLE SELECTION VISUALLY
            element.classList.toggle("__vs_selected");

            // UPDATE STORE DIRECTLY
            if (element.classList.contains("__vs_selected")) {
                addVisualSelection(elementData);
            } else {
                removeVisualSelection(elementData);
            }
        }

        return false;
    }

    function addVisualSelection(elementData) {
        // ENSURE VISUALSELECTIONS EXISTS
        const selections = jobState.formData.data.visualSelections || [];
        jobState.formData.data.visualSelections = [...selections, elementData];
        jobState.formData.data.visualSelectionType = selectedType;

        // ADD CORRESPONDING SELECTOR TO JOB DATA
        addSelectorFromVisual(elementData);
    }

    function removeVisualSelection(elementData) {
        // REMOVE FROM VISUALSELECTIONS
        if (jobState.formData.data.visualSelections) {
            jobState.formData.data.visualSelections =
                jobState.formData.data.visualSelections.filter(
                    (sel) => sel.cssPath !== elementData.cssPath,
                );
        }

        // REMOVE CORRESPONDING SELECTOR
        removeMatchingSelector(elementData);
    }

    function addSelectorFromVisual(elementData) {
        // CREATE JOB SELECTOR FROM VISUAL SELECTION
        const selector = {
            id: "sel_" + Math.random().toString(36).substring(2, 11),
            name: `${elementData.purpose} - ${elementData.tag}`,
            type: "css",
            value: elementData.cssPath,
            attribute: elementData.attribute,
            purpose: elementData.purpose,
            description: elementData.text
                ? `Extracts: ${elementData.text}`
                : "",
            priority: 0,
            isOptional: false,
            urlPattern: "",
        };

        // ADD TO SELECTORS ARRAY
        const selectors = jobState.formData.data.selectors || [];
        jobState.formData.data.selectors = [...selectors, selector];
    }

    function removeMatchingSelector(elementData) {
        // FIND AND REMOVE MATCHING SELECTOR
        if (jobState.formData.data.selectors) {
            jobState.formData.data.selectors =
                jobState.formData.data.selectors.filter(
                    (sel) => sel.value !== elementData.cssPath,
                );
        }
    }

    function createElementData(element) {
        // GET TEXT CONTENT TRUNCATED
        let text = element.textContent?.trim() || "";
        if (text.length > 50) {
            text = text.substring(0, 47) + "...";
        }

        return {
            tag: element.tagName.toLowerCase(),
            cssPath: generateCssPath(element),
            xPath: generateXPath(element),
            attribute: getDefaultAttribute(
                element.tagName.toLowerCase(),
                selectedType,
            ),
            purpose: selectedType,
            text: text,
            html: element.outerHTML?.substring(0, 100) || "",
        };
    }

    function generateCssPath(element) {
        if (!element || element.nodeType !== 1) return "";
        if (element.tagName.toLowerCase() === "html") return "html";
        if (element.tagName.toLowerCase() === "body") return "body";

        let path = [];
        let current = element;

        while (current && current.nodeType === Node.ELEMENT_NODE) {
            let selector = current.tagName.toLowerCase();

            // USE ID IF AVAILABLE
            if (current.id) {
                selector += "#" + CSS.escape(current.id);
                path.unshift(selector);
                break;
            }

            // USE CLASSES
            const classes = Array.from(current.classList)
                .filter((c) => !c.startsWith("__vs_"))
                .map((c) => CSS.escape(c))
                .join(".");

            if (classes) {
                selector += "." + classes;
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
                selector += `:nth-of-type(${siblingIndex})`;
            }

            path.unshift(selector);
            current = current.parentElement;
        }

        return path.join(" > ");
    }

    function generateXPath(element) {
        if (!element) return "";

        // GET XPATH FOR ELEMENT
        const parts = [];
        let current = element;

        while (current && current.nodeType === Node.ELEMENT_NODE) {
            let selector = current.nodeName.toLowerCase();

            // ADD INDEX IF NEEDED (SINCE THERE MIGHT BE MULTIPLE ELEMENTS WITH SAME NAME)
            let siblings = Array.from(current.parentNode.children).filter(
                (e) => e.nodeName === current.nodeName,
            );

            if (siblings.length > 1) {
                const index = siblings.indexOf(current) + 1;
                selector += `[${index}]`;
            }

            parts.unshift(selector);
            current = current.parentNode;

            if (current === document) break;
        }

        return "//" + parts.join("/");
    }

    function getDefaultAttribute(tagName, purpose) {
        if (purpose === "assets") {
            if (tagName === "img") return "src";
            if (tagName === "video") return "src";
            if (tagName === "audio") return "src";
            if (tagName === "source") return "src";
            if (tagName === "picture") return "src";
            return "src";
        } else if (purpose === "links" || purpose === "pagination") {
            return "href";
        } else if (purpose === "metadata") {
            return "text";
        }
        return "src";
    }

    function toggleInspectMode() {
        inspectMode = !inspectMode;
    }

    function clearSelections() {
        if (iframeLoaded && iframe?.contentDocument) {
            const elements =
                iframe.contentDocument.querySelectorAll(".__vs_selected");
            elements.forEach((el) => el.classList.remove("__vs_selected"));
        }

        // CLEAR FROM STORE
        jobState.formData.data.visualSelections = [];

        // REMOVE SELECTORS CREATED FROM VISUAL SELECTION
        if (jobState.formData.data.selectors?.length) {
            // THIS IS A SIMPLIFIED APPROACH - IN REAL IMPLEMENTATION
            // WE WOULD NEED A WAY TO TRACK WHICH SELECTORS CAME FROM VISUAL SELECTION
        }
    }

    function updateSelectionType(type) {
        selectedType = type;
        jobState.formData.data.visualSelectionType = type;
    }

    $effect(() => {
        const baseUrl = jobState.formData.data.baseUrl;
        if (baseUrl && !iframeLoaded && !loading) {
            loadIframe();
        }
    });
</script>

<div class="flex flex-col space-y-4">
    <!-- BROWSER UI -->
    <div class="mockup-browser border bg-base-300">
        <div class="mockup-browser-toolbar">
            <div class="input flex-1 join" style="padding-inline: 0;">
                <input
                    type="url"
                    bind:value={jobState.formData.data.baseUrl}
                    placeholder="https://example.com"
                    class="join-item input input-bordered w-full"
                />
                <button class="btn btn-primary join-item" onclick={loadIframe}>
                    <RefreshCw size={16} class="mr-1" />
                    load
                </button>
            </div>
        </div>

        <!-- CONTROLS -->
        <div
            class="bg-base-200 p-2 border-t border-base-300 flex flex-wrap justify-between items-center gap-2"
        >
            <div class="flex items-center gap-2">
                <select
                    bind:value={selectedType}
                    onchange={() => updateSelectionType(selectedType)}
                    class="select select-sm select-bordered"
                >
                    {#each selectionTypes as type}
                        <option value={type.id}>{type.label}</option>
                    {/each}
                </select>

                <button
                    class={`btn btn-sm ${inspectMode ? "btn-primary" : "btn-outline"}`}
                    onclick={toggleInspectMode}
                >
                    {#if inspectMode}
                        <MousePointer size={16} />
                    {:else}
                        <Eye size={16} />
                    {/if}
                    {inspectMode ? "inspecting" : "viewing"}
                </button>
            </div>

            <button class="btn btn-sm btn-outline" onclick={clearSelections}>
                Clear
            </button>
        </div>

        {#if currentSelector}
            <div
                class="bg-base-300 px-3 py-1 text-xs font-mono overflow-x-auto whitespace-nowrap border-t border-base-300"
            >
                {currentSelector}
            </div>
        {/if}

        <!-- IFRAME CONTAINER -->
        <div class="mockup-browser-content bg-base-200 relative">
            {#if loading}
                <div
                    class="absolute inset-0 flex items-center justify-center bg-base-100 bg-opacity-75 z-10"
                >
                    <span
                        class="loading loading-spinner loading-lg text-primary"
                    ></span>
                </div>
            {/if}

            <iframe
                bind:this={iframe}
                title="web page preview"
                class="h-full w-full border-0"
                sandbox="allow-same-origin allow-scripts"
                style="background: white; min-height: 700px; overflow: scroll;"
            ></iframe>

            {#if iframeError && !loading}
                <div
                    class="absolute inset-0 flex flex-col items-center justify-center bg-base-100"
                >
                    <XCircle class="h-12 w-12 text-error mb-4" />
                    <h3 class="text-lg font-medium mb-2">
                        unable to load page content
                    </h3>
                    <p class="text-center max-w-md mb-4">
                        the page could not be loaded. please check the URL and
                        try again.
                    </p>
                </div>
            {/if}
        </div>
    </div>
    {#if jobState.formData.data.visualSelections?.length > 0}
        <div class="bg-base-200 rounded-lg p-4">
            <h3 class="font-medium mb-3">
                selected elements ({jobState.formData.data.visualSelections
                    .length})
            </h3>

            <div class="overflow-x-auto">
                <table class="table table-compact w-full">
                    <thead>
                        <tr>
                            <th>element</th>
                            <th>selector</th>
                            <th>purpose</th>
                            <th>attribute</th>
                            <th>action</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each jobState.formData.data.visualSelections as element}
                            <tr>
                                <td class="font-mono text-xs">{element.tag}</td>
                                <td class="font-mono text-xs truncate max-w-xs"
                                    >{element.cssPath}</td
                                >
                                <td>
                                    <select
                                        class="select select-xs select-bordered w-full max-w-xs"
                                        bind:value={element.purpose}
                                        onchange={() => {
                                            element.attribute =
                                                getDefaultAttribute(
                                                    element.tag,
                                                    element.purpose,
                                                );
                                            // UPDATE CORRESPONDING SELECTOR
                                            removeMatchingSelector(element);
                                            addSelectorFromVisual(element);
                                        }}
                                    >
                                        {#each selectionTypes as type}
                                            <option value={type.id}
                                                >{type.label}</option
                                            >
                                        {/each}
                                    </select>
                                </td>
                                <td>
                                    <select
                                        class="select select-xs select-bordered w-full max-w-xs"
                                        bind:value={element.attribute}
                                        onchange={() => {
                                            // UPDATE CORRESPONDING SELECTOR
                                            const selector =
                                                jobState.formData.data.selectors.find(
                                                    (s) =>
                                                        s.value ===
                                                        element.cssPath,
                                                );
                                            if (selector) {
                                                selector.attribute =
                                                    element.attribute;
                                            }
                                        }}
                                    >
                                        <option value="src">src</option>
                                        <option value="href">href</option>
                                        <option value="text"
                                            >text content</option
                                        >
                                        <option value="html"
                                            >html content</option
                                        >
                                        <option value="data-src"
                                            >data-src</option
                                        >
                                        <option value="alt">alt</option>
                                        <option value="title">title</option>
                                    </select>
                                </td>
                                <td>
                                    <button
                                        class="btn btn-xs btn-error"
                                        onclick={() =>
                                            removeVisualSelection(element)}
                                    >
                                        remove
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
