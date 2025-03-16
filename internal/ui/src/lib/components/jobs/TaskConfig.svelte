<script>
  import Button from "$lib/components/common/Button.svelte";
  import ConditionBuilder from "./ConditionBuilder.svelte";
  import {
    ChevronDown,
    Filter,
    RefreshCw,
    ArrowRight,
    ArrowLeft,
    Settings,
    Grip
  } from 'lucide-svelte';

  // PROPS USING SVELTE 5 RUNES
  let {
    task = null,
    allTasks = [],
    isOpen = false,
    onclose = () => {},
    onsave = () => {}
  } = $props();

  // LOCAL STATE
  let editingTask = $state({});
  let selectedTab = $state('config');
  let availableInputs = $state([]);
  let availableOutputs = $state([]);
  let showAddInputModal = $state(false);
  let showRemoveInputModal = $state(false);
  let inputToRemove = $state(null);
  
  // INPUT FIELD VALIDATION
  let fieldValidation = $state({});
  let requiredFields = $state([]);

  // INITIALIZE
  $effect(() => {
    if (isOpen && task) {
      // CREATE A DEEP COPY TO AVOID DIRECT MODIFICATION
      try {
        editingTask = JSON.parse(JSON.stringify(task));
      } catch (e) {
        console.error("Error cloning task:", e);
        editingTask = { ...task };
      }
      
      // ENSURE CONFIG EXISTS
      if (!editingTask.config) {
        editingTask.config = {};
      }
      
      // ENSURE INPUT REFS ARRAY EXISTS
      if (!editingTask.inputRefs) {
        editingTask.inputRefs = [];
      }
      
      // ENSURE OUTPUT REF EXISTS
      if (!editingTask.outputRef) {
        editingTask.outputRef = `output_${Math.random().toString(36).substr(2, 9)}`;
      }
      
      // ENSURE CONDITION EXISTS
      if (!editingTask.condition) {
        editingTask.condition = { type: "always", config: {} };
      }
      
      // ENSURE RETRY CONFIG EXISTS
      if (!editingTask.retryConfig) {
        editingTask.retryConfig = { maxRetries: 3, delayMS: 1000, backoffRate: 1.5 };
      }
      
      // UPDATE AVAILABLE REFERENCES
      updateAvailableReferences();
      
      // GET REQUIRED FIELDS
      requiredFields = getRequiredFieldsForTaskType(editingTask.type);
    }
  });

  // ADD INPUT REFERENCE
  function addInputRef(outputRef) {
    if (!editingTask.inputRefs) {
      editingTask.inputRefs = [];
    }
    if (!editingTask.inputRefs.includes(outputRef)) {
      editingTask.inputRefs = [...editingTask.inputRefs, outputRef];
    }
    showAddInputModal = false;
  }

  // REMOVE INPUT REFERENCE
  function removeInputRef() {
    if (inputToRemove && editingTask.inputRefs) {
      editingTask.inputRefs = editingTask.inputRefs.filter(ref => ref !== inputToRemove);
    }
    inputToRemove = null;
    showRemoveInputModal = false;
  }

  // GET TASK AND SOURCE BY OUTPUT REFERENCE
  function getTaskByOutputRef(outputRef) {
    for (const task of allTasks) {
      if (task.outputRef === outputRef) {
        return task;
      }
    }
    return null;
  }

  // UPDATE AVAILABLE INPUTS AND OUTPUTS
  function updateAvailableReferences() {
    // Get all outputs from other tasks
    availableOutputs = allTasks
      .filter(t => t.id !== editingTask.id && t.outputRef)
      .map(t => ({
        id: t.outputRef,
        taskName: t.name || 'Unnamed task',
        taskType: t.type || 'unknown'
      }));
      
    // Get existing inputs
    availableInputs = (editingTask.inputRefs || [])
      .map(ref => {
        const sourceTask = getTaskByOutputRef(ref);
        return {
          id: ref,
          taskName: sourceTask ? (sourceTask.name || 'Unknown task') : 'Unknown task',
          taskType: sourceTask ? (sourceTask.type || 'unknown') : 'unknown'
        };
      });
  }

  // HANDLE CLOSE
  function handleClose() {
    onclose();
  }

  // HANDLE SAVE
  function handleSave() {
    // VALIDATE REQUIRED FIELDS
    let valid = true;
    let newValidation = {};
    requiredFields.forEach(field => {
      if (!editingTask.config[field] || editingTask.config[field] === '') {
        newValidation[field] = 'This field is required';
        valid = false;
      }
    });
    
    if (!valid) {
      fieldValidation = newValidation;
      return;
    }
    
    onsave(editingTask);
  }

  // GET REQUIRED FIELDS FOR THIS TASK TYPE
  function getRequiredFieldsForTaskType(taskType) {
    switch(taskType) {
      case 'createBrowser': return [];
      case 'createPage': return ['browserId'];
      case 'disposeBrowser': return ['browserId'];
      case 'disposePage': return ['pageId'];
      case 'navigate': return ['pageId', 'url'];
      case 'back':
      case 'forward':
      case 'reload':
      case 'waitForLoad': return ['pageId'];
      case 'click':
      case 'type':
      case 'select':
      case 'hover': return ['pageId', 'selector'];
      case 'extractText':
      case 'extractAttribute': return ['pageId', 'selector'];
      case 'extractLinks':
      case 'extractImages': return ['pageId'];
      case 'downloadAsset': return ['url'];
      case 'saveAsset': return ['url', 'jobId'];
      case 'wait': return ['duration'];
      case 'executeScript': return ['pageId', 'script'];
      default: return [];
    }
  }

  // GENERATE CONFIG FIELDS FOR TASK TYPE
  function getConfigFieldsForTaskType(taskType) {
    switch(taskType) {
      case 'createBrowser':
        return [
          {
            name: 'headless',
            label: 'Headless Mode',
            type: 'checkbox',
            description: 'Run browser without visible UI'
          },
          {
            name: 'userAgent',
            label: 'User Agent',
            type: 'text',
            description: 'Custom user agent string'
          }
        ];
      case 'createPage':
        return [
          {
            name: 'browserId',
            label: 'Browser ID',
            type: 'resource',
            resourceType: 'browser',
            description: 'Browser instance to use'
          },
          {
            name: 'viewportWidth',
            label: 'Viewport Width',
            type: 'number',
            description: 'Width of the browser viewport in pixels'
          },
          {
            name: 'viewportHeight',
            label: 'Viewport Height',
            type: 'number',
            description: 'Height of the browser viewport in pixels'
          },
          {
            name: 'recordVideo',
            label: 'Record Video',
            type: 'checkbox',
            description: 'Record browser activity as video'
          }
        ];
      case 'navigate':
        return [
          {
            name: 'pageId',
            label: 'Page ID',
            type: 'resource',
            resourceType: 'page',
            description: 'Page to navigate'
          },
          {
            name: 'url',
            label: 'URL',
            type: 'text',
            description: 'URL to navigate to'
          },
          {
            name: 'waitUntil',
            label: 'Wait Until',
            type: 'select',
            options: [
              { value: 'load', label: 'load' },
              { value: 'domcontentloaded', label: 'domcontentloaded' },
              { value: 'networkidle', label: 'networkidle' }
            ],
            description: 'When to consider navigation complete'
          },
          {
            name: 'timeout',
            label: 'Timeout (ms)',
            type: 'number',
            description: 'Navigation timeout in milliseconds'
          }
        ];
      case 'click':
        return [
          {
            name: 'pageId',
            label: 'Page ID',
            type: 'resource',
            resourceType: 'page',
            description: 'Page containing element'
          },
          {
            name: 'selector',
            label: 'Selector',
            type: 'text',
            description: 'CSS selector for element to click'
          },
          {
            name: 'button',
            label: 'Mouse Button',
            type: 'select',
            options: [
              { value: 'left', label: 'Left' },
              { value: 'middle', label: 'Middle' },
              { value: 'right', label: 'Right' }
            ],
            description: 'Which mouse button to use'
          },
          {
            name: 'clickCount',
            label: 'Click Count',
            type: 'number',
            description: 'Number of clicks (1 for single, 2 for double)'
          },
          {
            name: 'timeout',
            label: 'Timeout (ms)',
            type: 'number',
            description: 'Maximum time to wait for element'
          }
        ];
      case 'type':
        return [
          {
            name: 'pageId',
            label: 'Page ID',
            type: 'resource',
            resourceType: 'page',
            description: 'Page containing element'
          },
          {
            name: 'selector',
            label: 'Selector',
            type: 'text',
            description: 'CSS selector for input element'
          },
          {
            name: 'text',
            label: 'Text',
            type: 'text',
            description: 'Text to type into element'
          },
          {
            name: 'delay',
            label: 'Delay between keystrokes (ms)',
            type: 'number',
            description: 'Delay between keystrokes in milliseconds'
          },
          {
            name: 'clear',
            label: 'Clear field first',
            type: 'checkbox',
            description: 'Clear existing text before typing'
          }
        ];
      case 'extractText':
        return [
          {
            name: 'pageId',
            label: 'Page ID',
            type: 'resource',
            resourceType: 'page',
            description: 'Page containing element'
          },
          {
            name: 'selector',
            label: 'Selector',
            type: 'text',
            description: 'CSS selector for element to extract text from'
          },
          {
            name: 'multiple',
            label: 'Extract from multiple elements',
            type: 'checkbox',
            description: 'Extract text from all matching elements'
          },
          {
            name: 'trim',
            label: 'Trim whitespace',
            type: 'checkbox',
            description: 'Remove leading/trailing whitespace'
          }
        ];
      case 'downloadAsset':
        return [
          {
            name: 'url',
            label: 'URL',
            type: 'text',
            description: 'URL of asset to download'
          },
          {
            name: 'folder',
            label: 'Folder',
            type: 'text',
            description: 'Folder to save asset in'
          },
          {
            name: 'filename',
            label: 'Filename',
            type: 'text',
            description: 'Optional filename (auto-generated if empty)'
          },
          {
            name: 'timeout',
            label: 'Timeout (ms)',
            type: 'number',
            description: 'Download timeout in milliseconds'
          }
        ];
      case 'wait':
        return [
          {
            name: 'duration',
            label: 'Duration (ms)',
            type: 'number',
            description: 'Time to wait in milliseconds'
          }
        ];
      case 'executeScript':
        return [
          {
            name: 'pageId',
            label: 'Page ID',
            type: 'resource',
            resourceType: 'page',
            description: 'Page to execute script on'
          },
          {
            name: 'script',
            label: 'JavaScript Code',
            type: 'textarea',
            description: 'JavaScript code to execute in the page context'
          },
          {
            name: 'args',
            label: 'Arguments',
            type: 'json',
            description: 'JSON array of arguments to pass to the script'
          }
        ];
      case 'conditional':
        return [
          {
            name: 'condition',
            label: 'Condition',
            type: 'text',
            description: 'Expression or value to evaluate as condition'
          },
          {
            name: 'ifTrue',
            label: 'Value if True',
            type: 'json',
            description: 'Value to return if condition is true'
          },
          {
            name: 'ifFalse',
            label: 'Value if False',
            type: 'json',
            description: 'Value to return if condition is false'
          }
        ];
      case 'loop':
        return [
          {
            name: 'items',
            label: 'Items',
            type: 'json',
            description: 'Array of items to iterate over'
          },
          {
            name: 'parallelProcessing',
            label: 'Process in parallel',
            type: 'checkbox',
            description: 'Process items in parallel'
          },
          {
            name: 'maxWorkers',
            label: 'Maximum Workers',
            type: 'number',
            description: 'Maximum number of parallel workers'
          }
        ];
      default:
        return [];
    }
  }

  // GET SOURCES FOR INPUT
  function getSourceName(inputRef) {
    const task = getTaskByOutputRef(inputRef);
    return task ? (task.name || "Unknown") : 'Unknown';
  }
