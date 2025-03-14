<script>
  import { onMount} from 'svelte';
  import Button from "$lib/components/common/Button.svelte";
  import Modal from "$lib/components/common/Modal.svelte";
  import Tabs from "$lib/components/common/Tabs.svelte";
  import ConditionBuilder from "./ConditionBuilder.svelte";
  import {
    ChevronDown,
    Filter,
    RefreshCw,
    ArrowRight,
    ArrowLeft,
    Settings
  } from 'lucide-svelte';
  
  // PROPS
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
  
  // GET REQUIRED FIELDS FOR THIS TASK TYPE
  function getRequiredFieldsForTaskType(taskType) {
    switch(taskType) {
      case 'createBrowser':
        return [];
      case 'createPage':
        return ['browserId'];
      case 'disposeBrowser':
        return ['browserId'];
      case 'disposePage':
        return ['pageId'];
      case 'navigate':
        return ['pageId', 'url'];
      case 'back':
      case 'forward':
      case 'reload':
      case 'waitForLoad':
        return ['pageId'];
      case 'click':
      case 'type':
      case 'select':
      case 'hover':
        return ['pageId', 'selector'];
      case 'extractText':
      case 'extractAttribute':
        return ['pageId', 'selector'];
      case 'extractLinks':
      case 'extractImages':
        return ['pageId'];
      case 'downloadAsset':
        return ['url'];
      case 'saveAsset':
        return ['url', 'jobId'];
      case 'wait':
        return ['duration'];
      case 'executeScript':
        return ['pageId', 'script'];
      default:
        return [];
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
  
  // FETCH TASK INFO
  function getTaskInfo(taskType) {
    // THIS WOULD NORMALLY FETCH FROM THE TASK CATEGORIES
    // IN A REAL IMPLEMENTATION WE'D IMPORT THE TASK CATEGORIES
    return {
      name: taskType.charAt(0).toUpperCase() + taskType.slice(1),
      description: `Task type: ${taskType}`
    };
  }
  
  // GET SOURCES FOR INPUT
  function getSourceName(inputRef) {
    const task = getTaskByOutputRef(inputRef);
    return task ? (task.name || "Unknown") : 'Unknown';
  }
</script>

<Modal
  title={`Configure Task: ${editingTask.name || 'Unnamed Task'}`}
  size="lg"
  isOpen={isOpen}
  onclose={handleClose}
  primaryAction="Save Task"
  primaryVariant="primary"
  secondaryAction="Cancel"
  onprimaryAction={handleSave}
  onsecondaryAction={handleClose}
>
<div class="mb-4">
  <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
    <div>
      <label for="task-name" class="block text-sm font-medium text-dark-300 mb-1">
        Task Name <span class="text-danger-500">*</span>
      </label>
      <input
        id="task-name"
        type="text"
        bind:value={editingTask.name}
        placeholder="Enter task name"
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
      />
    </div>
    <div>
      <label for="edit-task-type" class="block text-sm font-medium text-dark-300 mb-1">
        Task Type
      </label>
      <div id="edit-task-type" class="px-3 py-2 bg-base-800 border border-dark-600 rounded-md">
        {editingTask.type || "Unknown"}
      </div>
    </div>
    <div class="md:col-span-2">
      <label for="task-description" class="block text-sm font-medium text-dark-300 mb-1">
        Description
      </label>
      <textarea
        id="task-description"
        bind:value={editingTask.description}
        placeholder="Describe what this task does"
        rows="2"
        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
      ></textarea>
    </div>
  </div>
  
  <Tabs 
    tabs={[
      { id: 'config', label: 'Configuration', icon: Settings },
      { id: 'inputs', label: 'Input Dependencies', icon: ArrowLeft },
      { id: 'outputs', label: 'Outputs', icon: ArrowRight },
      { id: 'condition', label: 'Execution Condition', icon: Filter },
      { id: 'retry', label: 'Retry Behavior', icon: RefreshCw }
    ]} 
    activeTab={selectedTab}
    onChange={({ tabId }) => { selectedTab = tabId; }}
  >
    <div data-tab="config" class="space-y-4">
      <h3 class="text-md font-medium mb-3">Task Configuration</h3>
      {#each getConfigFieldsForTaskType(editingTask.type || '') as field}
        <div class="mb-4">
          <label for={`field-${field.name}`} class="block text-sm font-medium text-dark-300 mb-1">
            {field.label} {#if requiredFields.includes(field.name)} <span class="text-danger-500">*</span> {/if}
          </label>
          {#if field.type === 'text'}
            <input
              id={`field-${field.name}`}
              type="text"
              bind:value={editingTask.config[field.name]}
              placeholder={`Enter ${field.label.toLowerCase()}`}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500 {fieldValidation[field.name] ? 'border-danger-500' : ''}"
            />
          {:else if field.type === 'number'}
            <input
              id={`field-${field.name}`}
              type="number"
              bind:value={editingTask.config[field.name]}
              placeholder="0"
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500 {fieldValidation[field.name] ? 'border-danger-500' : ''}"
            />
          {:else if field.type === 'checkbox'}
            <div class="flex items-center space-x-2">
              <input 
                id={`field-${field.name}`}
                type="checkbox" 
                bind:checked={editingTask.config[field.name]} 
                class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
              />
              <label for={`field-${field.name}`} class="text-sm text-dark-300">
                {field.description}
              </label>
            </div>
          {:else if field.type === 'select'}
            <select
              id={`field-${field.name}`}
              bind:value={editingTask.config[field.name]}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500 {fieldValidation[field.name] ? 'border-danger-500' : ''}"
            >
              <option value="">Select {field.label}</option>
              {#each field.options as option}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
          {:else if field.type === 'textarea'}
            <textarea
              id={`field-${field.name}`}
              bind:value={editingTask.config[field.name]}
              rows="4"
              placeholder={`Enter ${field.label.toLowerCase()}`}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500 {fieldValidation[field.name] ? 'border-danger-500' : ''}"
            ></textarea>
          {:else if field.type === 'json'}
            {#if typeof editingTask.config[field.name] === 'object'}
              <textarea
                id={`field-${field.name}`}
                bind:value={
                    () => JSON.stringify(editingTask.config[field.name], null, 2),
                    (val) => {
                        editingTask.config[field.name] = JSON.stringify(val, null, 2);
                    }
                }
                rows="4"
                placeholder="Enter JSON"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500 font-mono text-sm {fieldValidation[field.name] ? 'border-danger-500' : ''}"
              ></textarea>
            {:else}
              <textarea
                id={`field-${field.name}`}
                bind:value={editingTask.config[field.name]}
                rows="4"
                placeholder="Enter JSON"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500 font-mono text-sm {fieldValidation[field.name] ? 'border-danger-500' : ''}"
              ></textarea>
            {/if}
          {:else if field.type === 'resource'}
            <select
              id={`field-${field.name}`}
              bind:value={editingTask.config[field.name]}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500 {fieldValidation[field.name] ? 'border-danger-500' : ''}"
            >
              <option value="">Select {field.resourceType}</option>
              {#each availableOutputs.filter(output => {
                // THIS IS A SIMPLIFICATION - IN REALITY, WE'D MATCH RESOURCE TYPES PROPERLY
                if (field.resourceType === 'browser') return output.taskType === 'createBrowser';
                if (field.resourceType === 'page') return output.taskType === 'createPage';
                return true;
              }) as output}
                <option value={output.id}>{output.taskName} ({output.id})</option>
              {/each}
            </select>
          {/if}
          {#if fieldValidation[field.name]}
            <p class="text-danger-500 text-xs mt-1">{fieldValidation[field.name]}</p>
          {/if}
          {#if field.description && field.type !== 'checkbox'}
            <p class="text-dark-400 text-xs mt-1">{field.description}</p>
          {/if}
        </div>
      {/each}
      {#if getConfigFieldsForTaskType(editingTask.type || '').length === 0}
        <p class="text-dark-400 italic">This task type doesn't require any configuration.</p>
      {/if}
    </div>
    
    <div data-tab="inputs" class="space-y-4">
      <div class="flex justify-between items-center mb-3">
        <h3 class="text-md font-medium">Input Dependencies</h3>
        <Button 
          variant="outline" 
          size="sm" 
          onclick={() => showAddInputModal = true}
        >
          <ArrowLeft class="h-4 w-4 mr-1" />
          Add Input
        </Button>
      </div>
      {#if availableInputs.length === 0}
        <div class="bg-base-700 rounded-lg p-4 text-center">
          <p class="text-dark-400">No input dependencies configured.</p>
          <p class="text-xs text-dark-400 mt-1">
            Connect inputs from other task outputs to use as input for this task.
          </p>
        </div>
      {:else}
        <div class="space-y-2">
          {#each availableInputs as input}
            <div class="bg-base-700 rounded-lg p-3 flex justify-between items-center">
              <div>
                <div class="flex items-center">
                  <ArrowLeft class="h-4 w-4 mr-2 text-blue-400" />
                  <span class="font-medium">{input.taskName}</span>
                </div>
                <div class="text-xs text-dark-400 mt-1">
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
                class="text-danger-400 hover:text-danger-300"
              >
                Remove
              </Button>
            </div>
          {/each}
        </div>
      {/if}
    </div>
    
    <div data-tab="outputs" class="space-y-4">
      <h3 class="text-md font-medium mb-3">Task Output</h3>
      <div class="bg-base-700 rounded-lg p-4">
        <div class="mb-3">
          <label for="output-ref" class="block text-sm font-medium text-dark-300 mb-1">
            Output Reference ID
          </label>
          <input
            id="output-ref"
            type="text"
            bind:value={editingTask.outputRef}
            placeholder="Auto-generated output ID"
            class="w-full px-3 py-2 bg-base-800 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
          <p class="text-dark-400 text-xs mt-1">
            This ID will be used by other tasks to refer to this task's output.
          </p>
        </div>
        <div>
          <h4 class="text-sm font-medium text-dark-300 mb-1">Expected Output Type</h4>
          <div class="bg-base-800 rounded-md p-2 text-sm">
            {#if editingTask.type === 'extractText' || editingTask.type === 'extractAttribute'}
              <span class="text-amber-400">string</span> or <span class="text-amber-400">array</span> (if multiple)
            {:else if editingTask.type === 'extractLinks' || editingTask.type === 'extractImages' || editingTask.type === 'loop'}
              <span class="text-amber-400">array</span>
            {:else if editingTask.type === 'click' || editingTask.type === 'type' || editingTask.type === 'wait'}
              <span class="text-amber-400">boolean</span>
            {:else if editingTask.type === 'createBrowser' || editingTask.type === 'createPage'}
              <span class="text-amber-400">object</span> (resource ID)
            {:else if editingTask.type === 'downloadAsset' || editingTask.type === 'saveAsset'}
              <span class="text-amber-400">object</span> (asset info)
            {:else if editingTask.type === 'navigate'}
              <span class="text-amber-400">object</span> (navigation result)
            {:else if editingTask.type === 'executeScript'}
              <span class="text-amber-400">any</span> (depends on script)
            {:else if editingTask.type === 'conditional'}
              <span class="text-amber-400">any</span> (depends on condition)
            {:else}
              <span class="text-amber-400">any</span>
            {/if}
          </div>
        </div>
      </div>
    </div>
    
    <div data-tab="condition" class="space-y-4">
      <h3 class="text-md font-medium mb-3">Execution Condition</h3>
      <ConditionBuilder bind:condition={editingTask.condition} />
      <div class="mt-2 p-3 bg-amber-900/20 border border-amber-900/30 rounded-md">
        <h4 class="text-sm font-medium mb-1 flex items-center">
          <Filter class="h-4 w-4 mr-1" />
          About Conditions
        </h4>
        <p class="text-sm text-dark-300">
          Task conditions determine whether this task will run during execution. 
          If the condition evaluates to false, the task will be skipped.
        </p>
      </div>
    </div>
    
    <div data-tab="retry" class="space-y-4">
      <h3 class="text-md font-medium mb-3">Retry Configuration</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="max-retries" class="block text-sm font-medium text-dark-300 mb-1">
            Maximum Retries
          </label>
          <input
            id="max-retries"
            type="number"
            min="0"
            max="10"
            bind:value={editingTask.retryConfig.maxRetries}
            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
          <p class="text-dark-400 text-xs mt-1">
            Number of times to retry the task if it fails. Set to 0 to disable retries.
          </p>
        </div>
        <div>
          <label for="retry-delay" class="block text-sm font-medium text-dark-300 mb-1">
            Initial Delay (ms)
          </label>
          <input
            id="retry-delay"
            type="number"
            min="0"
            bind:value={editingTask.retryConfig.delayMS}
            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
          <p class="text-dark-400 text-xs mt-1">
            Initial delay before first retry attempt (in milliseconds).
          </p>
        </div>
        <div>
          <label for="backoff-rate" class="block text-sm font-medium text-dark-300 mb-1">
            Backoff Rate
          </label>
          <input
            id="backoff-rate"
            type="number"
            min="1"
            step="0.1"
            bind:value={editingTask.retryConfig.backoffRate}
            class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
          <p class="text-dark-400 text-xs mt-1">
            Multiplier for delay between retry attempts (e.g., 1.5 means each retry waits 1.5x longer).
          </p>
        </div>
      </div>
      {#if editingTask.retryConfig && editingTask.retryConfig.maxRetries > 0}
        <div class="mt-4 p-3 bg-base-700 rounded-md">
          <h4 class="text-sm font-medium mb-2">Retry Schedule Preview</h4>
          <div class="space-y-1 text-sm">
            {#each Array(Math.min(editingTask.retryConfig.maxRetries, 5)).fill(0) as _, index}
              {@const delay = editingTask.retryConfig.delayMS * Math.pow(editingTask.retryConfig.backoffRate, index)}
              <div class="flex items-center">
                <span class="w-20">Retry {index + 1}:</span>
                <span class="text-primary-400">{delay.toFixed(0)} ms</span>
                {#if index === 0}
                  <span class="ml-2 text-xs text-dark-400">(initial delay)</span>
                {/if}
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  </Tabs>
</div>
</Modal>

<!-- ADD INPUT MODAL -->
{#if showAddInputModal}
<Modal 
  title="Add Input Dependency"
  size="md"
  isOpen={showAddInputModal}
  onclose={() => showAddInputModal = false}
>
  <div class="mb-4">
    <p class="text-sm text-dark-300 mb-4">
      Select an output from another task to use as input for this task.
    </p>
    {#if availableOutputs.length === 0}
      <div class="bg-base-700 rounded-lg p-4 text-center">
        <p class="text-dark-400">No available outputs found.</p>
        <p class="text-xs text-dark-400 mt-1">
          Create other tasks with outputs first, then connect them as inputs to this task.
        </p>
      </div>
    {:else}
      <div class="space-y-2 max-h-64 overflow-y-auto pr-2">
        {#each availableOutputs as output}
          <button
            class="w-full bg-base-700 hover:bg-base-600 rounded-lg p-3 text-left transition-colors border border-transparent hover:border-primary-500"
            onclick={() => addInputRef(output.id)}
          >
            <div class="flex items-center">
              <ArrowLeft class="h-4 w-4 mr-2 text-blue-400" />
              <span class="font-medium">{output.taskName}</span>
            </div>
            <div class="text-xs text-dark-400 mt-1">
              Type: {output.taskType} (ID: {output.id})
            </div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
  <div slot="footer" class="flex justify-end">
    <Button variant="outline" onclick={() => showAddInputModal = false}>
      Cancel
    </Button>
  </div>
</Modal>
{/if}

<!-- REMOVE INPUT MODAL -->
{#if showRemoveInputModal}
<Modal 
  title="Remove Input Dependency"
  size="sm"
  isOpen={showRemoveInputModal}
  primaryAction="Remove"
  primaryVariant="danger"
  secondaryAction="Cancel"
  onprimaryAction={removeInputRef}
  onsecondaryAction={() => showRemoveInputModal = false}
>
  <p class="text-sm text-dark-300">
    Are you sure you want to remove the input dependency from 
    <span class="font-medium">{getSourceName(inputToRemove)}</span>?
  </p>
  <p class="text-xs text-dark-400 mt-2">
    This will disconnect this task from using that task's output as input.
  </p>
</Modal>
{/if}
