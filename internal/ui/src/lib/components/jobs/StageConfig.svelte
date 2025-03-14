<script>
  import { createEventDispatcher } from 'svelte';
  import Button from "$lib/components/common/Button.svelte";
  import Modal from "$lib/components/common/Modal.svelte";
  import Tabs from "$lib/components/common/Tabs.svelte";
  import ConditionBuilder from "./ConditionBuilder.svelte";
  import {
    ArrowLeftRight,
    ArrowDownUp,
    Settings,
    Filter,
    Layers
  } from 'lucide-svelte';
  
  // PROPS
  let {
    stage = null,
    isOpen = false,
    onclose = () => {},
    onsave = () => {}
  } = $props();
  
  // LOCAL STATE
  let editingStage = $state({
    id: "",
    name: "",
    description: "",
    condition: { type: "always", config: {} },
    parallelism: { mode: "sequential", maxWorkers: 1 },
    tasks: [],
    config: {}
  });
  
  let selectedTab = $state('basic');
  
  // PARALLELISM MODES
  const parallelismModes = [
    { id: 'sequential', name: 'Sequential', description: 'Execute tasks one after another' },
    { id: 'parallel', name: 'Parallel', description: 'Execute multiple tasks simultaneously' },
    { id: 'worker-per-item', name: 'Worker Per Item', description: 'Process collections in parallel' }
  ];
  
  // HANDLE SAVE
  function handleSave() {
    // VALIDATE REQUIRED FIELDS
    if (!editingStage.name) {
      alert('Stage name is required');
      return;
    }
    onsave(editingStage);
  }
  
  // Initialize the editing stage safely
  $effect(() => {
    if (isOpen && stage) {
      // Create a deep copy to avoid direct modification
      try {
        editingStage = JSON.parse(JSON.stringify(stage));
      } catch (e) {
        console.error("Error cloning stage:", e);
        editingStage = structuredClone(stage) || { ...stage };
      }
      
      // Ensure all required fields exist
      if (!editingStage.name) editingStage.name = "";
      if (!editingStage.description) editingStage.description = "";
      
      // Ensure config exists
      if (!editingStage.config) {
        editingStage.config = {};
      }
      
      // Ensure condition exists
      if (!editingStage.condition) {
        editingStage.condition = { type: "always", config: {} };
      }
      
      // Ensure parallelism config exists
      if (!editingStage.parallelism) {
        editingStage.parallelism = { mode: "sequential", maxWorkers: 1 };
      } else {
        // Ensure parallelism has all required fields
        if (!editingStage.parallelism.mode) {
          editingStage.parallelism.mode = "sequential";
        }
        if (typeof editingStage.parallelism.maxWorkers !== 'number') {
          editingStage.parallelism.maxWorkers = 1;
        }
      }
      
      // Ensure tasks array exists
      if (!Array.isArray(editingStage.tasks)) {
        editingStage.tasks = [];
      }
    }
  });
</script>

<Modal 
  title={`Configure Stage: ${editingStage.name || 'Unnamed Stage'}`}
  isOpen={isOpen}
  onclose={onclose}
  primaryAction="Save Stage"
  primaryVariant="primary"
  secondaryAction="Cancel"
  onprimaryAction={handleSave}
  onsecondaryAction={onclose}