</script>

{#if isOpen}
  <div class="modal modal-open fixed z-54 overflow-y-auto bg-black bg-opacity-50 flex items-center justify-center">
    <div class="modal-box bg-base-200 z-53 rounded-lg shadow-xl w-full max-w-4xl">
      <div class="flex justify-between items-center p-6 border-b border-base-300">
        <h3 class="text-xl font-bold">Configure Task: {editingTask.name || 'Unnamed Task'}</h3>
        <button 
          class="btn btn-sm btn-circle" 
          onclick={handleClose}
          aria-label="Close modal"
        >✕</button>
      </div>
      
      <div class="p-6">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          <div class="form-control">
            <label for="task-name" class="label">
              <span class="label-text font-medium">Task Name <span class="text-error">*</span></span>
            </label>
            <input
              id="task-name"
              type="text"
              bind:value={editingTask.name}
              placeholder="Enter task name"
              class="input input-bordered w-full"
            />
          </div>
          <div class="form-control">
            <label for="edit-task-type" class="label">
              <span class="label-text font-medium">Task Type</span>
            </label>
            <input
              id="edit-task-type"
              type="text"
              value={editingTask.type || "Unknown"}
              class="input input-bordered w-full"
              readonly
            />
          </div>
          <div class="form-control md:col-span-2">
            <label for="task-description" class="label">
              <span class="label-text font-medium">Description</span>
            </label>
            <textarea
              id="task-description"
              bind:value={editingTask.description}
              placeholder="Describe what this task does"
              rows="2"
              class="textarea textarea-bordered w-full"
            ></textarea>
          </div>
        </div>
        
        <div class="tabs tabs-boxed mb-6">
          <button 
            class={`tab ${selectedTab === 'config' ? 'tab-active' : ''}`}
            onclick={() => selectedTab = 'config'}
          >
            <Settings class="h-4 w-4 mr-1" />
            Configuration
          </button>
          <button 
            class={`tab ${selectedTab === 'inputs' ? 'tab-active' : ''}`}
            onclick={() => selectedTab = 'inputs'}
          >
            <ArrowLeft class="h-4 w-4 mr-1" />
            Input Dependencies
          </button>
          <button 
            class={`tab ${selectedTab === 'outputs' ? 'tab-active' : ''}`}
            onclick={() => selectedTab = 'outputs'}
          >
            <ArrowRight class="h-4 w-4 mr-1" />
            Outputs
          </button>
          <button 
            class={`tab ${selectedTab === 'condition' ? 'tab-active' : ''}`}
            onclick={() => selectedTab = 'condition'}
          >
            <Filter class="h-4 w-4 mr-1" />
            Execution Condition
          </button>
          <button 
            class={`tab ${selectedTab === 'retry' ? 'tab-active' : ''}`}
            onclick={() => selectedTab = 'retry'}
          >
            <RefreshCw class="h-4 w-4 mr-1" />
            Retry Behavior
          </button>
        </div>
        
        <div class="bg-base-100 p-6 rounded-lg">
          <!-- CONFIG TAB -->
          {#if selectedTab === 'config'}
            <h3 class="text-lg font-medium mb-6">Task Configuration</h3>
            {#each getConfigFieldsForTaskType(editingTask.type || '') as field, fieldIndex}
              <div class="form-control mb-6">
                {#if field.type === 'checkbox'}
                  <!-- Checkbox field -->
                  <div class="flex items-center space-x-3 mt-1">
                    <input 
                      id={`field-${field.name}-${fieldIndex}`}
                      type="checkbox" 
                      bind:checked={editingTask.config[field.name]} 
                      class="checkbox"
                    />
                    <label for={`field-${field.name}-${fieldIndex}`} class="label-text cursor-pointer">
                      {field.label}
                    </label>
                  </div>
                  {#if field.description}
                    <div class="text-xs opacity-70 mt-1 ml-7">
                      {field.description}
                    </div>
                  {/if}
                {:else}
                  <label for={`field-${field.name}-${fieldIndex}`} class="label">
                    <span class="label-text font-medium">{field.label} {#if requiredFields.includes(field.name)}<span class="text-error">*</span>{/if}</span>
                  </label>
                  {#if field.type === 'text'}
                    <input
                      id={`field-${field.name}-${fieldIndex}`}
                      type="text"
                      bind:value={editingTask.config[field.name]}
                      placeholder={`Enter ${field.label.toLowerCase()}`}
                      class="input input-bordered w-full"
                    />
                  {:else if field.type === 'number'}
                    <input
                      id={`field-${field.name}-${fieldIndex}`}
                      type="number"
                      bind:value={editingTask.config[field.name]}
                      placeholder="0"
                      class="input input-bordered w-full"
                    />
                  {:else if field.type === 'select'}
                    <select
                      id={`field-${field.name}-${fieldIndex}`}
                      bind:value={editingTask.config[field.name]}
                      class="select select-bordered w-full"
                    >
                      <option value="">Select {field.label}</option>
                      {#each field.options as option}
                        <option value={option.value}>{option.label}</option>
                      {/each}
                    </select>
                  {:else if field.type === 'textarea'}
                    <textarea
                      id={`field-${field.name}-${fieldIndex}`}
                      bind:value={editingTask.config[field.name]}
                      rows="4"
                      placeholder={`Enter ${field.label.toLowerCase()}`}
                      class="textarea textarea-bordered w-full"
                    ></textarea>
                  {:else if field.type === 'resource'}
                    <select
                      id={`field-${field.name}-${fieldIndex}`}
                      bind:value={editingTask.config[field.name]}
                      class="select select-bordered w-full"
                    >
                      <option value="">Select {field.resourceType}</option>
                      {#each availableOutputs.filter(output => {
                        if (field.resourceType === 'browser') return output.taskType === 'createBrowser';
                        if (field.resourceType === 'page') return output.taskType === 'createPage';
                        return true;
                      }) as output}
                        <option value={output.id}>{output.taskName} ({output.id})</option>
                      {/each}
                    </select>
                  {/if}
                  {#if fieldValidation[field.name]}
                    <div class="mt-1 text-xs text-error">
                      {fieldValidation[field.name]}
                    </div>
                  {/if}
                  {#if field.description}
                    <div class="label">
                      <span class="label-text-alt text-xs opacity-70">
                        {field.description}
                      </span>
                    </div>
                  {/if}
                {/if}
              </div>
            {/each}
            {#if getConfigFieldsForTaskType(editingTask.type || '').length === 0}
              <p class="text-base-content opacity-70 italic">This task type doesn't require any specific configuration.</p>
            {/if}
          {/if}
          
          <!-- INPUTS TAB -->
          {#if selectedTab === 'inputs'}
            <div class="flex justify-between items-center mb-6">
              <h3 class="text-lg font-medium">Input Dependencies</h3>
              <Button 
                variant="outline" 
                size="sm" 
                onclick={() => showAddInputModal = true}
              >
                Add Input
              </Button>
            </div>
            
            {#if !availableInputs.length}
              <div class="bg-base-200 rounded-lg p-6 text-center">
                <p class="opacity-70">No input dependencies configured.</p>
                <p class="text-xs opacity-60 mt-2">
                  Connect inputs from other task outputs to use as input for this task.
                </p>
              </div>
            {:else}
              <div class="space-y-3">
                {#each availableInputs as input}
                  <div class="bg-base-200 rounded-lg p-4 flex justify-between items-center">
                    <div>
                      <div class="flex items-center">
                        <ArrowLeft class="h-4 w-4 mr-2 text-primary" />
                        <span class="font-medium">{input.taskName}</span>
                      </div>
                      <div class="text-xs text-base-content opacity-60 mt-1">
                        Source: {input.taskType} (ID: {input.id})
                      </div>
                    </div>
                    <Button 
                      variant="ghost" 
                      size="sm" 
                      onclick={() => {
                        inputToRemove = input.id;
                        showRemoveInputModal = true;
                      }}
                      class="text-error hover:text-error hover:bg-base-300"
                    >
                      Remove
                    </Button>
                  </div>
                {/each}
              </div>
            {/if}
          {/if}
          
          <!-- OUTPUTS TAB -->
          {#if selectedTab === 'outputs'}
            <h3 class="text-lg font-medium mb-6">Task Output</h3>
            <div class="bg-base-200 rounded-lg p-6">
              <div class="mb-6 form-control">
                <label for="output-ref" class="label">
                  <span class="label-text font-medium">Output Reference ID</span>
                </label>
                <input
                  id="output-ref"
                  type="text"
                  bind:value={editingTask.outputRef}
                  placeholder="Auto-generated output ID"
                  class="input input-bordered w-full"
                />
                <div class="label">
                  <span class="label-text-alt text-xs opacity-70">
                    This ID will be used by other tasks to refer to this task's output.
                  </span>
                </div>
              </div>
              
              <div class="form-control">
                <label for="output-type" class="label">
                  <span class="label-text font-medium">Expected Output Type</span>
                </label>
                <div id="output-type" class="bg-base-300 rounded-md p-4 text-sm">
                  {#if editingTask.type === 'extractText' || editingTask.type === 'extractAttribute'}
                    <span class="text-warning">string</span> or <span class="text-warning">array</span> (if multiple)
                  {:else if editingTask.type === 'extractLinks' || editingTask.type === 'extractImages' || editingTask.type === 'loop'}
                    <span class="text-warning">array</span>
                  {:else if editingTask.type === 'click' || editingTask.type === 'type' || editingTask.type === 'wait'}
                    <span class="text-warning">boolean</span>
                  {:else if editingTask.type === 'createBrowser' || editingTask.type === 'createPage'}
                    <span class="text-warning">object</span> (resource ID)
                  {:else if editingTask.type === 'downloadAsset' || editingTask.type === 'saveAsset'}
                    <span class="text-warning">object</span> (asset info)
                  {:else if editingTask.type === 'navigate'}
                    <span class="text-warning">object</span> (navigation result)
                  {:else if editingTask.type === 'executeScript'}
                    <span class="text-warning">any</span> (depends on script)
                  {:else if editingTask.type === 'conditional'}
                    <span class="text-warning">any</span> (depends on condition)
                  {:else}
                    <span class="text-warning">any</span>
                  {/if}
                </div>
              </div>
            </div>
          {/if}
          
          <!-- CONDITION TAB -->
          {#if selectedTab === 'condition'}
            <h3 class="text-lg font-medium mb-6">Execution Condition</h3>
            <ConditionBuilder bind:condition={editingTask.condition} />
            <div class="mt-6 p-4 bg-warning/10 border border-warning/30 rounded-md">
              <h4 class="text-sm font-medium mb-2 flex items-center">
                <Filter class="h-4 w-4 mr-1" />
                About Conditions
              </h4>
              <p class="text-sm opacity-70">
                Task conditions determine whether this task will run during execution. 
                If the condition evaluates to false, the task will be skipped.
              </p>
            </div>
          {/if}
          
          <!-- RETRY TAB -->
          {#if selectedTab === 'retry'}
            <h3 class="text-lg font-medium mb-6">Retry Configuration</h3>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div class="form-control">
                <label for="max-retries" class="label">
                  <span class="label-text font-medium">Maximum Retries</span>
                </label>
                <input
                  id="max-retries"
                  type="number"
                  min="0"
                  max="10"
                  bind:value={editingTask.retryConfig.maxRetries}
                  class="input input-bordered w-full"
                />
                <div class="label">
                  <span class="label-text-alt text-xs opacity-70">
                    Number of times to retry the task if it fails. Set to 0 to disable retries.
                  </span>
                </div>
              </div>
              <div class="form-control">
                <label for="retry-delay" class="label">
                  <span class="label-text font-medium">Initial Delay (ms)</span>
                </label>
                <input
                  id="retry-delay"
                  type="number"
                  min="0"
                  bind:value={editingTask.retryConfig.delayMS}
                  class="input input-bordered w-full"
                />
                <div class="label">
                  <span class="label-text-alt text-xs opacity-70">
                    Initial delay before first retry attempt (in milliseconds).
                  </span>
                </div>
              </div>
              <div class="form-control">
                <label for="backoff-rate" class="label">
                  <span class="label-text font-medium">Backoff Rate</span>
                </label>
                <input
                  id="backoff-rate"
                  type="number"
                  min="1"
                  step="0.1"
                  bind:value={editingTask.retryConfig.backoffRate}
                  class="input input-bordered w-full"
                />
                <div class="label">
                  <span class="label-text-alt text-xs opacity-70">
                    Multiplier for delay between retry attempts (e.g., 1.5 means each retry waits 1.5x longer).
                  </span>
                </div>
              </div>
            </div>
          {/if}
        </div>
      </div>
      
      <div class="p-6 border-t border-base-300 flex justify-end space-x-3">
        <Button variant="ghost" onclick={handleClose}>Cancel</Button>
        <Button variant="primary" onclick={handleSave}>Save Task</Button>
      </div>
    </div>
  </div>

  <!-- ADD INPUT MODAL -->
  {#if showAddInputModal}
    <div class="modal modal-open z-60 flex items-center justify-center p-4 bg-black bg-opacity-70">
      <div class="modal-box bg-base-200 rounded-lg shadow-xl w-full max-w-xl">
        <div class="p-6 border-b border-base-300">
          <h3 class="font-bold text-lg mb-0">Add Input Dependency</h3>
        </div>
        
        <div class="p-6">
          <p class="mb-4 text-base-content opacity-70">
            Select an output from another task to use as input for this task.
          </p>
          
          {#if !availableOutputs.length}
            <div class="bg-base-100 rounded-lg p-6 text-center">
              <p class="text-base-content opacity-70">No available outputs found.</p>
              <p class="text-xs text-base-content opacity-60 mt-2">
                Create other tasks with outputs first, then connect them as inputs to this task.
              </p>
            </div>
          {:else}
            <div class="space-y-2 max-h-64 overflow-y-auto pr-2">
              {#each availableOutputs as output}
                <button
                  class="w-full bg-base-100 hover:bg-base-300 rounded-lg p-4 text-left transition-colors border border-base-300 hover:border-primary"
                  onclick={() => addInputRef(output.id)}
                >
                  <div class="flex items-center">
                    <ArrowLeft class="h-4 w-4 mr-2 text-primary" />
                    <span class="font-medium">{output.taskName}</span>
                  </div>
                  <div class="text-xs text-base-content opacity-60 mt-1">
                    Type: {output.taskType} (ID: {output.id})
                  </div>
                </button>
              {/each}
            </div>
          {/if}
        </div>
        
        <div class="p-4 border-t border-base-300 flex justify-end">
          <Button variant="ghost" onclick={() => showAddInputModal = false}>Cancel</Button>
        </div>
      </div>
    </div>
  {/if}

  <!-- REMOVE INPUT MODAL -->
  {#if showRemoveInputModal}
    <div class="modal modal-open fixed z-60 flex items-center justify-center p-4 bg-black bg-opacity-70">
      <div class="modal-box bg-base-200 rounded-lg shadow-xl w-full max-w-md">
        <div class="p-6 border-b border-base-300">
          <h3 class="font-bold text-lg mb-0">Remove Input Dependency</h3>
        </div>
        
        <div class="p-6">
          <p class="text-base-content opacity-80">
            Are you sure you want to remove the input dependency from 
            <span class="font-medium">{getSourceName(inputToRemove)}</span>?
          </p>
          <p class="text-sm opacity-60 mt-2">
            This will disconnect this task from using that task's output as input.
          </p>
        </div>
        
        <div class="p-4 border-t border-base-300 flex justify-end space-x-3">
          <Button variant="ghost" onclick={() => showRemoveInputModal = false}>Cancel</Button>
          <Button variant="error" onclick={removeInputRef}>Remove</Button>
        </div>
      </div>
    </div>
  {/if}
{/if}