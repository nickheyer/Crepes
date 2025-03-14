<script>
  import { Filter, Code } from 'lucide-svelte';
  
  // PROPS - CONDITION OBJECT WITH TYPE AND CONFIG
  let {
    condition = { type: "always", config: {} }
  } = $props();
  
  // CONDITION TYPES
  const conditionTypes = [
    { id: "always", name: "Always Execute", description: "Task will always execute" },
    { id: "never", name: "Never Execute", description: "Task will never execute (disabled)" },
    { id: "javascript", name: "JavaScript Expression", description: "Custom JavaScript expression" },
    { id: "comparison", name: "Value Comparison", description: "Compare values or variables" }
  ];
  
  // Ensure the condition object is properly structured
  $effect(() => {
    // Create a default condition if none exists
    if (!condition) {
      condition = { type: "always", config: {} };
    }
    
    // Make sure there's a config object
    if (!condition.config) {
      condition.config = {};
    }
    
    // Initialize config based on condition type
    if (condition.type === "javascript" && !condition.config.script) {
      condition.config.script = "";
    } else if (condition.type === "comparison") {
      if (!condition.config.left) condition.config.left = "";
      if (!condition.config.operator) condition.config.operator = "eq";
      if (!condition.config.right) condition.config.right = "";
    }
  });
</script>

<div class="bg-base-200 p-4 rounded-lg">
  <div class="mb-4">
    <label for="condition-type" class="block text-sm font-medium mb-1">
      Condition Type
    </label>
    <select
      id="condition-type"
      bind:value={condition.type}
      class="select select-bordered w-full"
    >
      {#each conditionTypes as type}
        <option value={type.id}>{type.name}</option>
      {/each}
    </select>
  </div>
  
  <!-- CONDITION TYPE DESCRIPTION -->
  <div class="mb-4 p-2 bg-base-300 rounded-md text-sm">
    <Filter class="h-4 w-4 inline mr-1" />
    {conditionTypes.find(t => t.id === condition.type)?.description || "Configure execution condition"}
  </div>
  
  <!-- CONDITION CONFIG BASED ON TYPE -->
  {#if condition.type === "javascript"}
    <div class="mb-4">
      <label for="js-condition" class="text-sm font-medium mb-1 flex items-center">
        <Code class="h-4 w-4 mr-1" />
        JavaScript Expression
      </label>
      <textarea
        id="js-condition"
        bind:value={condition.config.script}
        rows="4"
        placeholder="return true; // or any expression that evaluates to true/false"
        class="textarea textarea-bordered w-full font-mono text-sm"
      ></textarea>
      <p class="text-xs mt-1">
        Enter a JavaScript expression that evaluates to true or false.
        You can reference task outputs using <code class="bg-base-300 px-1 py-0.5 rounded">inputs.taskId</code>.
      </p>
    </div>
  {:else if condition.type === "comparison"}
    <div class="space-y-3">
      <div>
        <label for="left-value" class="block text-sm font-medium mb-1">
          Left Value
        </label>
        <input
          id="left-value"
          type="text"
          bind:value={condition.config.left}
          placeholder="Enter value or reference (e.g., inputs.someTask)"
          class="input input-bordered w-full"
        />
      </div>
      <div>
        <label for="operator" class="block text-sm font-medium mb-1">
          Operator
        </label>
        <select
          id="operator"
          bind:value={condition.config.operator}
          class="select select-bordered w-full"
        >
          <option value="eq">Equal to {`(==)`}</option>
          <option value="neq">Not equal to {`(!=)`}</option>
          <option value="gt">Greater than {`(>)`}</option>
          <option value="gte">Greater than or equal to {`(>=)`}</option>
          <option value="lt">Less than {`(<)`}</option>
          <option value="lte">Less than or equal to {`(<=)`}</option>
          <option value="contains">Contains</option>
          <option value="startsWith">Starts with</option>
          <option value="endsWith">Ends with</option>
          <option value="isEmpty">Is empty</option>
          <option value="isNotEmpty">Is not empty</option>
        </select>
      </div>
      {#if !["isEmpty", "isNotEmpty"].includes(condition.config.operator)}
        <div>
          <label for="right-value" class="block text-sm font-medium mb-1">
            Right Value
          </label>
          <input
            id="right-value"
            type="text"
            bind:value={condition.config.right}
            placeholder="Enter value or reference (e.g., inputs.someTask)"
            class="input input-bordered w-full"
          />
        </div>
      {/if}
      <div class="mt-3 p-2 bg-base-300 rounded-md text-xs font-mono">
        {#if ["isEmpty", "isNotEmpty"].includes(condition.config.operator)}
          {condition.config.left} {condition.config.operator === "isEmpty" ? "is empty" : "is not empty"}
        {:else}
          {condition.config.left || "[value]"} {condition.config.operator || "=="} {condition.config.right || "[value]"}
        {/if}
      </div>
    </div>
  {/if}
</div>