>
  <div class="mb-4">
    <div class="tabs tabs-boxed mb-4">
      <button 
        class={`tab ${selectedTab === 'basic' ? 'tab-active' : ''}`}
        onclick={() => selectedTab = 'basic'}
      >
        <Settings class="h-4 w-4 mr-1" />
        Basic Settings
      </button>
      <button 
        class={`tab ${selectedTab === 'parallelism' ? 'tab-active' : ''}`}
        onclick={() => selectedTab = 'parallelism'}
      >
        <ArrowLeftRight class="h-4 w-4 mr-1" />
        Parallelism
      </button>
      <button 
        class={`tab ${selectedTab === 'condition' ? 'tab-active' : ''}`}
        onclick={() => selectedTab = 'condition'}
      >
        <Filter class="h-4 w-4 mr-1" />
        Execution Condition
      </button>
      <button 
        class={`tab ${selectedTab === 'advanced' ? 'tab-active' : ''}`}
        onclick={() => selectedTab = 'advanced'}
      >
        <Layers class="h-4 w-4 mr-1" />
        Advanced
      </button>
    </div>

    {#if selectedTab === 'basic'}
      <div class="space-y-4">
        <div>
          <label for="stage-name" class="label">
            <span class="label-text">Stage Name <span class="text-error">*</span></span>
          </label>
          <input
            id="stage-name"
            type="text"
            bind:value={editingStage.name}
            placeholder="Enter stage name"
            class="input input-bordered w-full"
            required
          />
        </div>
        <div>
          <label for="stage-description" class="label">
            <span class="label-text">Description</span>
          </label>
          <textarea
            id="stage-description"
            bind:value={editingStage.description}
            placeholder="Describe the purpose of this stage"
            rows="3"
            class="textarea textarea-bordered w-full"
          ></textarea>
        </div>
      </div>
    {/if}
    
    {#if selectedTab === 'parallelism'}
      <div class="space-y-4">
        <h3 class="font-medium mb-3">Task Execution Mode</h3>
        <div>
          <label for="parallelism-mode" class="label">
            <span class="label-text">Execution Mode</span>
          </label>
          <select
            id="parallelism-mode"
            bind:value={editingStage.parallelism.mode}
            class="select select-bordered w-full"
          >
            {#each parallelismModes as mode}
              <option value={mode.id}>{mode.name}</option>
            {/each}
          </select>
          <div class="label">
            <span class="label-text-alt">
              {parallelismModes.find(m => m.id === editingStage.parallelism.mode)?.description}
            </span>
          </div>
        </div>
        
        {#if editingStage.parallelism && editingStage.parallelism.mode !== 'sequential'}
          <div>
            <label for="max-workers" class="label">
              <span class="label-text">Maximum Workers</span>
            </label>
            <input
              id="max-workers"
              type="number"
              min="1"
              max="20"
              bind:value={editingStage.parallelism.maxWorkers}
              class="input input-bordered w-full"
            />
            <div class="label">
              <span class="label-text-alt">Maximum number of tasks that can run in parallel.</span>
            </div>
          </div>
        {/if}
      </div>
    {/if}
    
    {#if selectedTab === 'condition'}
      <div class="space-y-4">
        <h3 class="font-medium mb-3">Stage Execution Condition</h3>
        <ConditionBuilder bind:condition={editingStage.condition} />
      </div>
    {/if}
    
    {#if selectedTab === 'advanced'}
      <div class="space-y-4">
        <h3 class="font-medium mb-3">Advanced Settings</h3>
        <div>
          <label for="stage-id" class="label">
            <span class="label-text">Stage ID</span>
          </label>
          <input
            id="stage-id"
            type="text"
            value={editingStage.id}
            disabled
            class="input input-bordered w-full opacity-70"
          />
          <div class="label">
            <span class="label-text-alt">Unique identifier for this stage (auto-generated).</span>
          </div>
        </div>
        <div>
          <label for="stage-config" class="label">
            <span class="label-text">Additional Configuration (JSON)</span>
          </label>
          <textarea
            id="stage-config"
            value={typeof editingStage.config === 'object' 
              ? JSON.stringify(editingStage.config, null, 2) 
              : editingStage.config}
            rows="6"
            class="textarea textarea-bordered w-full font-mono text-sm"
          ></textarea>
          <div class="label">
            <span class="label-text-alt">Advanced configuration in JSON format (for expert users).</span>
          </div>
        </div>
      </div>
    {/if}
  </div>
</Modal>