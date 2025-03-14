<script>
  import { ArrowLeftRight, Cpu, Workflow, AlertTriangle } from 'lucide-svelte';
  
  let {
      mode = "parallel",
      maxWorkers = 3,
      tasks = [],
      connectorMap = {}
  } = $props();
  
  // LOCAL STATE
  let allocatedTasks = $state([]);
  
  // MODEL TASK ALLOCATION WHEN PROPS CHANGE
  $effect(() => {
    // ENSURE TASKS IS ALWAYS AN ARRAY
    const tasksArray = Array.isArray(tasks) ? tasks : [];
    
    if (mode === 'parallel') {
      modelParallelExecution(tasksArray, maxWorkers);
    } else if (mode === 'worker-per-item') {
      modelWorkerPerItemExecution(tasksArray, maxWorkers);
    }
  });
  
  // MODEL PARALLEL EXECUTION
  function modelParallelExecution(tasks, maxWorkers) {
    // SIMPLE ALLOCATION - TASKS RUN IN PARALLEL UP TO MAX WORKERS
    // IN REALITY, THIS WOULD BE MORE COMPLEX WITH DEPENDENCIES
    const workerCount = Math.min(tasks.length, maxWorkers);
    const workers = Array(workerCount).fill().map(() => []);
    
    // ALLOCATE TASKS TO WORKERS USING ROUND-ROBIN
    tasks.forEach((task, index) => {
      if (workers.length > 0) {
        const workerIndex = index % workerCount;
        workers[workerIndex].push(task);
      }
    });
    
    allocatedTasks = workers;
  }
  
  // MODEL WORKER-PER-ITEM EXECUTION
  function modelWorkerPerItemExecution(tasks, maxWorkers) {
    // THIS IS A SIMPLIFIED MODEL - IN REALITY WOULD BE MORE COMPLEX
    // FINDS A TASK THAT OUTPUTS AN ARRAY AND MODELS WORKERS PROCESSING ITEMS
    
    // ENSURE TASKS IS ALWAYS AN ARRAY
    if (!Array.isArray(tasks) || tasks.length === 0) {
      allocatedTasks = [];
      return;
    }
    
    // FIND COLLECTION-PROCESSING TASK (SIMPLIFICATION)
    const collectionTask = tasks.find(t => 
      t && (
        t.type === 'extractLinks' || 
        t.type === 'extractImages' || 
        t.type === 'loop'
      )
    );
    
    if (!collectionTask) {
      // NO COLLECTION TASK FOUND, USE NORMAL PARALLEL MODEL
      modelParallelExecution(tasks, maxWorkers);
      return;
    }
    
    // MODEL WORKERS PROCESSING COLLECTION ITEMS
    const workers = Array(maxWorkers).fill().map(() => []);
    
    // ADD THE COLLECTION TASK TO ALL WORKERS (THEY ALL PROCESS DIFFERENT ITEMS)
    workers.forEach(worker => {
      worker.push(collectionTask);
    });
    
    // ADD REMAINING TASKS TO FIRST WORKER (SIMPLIFICATION)
    const remainingTasks = tasks.filter(t => t && t.id !== collectionTask.id);
    if (remainingTasks.length > 0 && workers.length > 0) {
      workers[0].push(...remainingTasks);
    }
    
    allocatedTasks = workers;
  }
  
  // GET DEPENDENCIES FOR A TASK
  function getDependencies(taskId) {
    if (!taskId) return [];
    
    const dependencies = [];
    
    // FIND ALL CONNECTIONS WHERE TASK IS TARGET
    Object.values(connectorMap || {}).forEach(connection => {
      if (connection && connection.target && connection.target.taskId === taskId) {
        dependencies.push(connection.source.taskId);
      }
    });
    
    return dependencies;
  }
  
  // CHECK IF ALLOCATION HAS DEPENDENCY ISSUES
  function hasAllocationIssues() {
    // SIMPLIFICATION - JUST CHECK IF ANY WORKER HAS BOTH A TASK AND ITS DEPENDENCY
    for (const worker of allocatedTasks) {
      const workerTaskIds = worker.map(t => t && t.id).filter(Boolean);
      
      for (const task of worker) {
        if (!task || !task.id) continue;
        
        const dependencies = getDependencies(task.id);
        
        // IF ANY DEPENDENCY IS ALSO IN THIS WORKER, WE HAVE AN ISSUE
        if (dependencies.some(depId => workerTaskIds.includes(depId))) {
          return true;
        }
      }
    }
    
    return false;
  }
</script>

<div class="worker-allocation mt-4 mb-2">
  <div class="flex items-center mb-2">
    <h3 class="text-sm font-medium flex items-center">
      <ArrowLeftRight class="h-4 w-4 mr-1" />
      Parallel Execution Preview
    </h3>
    {#if hasAllocationIssues()}
      <div class="ml-3 text-xs flex items-center text-amber-400">
        <AlertTriangle class="h-3 w-3 mr-1" />
        Potential dependency issues
      </div>
    {/if}
  </div>

  <div class="bg-base-900 p-3 rounded-lg">
    <div class="text-xs text-dark-400 mb-2">
      {#if mode === 'parallel'}
        Tasks will execute in parallel using {allocatedTasks.length} workers.
      {:else}
        Items from collection will be processed using {allocatedTasks.length} parallel workers.
      {/if}
    </div>
    
    <div class="grid grid-cols-1 gap-3">
      {#each allocatedTasks as workerTasks, workerIndex}
        <div class="worker p-2 bg-base-800 rounded-md">
          <div class="worker-header flex items-center mb-2">
            <Cpu class="h-3 w-3 mr-1 text-primary-400" />
            <span class="text-xs font-medium">Worker {workerIndex + 1}</span>
          </div>
          
          <div class="worker-tasks space-y-1">
            {#each workerTasks as task}
              {#if task}
                <div class="flex items-center rounded p-1 bg-base-700 text-xs">
                  <Workflow class="h-3 w-3 mr-1 {
                    task.type === 'extractLinks' || task.type === 'extractImages' || task.type === 'loop'
                      ? 'text-amber-400'
                      : 'text-dark-400'
                  }" />
                  <span>{task.name || "Unnamed Task"}</span>
                  {#if mode === 'worker-per-item' && (
                    task.type === 'extractLinks' || task.type === 'extractImages' || task.type === 'loop'
                  )}
                    <span class="ml-auto text-amber-400">Collection</span>
                  {/if}
                </div>
              {/if}
            {/each}
            
            {#if !workerTasks || workerTasks.length === 0}
              <div class="text-center text-xs text-dark-500 p-1">
                No tasks assigned
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  </div>
</div>