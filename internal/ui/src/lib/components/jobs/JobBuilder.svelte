<script>
  import { onMount } from 'svelte';
  import {
    DndContext,
    DragOverlay,
    closestCenter,
    PointerSensor,
    useSensors,
    useSensor
  } from '@dnd-kit-svelte/core';
  import { restrictToWindowEdges } from '@dnd-kit-svelte/modifiers';
  import {
    SortableContext,
    arrayMove,
    verticalListSortingStrategy,
    useSortable
  } from '@dnd-kit-svelte/sortable';
  import { state as jobState } from "$lib/stores/jobStore.svelte.js";

  // UI COMPONENTS
  import Button from "$lib/components/common/Button.svelte";
  import Card from "$lib/components/common/Card.svelte";
  import Modal from "$lib/components/common/Modal.svelte";
  import Tabs from "$lib/components/common/Tabs.svelte";
  import ConditionBuilder from "./ConditionBuilder.svelte";
  import StageConfig from "./StageConfig.svelte";
  import TaskConfig from "./TaskConfig.svelte";
  import ResourceBadge from "./ResourceBadge.svelte";
  import WorkerAllocation from "./WorkerAllocation.svelte";
  import {
    Blocks,
    ArrowLeftRight,
    ArrowDownUp,
    ChevronDown,
    ChevronRight,
    Grip,
    Plus,
    Trash,
    Settings,
    Play,
    XCircle,
    Copy,
    Save,
    CodeIcon,
    Cloud,
    Zap,
    Workflow,
    LucideWorkflow,
    Layers,
    Database,
    RefreshCw,
    FileCode,
    Upload,
    Wand2,
    Clock,
    Filter,
    Search,
    Eye,
    Download,
    Loader2
  } from 'lucide-svelte';
  // TASK REGISTRY - ORGANIZED BY CATEGORY
  const taskCategories = [
    {
      id: "browser",
      name: "Browser",
      icon: Cloud,
      description: "Control browser instances and pages",
      tasks: [
        { id: "createBrowser", name: "Create Browser", icon: Cloud, description: "Initialize a new browser instance" },
        { id: "createPage", name: "Create Page", icon: FileCode, description: "Open a new page in a browser" },
        { id: "disposeBrowser", name: "Close Browser", icon: XCircle, description: "Close and clean up a browser instance" },
        { id: "disposePage", name: "Close Page", icon: XCircle, description: "Close a browser page" }
      ]
    },
    {
      id: "navigation",
      name: "Navigation",
      icon: Workflow,
      description: "Navigate and interact with web pages",
      tasks: [
        { id: "navigate", name: "Navigate", icon: Workflow, description: "Navigate to a URL" },
        { id: "back", name: "Back", icon: ArrowLeftRight, description: "Go back in browser history" },
        { id: "forward", name: "Forward", icon: ArrowLeftRight, description: "Go forward in browser history" },
        { id: "reload", name: "Reload", icon: RefreshCw, description: "Reload the current page" },
        { id: "waitForLoad", name: "Wait For Load", icon: Clock, description: "Wait for page to finish loading" },
        { id: "takeScreenshot", name: "Take Screenshot", icon: Eye, description: "Capture screenshot of page or element" },
        { id: "executeScript", name: "Execute Script", icon: CodeIcon, description: "Run JavaScript on the page" }
      ]
    },
    {
      id: "interaction",
      name: "Interaction",
      icon: LucideWorkflow,
      description: "Interact with page elements",
      tasks: [
        { id: "click", name: "Click", icon: LucideWorkflow, description: "Click on an element" },
        { id: "type", name: "Type", icon: LucideWorkflow, description: "Enter text into a field" },
        { id: "select", name: "Select Dropdown", icon: LucideWorkflow, description: "Select option(s) from a dropdown" },
        { id: "hover", name: "Hover", icon: LucideWorkflow, description: "Hover over an element" },
        { id: "scroll", name: "Scroll", icon: LucideWorkflow, description: "Scroll the page or an element" }
      ]
    },
    {
      id: "extraction",
      name: "Extraction",
      icon: Database,
      description: "Extract data from pages",
      tasks: [
        { id: "extractText", name: "Extract Text", icon: Database, description: "Extract text content from element(s)" },
        { id: "extractAttribute", name: "Extract Attribute", icon: Database, description: "Extract attribute value from element(s)" },
        { id: "extractLinks", name: "Extract Links", icon: Database, description: "Extract all links from a page" },
        { id: "extractImages", name: "Extract Images", icon: Database, description: "Extract all images from a page" }
      ]
    },
    {
      id: "assets",
      name: "Assets",
      icon: Download,
      description: "Download and save assets",
      tasks: [
        { id: "downloadAsset", name: "Download Asset", icon: Download, description: "Download a file from a URL" },
        { id: "saveAsset", name: "Save Asset", icon: Save, description: "Save downloaded asset to the database" }
      ]
    },
    {
      id: "flow",
      name: "Flow Control",
      icon: Layers,
      description: "Control flow of execution",
      tasks: [
        { id: "conditional", name: "Conditional", icon: Filter, description: "Execute tasks based on a condition" },
        { id: "loop", name: "Loop", icon: RefreshCw, description: "Loop over a collection of items" },
        { id: "wait", name: "Wait", icon: Clock, description: "Pause execution for a specified time" }
      ]
    },
    {
      id: "transformation",
      name: "Transformation",
      icon: Wand2,
      description: "Transform and process data",
      tasks: [
        { id: "mapItems", name: "Map Items", icon: Wand2, description: "Transform each item in a collection" },
        { id: "filterItems", name: "Filter Items", icon: Filter, description: "Filter items in a collection" },
        { id: "sortItems", name: "Sort Items", icon: Layers, description: "Sort items in a collection" },
        { id: "mergeData", name: "Merge Data", icon: Layers, description: "Combine multiple data sources" },
        { id: "formatData", name: "Format Data", icon: Wand2, description: "Format or restructure data" }
      ]
    }
  ];
  // LOCAL STATE
  let pipeline = $state([]);
  let selectedStage = $state(null);
  let selectedTask = $state(null);
  let activeItem = $state(null);
  let expandedStages = $state({});
  let expandedTasks = $state({});
  let showTaskLibrary = $state(false);
  let newStageModalOpen = $state(false);
  let stageConfigModalOpen = $state(false);
  let taskConfigModalOpen = $state(false);
  let jobConfigModalOpen = $state(false);
  let viewPipelineModalOpen = $state(false);
  let isDraggingStage = $state(false);
  let isDraggingTask = $state(false);
  let taskSearchQuery = $state('');
  let activeTaskCategory = $state('all');
  let connectorMap = $state({});  // For visualizing task connections
  // JOB CONFIGURATION
  let jobConfig = $state({
    browserSettings: {
      headless: true,
      userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      viewportWidth: 1280,
      viewportHeight: 800,
      locale: "en-US",
      timezone: "UTC",
      defaultTimeout: 30000,
      recordVideo: false
    },
    scraperSettings: {
      maxDepth: 3,
      maxPages: 100,
      maxAssets: 1000,
      maxConcurrentRequests: 5,
      defaultNavigationMode: "domcontentloaded",
      followRedirects: true,
      sameDomainOnly: true,
      includeSubdomains: true
    },
    rateLimiting: {
      enabled: true,
      requestDelay: 1000,
      randomizeDelay: true,
      delayVariation: 0.3
    },
    resourceSettings: {
      maxBrowsers: 2,
      maxPages: 5,
      maxWorkers: 5
    }
  });
  // STAGE TEMPLATES
  let newStage = $state({
    id: "",
    name: "",
    description: "",
    condition: { type: "always", config: {} },
    parallelism: { mode: "sequential", maxWorkers: 1 },
    tasks: [],
    config: {}
  });
  let editingStage = $state(null);
  let editingTask = $state(null);
  // DND-KIT SETUP
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8, // 8px movement required before drag starts
      },
    })
  );
  onMount(() => {
    // INITIALIZE WITH DEFAULT PIPELINE OR LOAD FROM JOB
    if (jobState.formData?.data?.pipeline) {
      try {
        const parsedPipeline = JSON.parse(jobState.formData.data.pipeline);
        pipeline = parsedPipeline;
        // EXPAND ALL STAGES BY DEFAULT
        pipeline.forEach(stage => {
          expandedStages[stage.id] = true;
        });
      } catch (error) {
        console.error("Failed to parse pipeline:", error);
        initializeDefaultPipeline();
      }
    } else {
      initializeDefaultPipeline();
    }
    // INITIALIZE JOB CONFIG
    if (jobState.formData?.data?.jobConfig) {
      try {
        jobConfig = JSON.parse(jobState.formData.data.jobConfig);
      } catch (error) {
        console.error("Failed to parse job config:", error);
      }
    }
    // BUILD CONNECTION MAP FOR VISUALIZING TASK DEPENDENCIES
    buildConnectionMap();
  });
  // INITIALIZE A DEFAULT PIPELINE WITH A BASIC STRUCTURE
  function initializeDefaultPipeline() {
    pipeline = [
      {
        id: generateId("stage"),
        name: "Initialize",
        description: "Setup browser and initial page",
        condition: { type: "always", config: {} },
        parallelism: { mode: "sequential", maxWorkers: 1 },
        tasks: [],
        config: {}
      },
      {
        id: generateId("stage"),
        name: "Extract Content",
        description: "Extract and process data from pages",
        condition: { type: "always", config: {} },
        parallelism: { mode: "sequential", maxWorkers: 1 },
        tasks: [],
        config: {}
      },
      {
        id: generateId("stage"),
        name: "Cleanup",
        description: "Close resources and finalize",
        condition: { type: "always", config: {} },
        parallelism: { mode: "sequential", maxWorkers: 1 },
        tasks: [],
        config: {}
      }
    ];
    // EXPAND ALL STAGES BY DEFAULT
    pipeline.forEach(stage => {
      expandedStages[stage.id] = true;
    });
  }
  // SAVE PIPELINE TO JOB
  function savePipelineToJob() {
    // CONVERT PIPELINE TO JSON STRING
    jobState.formData.data.pipeline = JSON.stringify(pipeline);
    // SAVE CONFIG AS WELL
    jobState.formData.data.jobConfig = JSON.stringify(jobConfig);
    
    // PREPARE RESPONSE FOR CUSTOM EVENT
    const saveData = {
      pipeline,
      jobConfig
    };
    
    // DISPATCH CUSTOM EVENT
    const event = new CustomEvent('save', {
      detail: saveData,
      bubbles: true
    });
    
    document.dispatchEvent(event);
  }
  // CREATE NEW STAGE
  function createNewStage() {
    newStage = {
      id: generateId("stage"),
      name: "",
      description: "",
      condition: { type: "always", config: {} },
      parallelism: { mode: "sequential", maxWorkers: 1 },
      tasks: [],
      config: {}
    };
    newStageModalOpen = true;
  }
  // ADD NEW STAGE
  function addNewStage() {
    if (!newStage.name) return;
    // ADD STAGE TO PIPELINE
    pipeline = [...pipeline, newStage];
    // EXPAND NEW STAGE
    expandedStages[newStage.id] = true;
    // RESET AND CLOSE
    newStage = {
      id: generateId("stage"),
      name: "",
      description: "",
      condition: { type: "always", config: {} },
      parallelism: { mode: "sequential", maxWorkers: 1 },
      tasks: [],
      config: {}
    };
    newStageModalOpen = false;
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // DUPLICATE STAGE
  function duplicateStage(stageToDuplicate) {
    const newStageCopy = JSON.parse(JSON.stringify(stageToDuplicate));
    newStageCopy.id = generateId("stage");
    newStageCopy.name = `${stageToDuplicate.name} (copy)`;
    // GENERATE NEW IDs FOR ALL TASKS
    newStageCopy.tasks = newStageCopy.tasks.map(task => ({
      ...task,
      id: generateId("task")
    }));
    // ADD STAGE TO PIPELINE
    pipeline = [...pipeline, newStageCopy];
    // EXPAND NEW STAGE
    expandedStages[newStageCopy.id] = true;
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // EDIT STAGE
  function editStage(stage) {
    // CREATE A DEEP COPY TO AVOID DIRECT MODIFICATION
    editingStage = JSON.parse(JSON.stringify(stage));
    stageConfigModalOpen = true;
  }
  // UPDATE STAGE
  function updateStage() {
    if (!editingStage || !editingStage.id) return;
    // UPDATE STAGE IN PIPELINE
    pipeline = pipeline.map(stage => 
      stage.id === editingStage.id ? editingStage : stage
    );
    // RESET AND CLOSE
    editingStage = null;
    stageConfigModalOpen = false;
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // DELETE STAGE
  function deleteStage(stageId) {
    if (!confirm("Are you sure you want to delete this stage?")) return;
    pipeline = pipeline.filter(stage => stage.id !== stageId);
    // CLEAN UP EXPANDED STATE
    delete expandedStages[stageId];
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // EDIT TASK
  function editTask(stageId, task) {
    const stage = pipeline.find(s => s.id === stageId);
    if (!stage) return;
    
    // CREATE A DEEP COPY TO AVOID DIRECT MODIFICATION
    const taskCopy = JSON.parse(JSON.stringify(task));
    
    // ENSURE TASK HAS RETRY CONFIG
    if (!taskCopy.retryConfig) {
      taskCopy.retryConfig = {
        maxRetries: 3,
        delayMS: 1000,
        backoffRate: 1.5
      };
    }
    
    // ENSURE TASK HAS CONDITION
    if (!taskCopy.condition) {
      taskCopy.condition = { 
        type: "always", 
        config: {} 
      };
    }
    
    // ENSURE TASK HAS CONFIG
    if (!taskCopy.config) {
      taskCopy.config = {};
    }
    
    // ENSURE TASK HAS INPUT REFS ARRAY
    if (!taskCopy.inputRefs) {
      taskCopy.inputRefs = [];
    }
    
    // ENSURE TASK HAS OUTPUT REF
    if (!taskCopy.outputRef) {
      taskCopy.outputRef = `output_${generateId("")}`;
    }
    
    editingTask = taskCopy;
    selectedStage = stageId;
    taskConfigModalOpen = true;
  }
  // UPDATE TASK
  function updateTask() {
    if (!editingTask || !editingTask.id || !selectedStage) return;
    // UPDATE TASK IN PIPELINE
    pipeline = pipeline.map(stage => {
      if (stage.id === selectedStage) {
        return {
          ...stage,
          tasks: stage.tasks.map(task => 
            task.id === editingTask.id ? editingTask : task
          )
        };
      }
      return stage;
    });
    // RESET AND CLOSE
    editingTask = null;
    selectedStage = null;
    taskConfigModalOpen = false;
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // DUPLICATE TASK
  function duplicateTask(stageId, taskToDuplicate) {
    const stage = pipeline.find(s => s.id === stageId);
    if (!stage) return;
    const newTaskCopy = JSON.parse(JSON.stringify(taskToDuplicate));
    newTaskCopy.id = generateId("task");
    newTaskCopy.name = `${taskToDuplicate.name} (copy)`;
    // ADD TASK TO STAGE
    pipeline = pipeline.map(s => {
      if (s.id === stageId) {
        return {
          ...s,
          tasks: [...s.tasks, newTaskCopy]
        };
      }
      return s;
    });
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // DELETE TASK
  function deleteTask(stageId, taskId) {
    if (!confirm("Are you sure you want to delete this task?")) return;
    pipeline = pipeline.map(stage => {
      if (stage.id === stageId) {
        return {
          ...stage,
          tasks: stage.tasks.filter(task => task.id !== taskId)
        };
      }
      return stage;
    });
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // TOGGLE STAGE EXPANSION
  function toggleStageExpand(stageId) {
    expandedStages[stageId] = !expandedStages[stageId];
  }
  // TOGGLE TASK EXPANSION
  function toggleTaskExpand(taskId) {
    expandedTasks[taskId] = !expandedTasks[taskId];
  }
  // ADD TASK FROM LIBRARY
  function addTaskFromLibrary(stageId, taskType) {
    // FIND TASK DEFINITION
    let taskDef = null;
    for (const category of taskCategories) {
      const task = category.tasks.find(t => t.id === taskType);
      if (task) {
        taskDef = task;
        break;
      }
    }
    if (!taskDef) return;
    // CREATE NEW TASK
    const newTask = {
      id: generateId("task"),
      name: taskDef.name,
      type: taskDef.id,
      description: taskDef.description || "",
      config: getDefaultConfigForTaskType(taskDef.id),
      inputRefs: [],
      outputRef: `output_${generateId("")}`,
      condition: { type: "always", config: {} },
      retryConfig: { maxRetries: 3, delayMS: 1000, backoffRate: 1.5 }
    };
    // ADD TASK TO STAGE
    pipeline = pipeline.map(stage => {
      if (stage.id === stageId) {
        return {
          ...stage,
          tasks: [...stage.tasks, newTask]
        };
      }
      return stage;
    });
    showTaskLibrary = false;
    // EXPAND THE TASK BY DEFAULT
    expandedTasks[newTask.id] = true;
    // UPDATE CONNECTION MAP
    buildConnectionMap();
  }
  // GET DEFAULT CONFIG FOR TASK TYPE
  function getDefaultConfigForTaskType(taskType) {
    switch (taskType) {
      case "createBrowser":
        return {
          headless: true,
          userAgent: jobConfig.browserSettings.userAgent
        };
      case "createPage":
        return {
          browserId: "",
          viewportWidth: jobConfig.browserSettings.viewportWidth,
          viewportHeight: jobConfig.browserSettings.viewportHeight
        };
      case "navigate":
        return {
          pageId: "",
          url: "",
          waitUntil: "domcontentloaded",
          timeout: 30000
        };
      case "click":
        return {
          pageId: "",
          selector: "",
          button: "left",
          timeout: 10000
        };
      case "type":
        return {
          pageId: "",
          selector: "",
          text: "",
          delay: 50,
          clear: true
        };
      case "extractText":
        return {
          pageId: "",
          selector: "",
          multiple: false,
          trim: true
        };
      case "extractLinks":
        return {
          pageId: "",
          selector: "a",
          includeText: true,
          normalizeUrls: true
        };
      case "downloadAsset":
        return {
          url: "",
          folder: "downloads",
          timeout: 60000
        };
      case "conditional":
        return {
          condition: "",
          ifTrue: null,
          ifFalse: null
        };
      case "loop":
        return {
          items: [],
          parallelProcessing: false,
          maxWorkers: 3
        };
      case "wait":
        return {
          duration: 1000 // milliseconds
        };
      default:
        return {};
    }
  }
  // GENERATE A UNIQUE ID
  function generateId(prefix = '') {
    return `${prefix}_${Math.random().toString(36).substr(2, 9)}`;
  }
  // HANDLE REORDERING STAGES
  function handleReorderStages(oldIndex, newIndex) {
    pipeline = arrayMove(pipeline, oldIndex, newIndex);
    buildConnectionMap();
  }
  // HANDLE REORDERING TASKS
  function handleReorderTasks(stageId, oldIndex, newIndex) {
    pipeline = pipeline.map(stage => {
      if (stage.id === stageId) {
        return {
          ...stage,
          tasks: arrayMove(stage.tasks, oldIndex, newIndex)
        };
      }
      return stage;
    });
    buildConnectionMap();
  }
  // HANDLE DND START
  function handleDragStart(event) {
    const { active } = event;
    activeItem = active.id;
    // DETERMINE IF DRAGGING STAGE OR TASK
    isDraggingStage = active.id.startsWith('stage_');
    isDraggingTask = active.id.startsWith('task_');
  }
  // HANDLE DND END
  function handleDragEnd(event) {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      // IF DRAGGING STAGES
      if (active.id.startsWith('stage_') && over.id.startsWith('stage_')) {
        const oldIndex = pipeline.findIndex(stage => stage.id === active.id);
        const newIndex = pipeline.findIndex(stage => stage.id === over.id);
        if (oldIndex !== -1 && newIndex !== -1) {
          handleReorderStages(oldIndex, newIndex);
        }
      }
      // IF DRAGGING TASKS
      if (active.id.startsWith('task_') && over.id.startsWith('task_')) {
        // EXTRACT STAGE ID FROM DATA ATTRIBUTES
        const activeStageId = active.data.current.sortable.containerId;
        const overStageId = over.data.current.sortable.containerId;
        const getTaskIndex = (stageId, taskId) => {
          const stage = pipeline.find(s => s.id === stageId);
          return stage ? stage.tasks.findIndex(t => t.id === taskId) : -1;
        };
        if (activeStageId === overStageId) {
          // SAME STAGE REORDERING
          const oldIndex = getTaskIndex(activeStageId, active.id);
          const newIndex = getTaskIndex(overStageId, over.id);
          if (oldIndex !== -1 && newIndex !== -1) {
            handleReorderTasks(activeStageId, oldIndex, newIndex);
          }
        } else {
          // MOVING BETWEEN STAGES
          const activeStage = pipeline.find(s => s.id === activeStageId);
          const taskIndex = getTaskIndex(activeStageId, active.id);
          if (activeStage && taskIndex !== -1) {
            const task = { ...activeStage.tasks[taskIndex] };
            // REMOVE FROM ORIGINAL STAGE
            pipeline = pipeline.map(stage => {
              if (stage.id === activeStageId) {
                return {
                  ...stage,
                  tasks: stage.tasks.filter((_, i) => i !== taskIndex)
                };
              }
              return stage;
            });
            // ADD TO NEW STAGE
            const overTaskIndex = getTaskIndex(overStageId, over.id);
            pipeline = pipeline.map(stage => {
              if (stage.id === overStageId) {
                const newTasks = [...stage.tasks];
                newTasks.splice(overTaskIndex + 1, 0, task);
                return {
                  ...stage,
                  tasks: newTasks
                };
              }
              return stage;
            });
            // UPDATE CONNECTION MAP
            buildConnectionMap();
          }
        }
      }
    }
    // RESET DRAG STATE
    activeItem = null;
    isDraggingStage = false;
    isDraggingTask = false;
  }
  // OPEN JOB CONFIG MODAL
  function openJobConfig() {
    jobConfigModalOpen = true;
  }
  // SAVE JOB CONFIG
  function saveJobConfig() {
    // SAVE CONFIG TO JOB
    jobState.formData.data.jobConfig = JSON.stringify(jobConfig);
    jobConfigModalOpen = false;
  }
  // VIEW PIPELINE JSON
  function viewPipelineJSON() {
    viewPipelineModalOpen = true;
  }
  // SAVE ALL CHANGES
  function saveAllChanges() {
    savePipelineToJob();
    // SAVE JOB CONFIG
    jobState.formData.data.jobConfig = JSON.stringify(jobConfig);
    alert("Pipeline and configuration saved to job");
  }
  // BUILD CONNECTION MAP FOR VISUALIZING TASK DEPENDENCIES
  function buildConnectionMap() {
    connectorMap = {};
    // FOR EACH STAGE
    pipeline.forEach(stage => {
      // FOR EACH TASK
      stage.tasks.forEach(task => {
        // MAP INPUT REFERENCES TO SOURCE TASKS
        if (task.inputRefs && task.inputRefs.length > 0) {
          task.inputRefs.forEach(inputRef => {
            // FIND SOURCE TASK
            let sourceTask = null;
            let sourceStage = null;
            pipeline.forEach(s => {
              s.tasks.forEach(t => {
                if (t.outputRef === inputRef) {
                  sourceTask = t;
                  sourceStage = s;
                }
              });
            });
            if (sourceTask && sourceStage) {
              // CREATE CONNECTION
              const key = `${sourceTask.id}_to_${task.id}`;
              connectorMap[key] = {
                source: {
                  taskId: sourceTask.id,
                  stageId: sourceStage.id,
                  outputRef: sourceTask.outputRef
                },
                target: {
                  taskId: task.id,
                  stageId: stage.id,
                  inputRef: inputRef
                }
              };
            }
          });
        }
      });
    });
  }
  // FILTER TASKS BASED ON SEARCH AND CATEGORY
  $effect(() => {
    if (taskSearchQuery || activeTaskCategory !== 'all') {
      // RESET SEARCH WHEN CATEGORY CHANGES
      taskSearchQuery = '';
    }
  });
  function getFilteredTaskCategories() {
    if (!taskSearchQuery && activeTaskCategory === 'all') {
      return taskCategories;
    }
    return taskCategories
      .filter(category => {
        if (activeTaskCategory !== 'all' && category.id !== activeTaskCategory) {
          return false;
        }
        if (!taskSearchQuery) {
          return true;
        }
        // FILTER TASKS WITHIN CATEGORY
        const matchingTasks = category.tasks.filter(task => 
          task.name.toLowerCase().includes(taskSearchQuery.toLowerCase()) ||
          task.description.toLowerCase().includes(taskSearchQuery.toLowerCase())
        );
        return matchingTasks.length > 0;
      })
      .map(category => ({
        ...category,
        tasks: category.tasks.filter(task => 
          !taskSearchQuery || 
          task.name.toLowerCase().includes(taskSearchQuery.toLowerCase()) ||
          task.description.toLowerCase().includes(taskSearchQuery.toLowerCase())
        )
      }));
  }
  // GET TASK ICON COMPONENT
  function getTaskIconComponent(taskType) {
    for (const category of taskCategories) {
      const task = category.tasks.find(t => t.id === taskType);
      if (task && task.icon) {
        return task.icon;
      }
    }
    return null;
  }
  // GET TASK COLOR BASED ON CATEGORY
  function getTaskColorClass(taskType) {
    for (const category of taskCategories) {
      if (category.tasks.some(t => t.id === taskType)) {
        switch (category.id) {
          case 'browser': return 'bg-blue-700 text-white';
          case 'navigation': return 'bg-purple-700 text-white';
          case 'interaction': return 'bg-green-700 text-white';
          case 'extraction': return 'bg-amber-700 text-white';
          case 'assets': return 'bg-rose-700 text-white';
          case 'flow': return 'bg-cyan-700 text-white';
          case 'transformation': return 'bg-indigo-700 text-white';
          default: return 'bg-gray-700 text-white';
        }
      }
    }
    return 'bg-gray-700 text-white';
  }
  // CHECK IF A TASK DEPENDS ON ANOTHER TASK
  function getDependencies(taskId) {
    const dependencies = [];
    // FIND ALL CONNECTIONS WHERE TASK IS TARGET
    Object.values(connectorMap).forEach(connection => {
      if (connection.target && connection.target.taskId === taskId) {
        dependencies.push(connection.source.taskId);
      }
    });
    return dependencies;
  }
  // CHECK IF A TASK IS DEPENDED ON BY OTHER TASKS
  function getDependents(taskId) {
    const dependents = [];
    // FIND ALL CONNECTIONS WHERE TASK IS SOURCE
    Object.values(connectorMap).forEach(connection => {
      if (connection.source && connection.source.taskId === taskId) {
        dependents.push(connection.target.taskId);
      }
    });
    return dependents;
  }
  function findTaskInfo(id) {
      let taskInfo = null;
      pipeline.forEach((s) => {
          const task = s.tasks.find(t => t.id === id);
          if (task) {
              taskInfo = { task, stage: s };
          }
      });
      return taskInfo;
  }
</script>
<div class="job-builder">
  <div class="flex justify-between items-center mb-4">
    <h2 class="text-xl font-bold">Pipeline Builder</h2>
    <div class="flex gap-2">
      <Button variant="outline" size="sm" onclick={openJobConfig}>
        <Settings class="h-4 w-4 mr-1" />
        Job Settings
      </Button>
      <Button variant="outline" size="sm" onclick={viewPipelineJSON}>
        <CodeIcon class="h-4 w-4 mr-1" />
        View JSON
      </Button>
      <Button variant="primary" size="sm" onclick={saveAllChanges}>
        <Save class="h-4 w-4 mr-1" />
        Save Pipeline
      </Button>
    </div>
  </div>
  <div class="pipeline-container bg-base-800 rounded-lg border border-base-700 overflow-hidden">
    <div class="pipeline-header bg-base-700 p-3 flex justify-between items-center">
      <div class="flex items-center">
        <Blocks class="h-5 w-5 mr-2" />
        <span class="font-medium">Pipeline Stages</span>
      </div>
      <Button variant="primary" size="sm" onclick={createNewStage}>
        <Plus class="h-4 w-4 mr-1" />
        Add Stage
      </Button>
    </div>
    <!-- PIPELINE STAGES -->
    {#if pipeline.length === 0}
      <div class="p-8 text-center">
        <div class="bg-base-700 inline-flex rounded-full p-3 mb-3">
          <Blocks class="h-6 w-6 text-primary-400" />
        </div>
        <h3 class="text-lg font-medium mb-2">No Pipeline Stages</h3>
        <p class="text-dark-400 mb-4 max-w-md mx-auto">
          Start by adding a stage to your pipeline. Each stage can contain multiple tasks that run in sequence or parallel.
        </p>
        <Button variant="primary" onclick={createNewStage}>
          <Plus class="h-4 w-4 mr-1" />
          Add First Stage
        </Button>
      </div>
    {:else}
      <DndContext 
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
        modifiers={[restrictToWindowEdges]}
        collisionDetection={closestCenter}
        sensors={sensors}>
        <SortableContext items={pipeline.map(s => s.id)} strategy={verticalListSortingStrategy}>
          <div class="pipeline-stages">
            {#each pipeline as stage, stageIndex}
              <div class="stage-container {activeItem === stage.id ? 'border-primary-500' : ''} 
                          relative mb-2 rounded-lg border border-base-700 overflow-hidden">
                <!-- STAGE CONDITION INDICATOR -->
                {#if stage.condition && stage.condition.type !== "always"}
                  <div class="absolute top-3 right-32 px-2 py-1 text-xs bg-amber-800 rounded-full flex items-center gap-1">
                    <Filter class="h-3 w-3" />
                    {stage.condition.type === "never" 
                      ? "Never Run" 
                      : stage.condition.type === "javascript"
                        ? "JS Condition"
                        : "Conditional"}
                  </div>
                {/if}
                <!-- STAGE HEADER -->
                <div class="stage-header flex items-center p-3 bg-base-700 cursor-pointer relative">
                  <!-- ORDER NUMBER -->
                  <div class="absolute left-0 top-0 bottom-0 w-6 flex items-center justify-center bg-base-800 text-xs font-mono text-dark-300">
                    {stageIndex + 1}
                  </div>
                  <div class="stage-drag-handle mr-2 cursor-grab ml-4" data-dnd-handle>
                    <Grip class="h-4 w-4 text-dark-400" />
                  </div>
                  <button 
                    class="mr-2 text-dark-300 hover:text-dark-100 focus:outline-none" 
                    onclick={() => toggleStageExpand(stage.id)}
                  >
                    {#if expandedStages[stage.id]}
                      <ChevronDown class="h-4 w-4" />
                    {:else}
                      <ChevronRight class="h-4 w-4" />
                    {/if}
                  </button>
                  <div class="flex-1">
                    <h3 class="font-medium text-sm">{stage.name || "Unnamed Stage"}</h3>
                    {#if stage.description}
                      <p class="text-xs text-dark-400">{stage.description}</p>
                    {/if}
                  </div>
                  <!-- PARALLELISM MODE BADGE -->
                  <div class="stage-parallelism px-2 py-1 text-xs bg-base-800 rounded-full flex items-center mr-2">
                    {#if stage.parallelism && stage.parallelism.mode === "sequential"}
                      <ArrowDownUp class="h-3 w-3 mr-1" />
                      Sequential
                    {:else if stage.parallelism && stage.parallelism.mode === "parallel"}
                      <ArrowLeftRight class="h-3 w-3 mr-1" />
                      Parallel ({stage.parallelism.maxWorkers})
                    {:else}
                      <ArrowLeftRight class="h-3 w-3 mr-1" />
                      Worker-per-item ({stage.parallelism.maxWorkers})
                    {/if}
                  </div>
                  <!-- TASK COUNT BADGE -->
                  <div class="px-2 py-1 text-xs bg-base-800 rounded-full mr-2">
                    {stage.tasks ? stage.tasks.length : 0} tasks
                  </div>
                  <div class="stage-actions flex space-x-1">
                    <button 
                      class="p-1 text-dark-300 hover:text-primary-400 focus:outline-none" 
                      onclick={() => editStage(stage)}
                      title="Edit Stage"
                    >
                      <Settings class="h-4 w-4" />
                    </button>
                    <button 
                      class="p-1 text-dark-300 hover:text-primary-400 focus:outline-none" 
                      onclick={() => duplicateStage(stage)}
                      title="Duplicate Stage"
                    >
                      <Copy class="h-4 w-4" />
                    </button>
                    <button 
                      class="p-1 text-dark-300 hover:text-danger-400 focus:outline-none" 
                      onclick={() => deleteStage(stage.id)}
                      title="Delete Stage"
                    >
                      <Trash class="h-4 w-4" />
                    </button>
                  </div>
                </div>
                {#if expandedStages[stage.id]}
                  <div class="stage-content p-3 bg-base-800">
                    <!-- TASKS -->
                    {#if !stage.tasks || stage.tasks.length === 0}
                      <div class="empty-tasks p-4 text-center border border-dashed border-base-600 rounded-lg">
                        <p class="text-dark-400 mb-2">No tasks in this stage</p>
                        <Button 
                          variant="outline" 
                          size="sm" 
                          onclick={() => {
                            selectedStage = stage.id;
                            showTaskLibrary = true;
                          }}
                        >
                          <Plus class="h-4 w-4 mr-1" />
                          Add Task
                        </Button>
                      </div>
                    {:else}
                      <SortableContext 
                        items={stage.tasks.map(t => t.id)} 
                        strategy={verticalListSortingStrategy} 
                        id={stage.id}>
                        <div class="tasks-list space-y-2">
                          {#each stage.tasks as task, taskIndex}
                            {@const taskIconComponent = getTaskIconComponent(task.type)}
                            <div 
                              class="task-item flex flex-col p-2 bg-base-700 rounded-lg hover:bg-base-650
                                     {activeItem === task.id ? 'border-primary-500' : 'border-transparent'} border-2"
                              data-stage-id={stage.id}
                            >
                              <!-- TASK HEADER -->
                              <div class="flex items-center">
                                <!-- ORDER NUMBER -->
                                <div class="relative mr-2 h-6 w-6 flex items-center justify-center rounded bg-base-800 text-xs font-mono">
                                  {taskIndex + 1}
                                </div>
                                <div class="task-drag-handle mr-2 cursor-grab" data-dnd-handle>
                                  <Grip class="h-4 w-4 text-dark-400" />
                                </div>
                                <!-- TASK ICON AND TYPE BADGE -->
                                <div class="task-icon mr-2 p-1 rounded {getTaskColorClass(task.type)}">
                                  {#if taskIconComponent}
                                    <taskIconComponent class="h-3 w-3"></taskIconComponent>
                                  {:else}
                                    <Settings class="h-3 w-3" />
                                  {/if}
                                </div>
                                <div class="flex-1">
                                  <p class="font-medium text-sm">{task.name || "Unnamed Task"}</p>
                                  <p class="text-xs text-dark-400">
                                    {task.description}
                                  </p>
                                </div>
                                <!-- DEPENDENCY & OUTPUT BADGES -->
                                <div class="flex items-center gap-1 mr-2">
                                  {#if task.inputRefs?.length > 0}
                                    <div class="px-2 py-0.5 text-xs bg-blue-900 rounded-full flex items-center gap-1" title="Has input dependencies">
                                      <ArrowDownUp class="h-3 w-3" />
                                      In: {task.inputRefs.length}
                                    </div>
                                  {/if}
                                  {#if getDependents(task.id).length > 0}
                                    <div class="px-2 py-0.5 text-xs bg-green-900 rounded-full flex items-center gap-1" title="Output used by other tasks">
                                      <ArrowDownUp class="h-3 w-3 rotate-180" />
                                      Out: {getDependents(task.id).length}
                                    </div>
                                  {/if}
                                </div>
                                <button 
                                  class="p-1 mr-1 text-dark-300 hover:text-dark-100 focus:outline-none" 
                                  onclick={() => toggleTaskExpand(task.id)}
                                >
                                  {#if expandedTasks[task.id]}
                                    <ChevronDown class="h-4 w-4" />
                                  {:else}
                                    <ChevronRight class="h-4 w-4" />
                                  {/if}
                                </button>
                                <div class="task-actions flex space-x-1">
                                  <button 
                                    class="p-1 text-dark-300 hover:text-primary-400 focus:outline-none" 
                                    onclick={() => editTask(stage.id, task)}
                                    title="Edit Task"
                                  >
                                    <Settings class="h-3.5 w-3.5" />
                                  </button>
                                  <button 
                                    class="p-1 text-dark-300 hover:text-primary-400 focus:outline-none" 
                                    onclick={() => duplicateTask(stage.id, task)}
                                    title="Duplicate Task"
                                  >
                                    <Copy class="h-3.5 w-3.5" />
                                  </button>
                                  <button 
                                    class="p-1 text-dark-300 hover:text-danger-400 focus:outline-none" 
                                    onclick={() => deleteTask(stage.id, task.id)}
                                    title="Delete Task"
                                  >
                                    <Trash class="h-3.5 w-3.5" />
                                  </button>
                                </div>
                              </div>
                              <!-- TASK EXPANDED VIEW -->
                              {#if expandedTasks[task.id]}
                                <div class="task-details mt-2 pt-2 border-t border-base-600">
                                  <!-- INPUT DEPENDENCIES -->
                                  {#if task.inputRefs?.length > 0}
                                    <div class="mb-2">
                                      <h4 class="text-xs font-medium mb-1">Inputs:</h4>
                                      <div class="flex flex-wrap gap-1">
                                        {#each task.inputRefs as inputRef}
                                          <ResourceBadge 
                                            type="input" 
                                            resourceId={inputRef}
                                            connectorMap={connectorMap} 
                                          />
                                        {/each}
                                      </div>
                                    </div>
                                  {/if}
                                  <!-- OUTPUT -->
                                  {#if task.outputRef}
                                    <div class="mb-2">
                                      <h4 class="text-xs font-medium mb-1">Output:</h4>
                                      <ResourceBadge 
                                        type="output" 
                                        resourceId={task.outputRef}
                                        connectorMap={connectorMap} 
                                      />
                                    </div>
                                  {/if}
                                  <!-- CONFIG SUMMARY -->
                                  {#if task.config && Object.keys(task.config).length > 0}
                                    <div class="mb-2">
                                      <h4 class="text-xs font-medium mb-1">Configuration:</h4>
                                      <div class="text-xs bg-base-800 p-2 rounded max-h-32 overflow-y-auto">
                                        {#each Object.entries(task.config) as [key, value]}
                                          <div class="flex mb-1">
                                            <span class="text-primary-400 mr-1">{key}:</span>
                                            <span class="text-dark-300">
                                              {typeof value === 'object' ? JSON.stringify(value) : value}
                                            </span>
                                          </div>
                                        {/each}
                                      </div>
                                    </div>
                                  {/if}
                                  <!-- CONDITION -->
                                  {#if task.condition && task.condition.type !== "always"}
                                    <div class="mb-2">
                                      <h4 class="text-xs font-medium mb-1 flex items-center">
                                        <Filter class="h-3 w-3 mr-1" />
                                        Condition:
                                      </h4>
                                      <div class="text-xs px-2 py-1 bg-amber-900/30 rounded-md">
                                        {task.condition.type === "never" 
                                          ? "Never executed" 
                                          : task.condition.type === "javascript"
                                            ? "JavaScript condition"
                                            : "Comparison condition"}
                                      </div>
                                    </div>
                                  {/if}
                                  <!-- RETRY CONFIG -->
                                  {#if task.retryConfig && task.retryConfig.maxRetries > 0}
                                    <div>
                                      <h4 class="text-xs font-medium mb-1 flex items-center">
                                        <RefreshCw class="h-3 w-3 mr-1" />
                                        Retry:
                                      </h4>
                                      <div class="text-xs">
                                        Max: {task.retryConfig.maxRetries}, 
                                        Delay: {task.retryConfig.delayMS}ms, 
                                        Backoff: {task.retryConfig.backoffRate}x
                                      </div>
                                    </div>
                                  {/if}
                                </div>
                              {/if}
                            </div>
                          {/each}
                        </div>
                      </SortableContext>
                      <!-- PARALLEL EXECUTION VISUALIZATION -->
                      {#if stage.parallelism && stage.parallelism.mode !== "sequential" && stage.tasks.length > 0}
                        <WorkerAllocation 
                          mode={stage.parallelism.mode} 
                          maxWorkers={stage.parallelism.maxWorkers}
                          tasks={stage.tasks}
                          connectorMap={connectorMap}
                        />
                      {/if}
                    {/if}
                    <!-- ADD TASK BUTTON -->
                    <div class="mt-3 text-center">
                      <Button 
                        variant="outline" 
                        size="sm" 
                        onclick={() => {
                          selectedStage = stage.id;
                          showTaskLibrary = true;
                        }}
                      >
                        <Plus class="h-4 w-4 mr-1" />
                        Add Task
                      </Button>
                    </div>
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        </SortableContext>
        <!-- DRAG OVERLAY -->
        <DragOverlay adjustScale={true} dropAnimation={{ duration: 200, easing: 'cubic-bezier(0.18, 0.67, 0.6, 1.22)' }}>
          {#if activeItem && isDraggingStage}
            <div class="drag-preview bg-base-700 opacity-80 p-3 rounded-lg border-2 border-primary-500 shadow-lg w-full max-w-sm">
              <div class="flex items-center">
                <Blocks class="h-4 w-4 mr-2" />
                <span class="font-medium text-sm">
                  {pipeline.find(s => s.id === activeItem)?.name || "Stage"}
                </span>
              </div>
            </div>
          {/if}
          {#if activeItem && isDraggingTask}
            {@const draggedTask = (() => {
              let foundTask = null;
              pipeline.forEach(s => {
                const task = s.tasks.find(t => t.id === activeItem);
                if (task) foundTask = { task, stage: s };
              });
              return foundTask;
            })()}
            <div class="drag-preview bg-base-700 opacity-80 p-3 rounded-lg border-2 border-primary-500 shadow-lg w-full max-w-xs">
              {#if draggedTask}
                  {@const draggedIconComponent = getTaskIconComponent(draggedTask.task.type)}
                  <div class="flex items-center">
                  <div class="task-icon mr-2 p-1 rounded {getTaskColorClass(draggedTask.task.type)}">
                      {#if draggedIconComponent}
                        <draggedIconComponent class="h-3 w-3"></draggedIconComponent>
                      {:else}
                        <Settings class="h-3 w-3" />
                      {/if}
                  </div>
                  <span class="font-medium text-sm">{draggedTask.task.name}</span>
                  </div>
              {/if}
          </div>
        {/if}
        </DragOverlay>
      </DndContext>
    {/if}
  </div>
</div>
<!-- TASK LIBRARY MODAL -->
{#if showTaskLibrary}
  <Modal 
    title="Add Task" 
    size="lg"
    onclose={() => showTaskLibrary = false}
  >
    <div class="mb-4">
      <div class="flex mb-3">
        <div class="relative flex-1 mr-2">
          <Search class="h-4 w-4 absolute left-2 top-2.5 text-dark-400" />
          <input
            type="text"
            bind:value={taskSearchQuery}
            placeholder="Search tasks..."
            class="w-full pl-8 pr-4 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
          />
        </div>
        <select
          bind:value={activeTaskCategory}
          class="px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
        >
          <option value="all">All Categories</option>
          {#each taskCategories as category}
            <option value={category.id}>{category.name}</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="task-library-content max-h-[60vh] overflow-y-auto pr-2 space-y-4">
      {#each getFilteredTaskCategories() as category}
        <div class="task-category">
          <div class="flex items-center mb-2 pb-1 border-b border-base-600">
            <category.icon class="h-4 w-4 mr-2"></category.icon>
            <h3 class="font-medium">{category.name}</h3>
          </div>
          <p class="text-sm text-dark-400 mb-2">{category.description}</p>
          <div class="grid grid-cols-2 lg:grid-cols-3 gap-2">
            {#each category.tasks as task}
              <button
                class="task-item p-3 bg-base-700 rounded-lg hover:bg-base-600 text-left transition-colors flex flex-col h-full border border-transparent hover:border-primary-500"
                onclick={() => addTaskFromLibrary(selectedStage, task.id)}
              >
                <div class="flex items-center mb-1">
                  <div class="task-icon p-1 mr-2 rounded {getTaskColorClass(task.id)}">
                    <task.icon class="h-4 w-4"></task.icon>
                  </div>
                  <span class="font-medium">{task.name}</span>
                </div>
                <p class="text-xs text-dark-400 flex-1">{task.description}</p>
              </button>
            {/each}
          </div>
        </div>
      {/each}
    </div>
    <div slot="footer" class="flex justify-end">
      <Button variant="outline" onclick={() => showTaskLibrary = false}>
        Close
      </Button>
    </div>
  </Modal>
{/if}
<!-- NEW STAGE MODAL -->
{#if newStageModalOpen}
  <Modal 
    title="Add New Stage" 
    primaryAction="Add Stage" 
    primaryVariant="primary" 
    secondaryAction="Cancel"
    onclose={() => newStageModalOpen = false}
    onprimaryAction={addNewStage}
    onsecondaryAction={() => newStageModalOpen = false}
    isOpen={newStageModalOpen}
  >
    <div class="space-y-4">
      <div>
        <label for="stage-name" class="block text-sm font-medium text-dark-300 mb-1">
          Stage Name <span class="text-danger-500">*</span>
        </label>
        <input
          id="stage-name"
          type="text"
          bind:value={newStage.name}
          placeholder="E.g., Initialize Browser"
          class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        />
      </div>
      <div>
        <label for="stage-description" class="block text-sm font-medium text-dark-300 mb-1">
          Description
        </label>
        <textarea
          id="stage-description"
          bind:value={newStage.description}
          placeholder="What does this stage do?"
          rows="2"
          class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        ></textarea>
      </div>
      <div>
        <label for="stage-parallelism" class="block text-sm font-medium text-dark-300 mb-1">
          Execution Mode
        </label>
        <select
          id="stage-parallelism"
          bind:value={newStage.parallelism.mode}
          class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
        >
          <option value="sequential">Sequential (one after another)</option>
          <option value="parallel">Parallel (run multiple tasks simultaneously)</option>
          <option value="worker-per-item">Worker Per Item (process collections in parallel)</option>
        </select>
        {#if newStage.parallelism.mode !== "sequential"}
          <div class="mt-2">
            <label for="max-workers" class="block text-sm font-medium text-dark-300 mb-1">
              Maximum Workers
            </label>
            <input
              id="max-workers"
              type="number"
              min="1"
              max="20"
              bind:value={newStage.parallelism.maxWorkers}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
          </div>
        {/if}
      </div>
      <div>
        <label for="exec-builder" class="block text-sm font-medium text-dark-300 mb-1">
          Execution Condition
        </label>
        <ConditionBuilder id="exec-builder" bind:condition={newStage.condition} />
      </div>
    </div>
  </Modal>
{/if}
<!-- STAGE CONFIG MODAL -->
{#if stageConfigModalOpen && editingStage}
  <StageConfig 
    stage={editingStage} 
    isOpen={stageConfigModalOpen} 
    onclose={() => stageConfigModalOpen = false}
    onsave={updateStage}
  />
{/if}
<!-- TASK CONFIG MODAL -->
{#if taskConfigModalOpen && editingTask}
  <TaskConfig 
    task={editingTask} 
    allTasks={pipeline.flatMap(stage => stage.tasks)}
    isOpen={taskConfigModalOpen} 
    onclose={() => taskConfigModalOpen = false}
    onsave={updateTask}
  />
{/if}
<!-- JOB CONFIG MODAL -->
{#if jobConfigModalOpen}
  <Modal 
    title="Job Settings" 
    size="xl"
    primaryAction="Save Settings" 
    primaryVariant="primary" 
    secondaryAction="Cancel"
    onclose={() => jobConfigModalOpen = false}
    onprimaryAction={saveJobConfig}
    onsecondaryAction={() => jobConfigModalOpen = false}
    isOpen={jobConfigModalOpen}
  >
    <Tabs 
      tabs={[
        { id: 'browser', label: 'Browser', icon: Cloud },
        { id: 'scraper', label: 'Scraper', icon: Search },
        { id: 'rate', label: 'Rate Limiting', icon: Clock },
        { id: 'resources', label: 'Resources', icon: Database }
      ]} 
      activeTab="browser"
    >
      <div data-tab="browser" class="space-y-4">
        <h3 class="text-lg font-medium mb-2">Browser Settings</h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="headless-mode" class="block text-sm font-medium text-dark-300 mb-1">
              Headless Mode
            </label>
            <div id="headless-mode" class="flex items-center space-x-2">
              <input 
                type="checkbox" 
                id="headless" 
                bind:checked={jobConfig.browserSettings.headless} 
                class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
              />
              <label for="headless" class="text-sm text-dark-300">
                Run browsers in headless mode
              </label>
            </div>
          </div>
          <div>
            <label for="record-video" class="block text-sm font-medium text-dark-300 mb-1">
              Record Video
            </label>
            <div id="record-video" class="flex items-center space-x-2">
              <input 
                type="checkbox" 
                id="recordVideo" 
                bind:checked={jobConfig.browserSettings.recordVideo} 
                class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
              />
              <label for="recordVideo" class="text-sm text-dark-300">
                Record browser sessions as video
              </label>
            </div>
          </div>
          <div>
            <label for="user-agent" class="block text-sm font-medium text-dark-300 mb-1">
              User Agent
            </label>
            <input
              id="user-agent"
              type="text"
              bind:value={jobConfig.browserSettings.userAgent}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </div>
          <div>
            <label for="viewport" class="block text-sm font-medium text-dark-300 mb-1">
              Viewport Size
            </label>
            <div class="flex space-x-2">
              <input
                type="number"
                bind:value={jobConfig.browserSettings.viewportWidth}
                placeholder="Width"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
              />
              <span class="flex items-center text-dark-400">×</span>
              <input
                type="number"
                bind:value={jobConfig.browserSettings.viewportHeight}
                placeholder="Height"
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
              />
            </div>
          </div>
          <div>
            <label for="locale" class="block text-sm font-medium text-dark-300 mb-1">
              Locale
            </label>
            <input
              id="locale"
              type="text"
              bind:value={jobConfig.browserSettings.locale}
              placeholder="en-US"
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </div>
          <div>
            <label for="timezone" class="block text-sm font-medium text-dark-300 mb-1">
              Timezone
            </label>
            <input
              id="timezone"
              type="text"
              bind:value={jobConfig.browserSettings.timezone}
              placeholder="UTC"
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </div>
          <div>
            <label for="default-timeout" class="block text-sm font-medium text-dark-300 mb-1">
              Default Timeout (ms)
            </label>
            <input
              id="default-timeout"
              type="number"
              bind:value={jobConfig.browserSettings.defaultTimeout}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </div>
        </div>
      </div>
      <div data-tab="scraper" class="space-y-4">
        <h3 class="text-lg font-medium mb-2">Scraper Settings</h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="max-depth" class="block text-sm font-medium text-dark-300 mb-1">
              Maximum Crawl Depth
            </label>
            <input
              id="max-depth"
              type="number"
              min="1"
              bind:value={jobConfig.scraperSettings.maxDepth}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </div>
          <div>
            <label for="max-pages" class="block text-sm font-medium text-dark-300 mb-1">
              Maximum Pages to Crawl
            </label>
            <input
              id="max-pages"
              type="number"
              min="1"
              bind:value={jobConfig.scraperSettings.maxPages}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </div>
          <div>
            <label for="max-assets" class="block text-sm font-medium text-dark-300 mb-1">
              Maximum Assets to Extract
            </label>
            <input
              id="max-assets"
              type="number"
              min="1"
              bind:value={jobConfig.scraperSettings.maxAssets}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
          </div>
          <div>
            <label for="navigation-mode" class="block text-sm font-medium text-dark-300 mb-1">
              Default Navigation Mode
            </label>
            <select
              id="navigation-mode"
              bind:value={jobConfig.scraperSettings.defaultNavigationMode}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            >
              <option value="load">load (wait for load event)</option>
              <option value="domcontentloaded">domcontentloaded (wait for DOM only)</option>
              <option value="networkidle">networkidle (wait for network idle)</option>
            </select>
          </div>
          <div id="url-scope">
            <label for="url-scope" class="block text-sm font-medium text-dark-300 mb-1">URL Scope</label>
            <div class="space-y-2">
              <div class="flex items-center space-x-2">
                <input 
                  type="checkbox" 
                  id="follow-redirects" 
                  bind:checked={jobConfig.scraperSettings.followRedirects} 
                  class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
                />
                <label for="follow-redirects" class="text-sm text-dark-300">
                  Follow redirects
                </label>
              </div>
              <div class="flex items-center space-x-2">
                <input 
                  type="checkbox" 
                  id="same-domain" 
                  bind:checked={jobConfig.scraperSettings.sameDomainOnly} 
                  class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
                />
                <label for="same-domain" class="text-sm text-dark-300">
                  Stay on same domain
                </label>
              </div>
              <div class="flex items-center space-x-2">
                <input 
                  type="checkbox" 
                  id="include-subdomains" 
                  bind:checked={jobConfig.scraperSettings.includeSubdomains} 
                  class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
                />
                <label for="include-subdomains" class="text-sm text-dark-300">
                  Include subdomains
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div data-tab="rate" class="space-y-4">
        <h3 class="text-lg font-medium mb-2">Rate Limiting</h3>
        <div>
          <div class="flex items-center space-x-2 mb-4">
            <input 
              type="checkbox" 
              id="rate-limiting-enabled" 
              bind:checked={jobConfig.rateLimiting.enabled} 
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
            />
            <label for="rate-limiting-enabled" class="text-sm font-medium text-dark-300">
              Enable rate limiting
            </label>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4" class:opacity-50={!jobConfig.rateLimiting.enabled}>
            <div>
              <label for="request-delay" class="block text-sm font-medium text-dark-300 mb-1">
                Request Delay (ms)
              </label>
              <input
                id="request-delay"
                type="number"
                min="0"
                bind:value={jobConfig.rateLimiting.requestDelay}
                disabled={!jobConfig.rateLimiting.enabled}
                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
              />
            </div>
            <div id="random-delay-container">
              <label for="random-delay-container" class="block text-sm font-medium text-dark-300 mb-1">Randomize Delay</label>
              <div class="flex items-center space-x-2">
                <input 
                  type="checkbox" 
                  id="randomize-delay" 
                  bind:checked={jobConfig.rateLimiting.randomizeDelay} 
                  disabled={!jobConfig.rateLimiting.enabled}
                  class="h-4 w-4 text-primary-600 focus:ring-primary-500 rounded bg-base-700 border-dark-500"
                />
                <label for="randomize-delay" class="text-sm text-dark-300">
                  Add random variation to delay
                </label>
              </div>
            </div>
            {#if jobConfig.rateLimiting.randomizeDelay}
              <div>
                <label for="delay-variation" class="block text-sm font-medium text-dark-300 mb-1">
                  Delay Variation (0-1)
                </label>
                <input
                  id="delay-variation"
                  type="number"
                  min="0"
                  max="1"
                  step="0.1"
                  bind:value={jobConfig.rateLimiting.delayVariation}
                  disabled={!jobConfig.rateLimiting.enabled || !jobConfig.rateLimiting.randomizeDelay}
                  class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
                />
              </div>
            {/if}
          </div>
        </div>
      </div>
      <div data-tab="resources" class="space-y-4">
        <h3 class="text-lg font-medium mb-2">Resource Allocation</h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label for="max-browsers" class="block text-sm font-medium text-dark-300 mb-1">
              Maximum Browser Instances
            </label>
            <input
              id="max-browsers"
              type="number"
              min="1"
              max="10"
              bind:value={jobConfig.resourceSettings.maxBrowsers}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
            <p class="text-xs text-dark-400 mt-1">
              Limit the number of concurrent browser instances
            </p>
          </div>
          <div>
            <label for="max-pages" class="block text-sm font-medium text-dark-300 mb-1">
              Maximum Pages Per Browser
            </label>
            <input
              id="max-pages"
              type="number"
              min="1"
              max="20"
              bind:value={jobConfig.resourceSettings.maxPages}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
            <p class="text-xs text-dark-400 mt-1">
              Limit the number of pages open in each browser
            </p>
          </div>
          <div>
            <label for="max-workers" class="block text-sm font-medium text-dark-300 mb-1">
              Maximum Worker Threads
            </label>
            <input
              id="max-workers"
              type="number"
              min="1"
              max="20"
              bind:value={jobConfig.resourceSettings.maxWorkers}
              class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
            <p class="text-xs text-dark-400 mt-1">
              Limit the number of concurrent worker threads
            </p>
          </div>
        </div>
      </div>
    </Tabs>
  </Modal>
{/if}
<!-- VIEW PIPELINE JSON MODAL -->
{#if viewPipelineModalOpen}
  <Modal 
    title="Pipeline JSON" 
    size="lg"
    onclose={() => viewPipelineModalOpen = false}
    isOpen={viewPipelineModalOpen}
  >
    <div class="relative">
      <Button 
        variant="outline" 
        size="sm" 
        class="absolute top-0 right-0" 
        onclick={() => {
          navigator.clipboard.writeText(JSON.stringify(pipeline, null, 2));
        }}
      >
        <Copy class="h-4 w-4 mr-1" />
        Copy
      </Button>
      <pre class="bg-base-900 p-4 rounded-lg overflow-auto max-h-[60vh] text-sm mt-4">
        {JSON.stringify(pipeline, null, 2)}
      </pre>
    </div>
    <div slot="footer" class="flex justify-end">
      <Button variant="outline" onclick={() => viewPipelineModalOpen = false}>
        Close
      </Button>
    </div>
  </Modal>
{/if}
<style>
  .task-item {
    transition: all 0.2s ease;
  }
  .task-item:hover {
    transform: translateY(-1px);
  }
  .pipeline-stages {
    padding: 1rem;
  }
  /* ANIMATION FOR DRAG PREVIEW */
  @keyframes pulse {
    0%, 100% { opacity: 0.9; }
    50% { opacity: 0.7; }
  }
  .drag-preview {
    animation: pulse 1.5s infinite;
  }
</style>
