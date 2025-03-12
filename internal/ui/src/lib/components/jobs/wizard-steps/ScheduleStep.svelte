<script>
    import { onMount } from "svelte";
    import { isValidCron } from "$lib/utils/validation";
    import { getCronDescription } from "$lib/utils/formatters";
    import { state as jobState, setStepValidity } from "$lib/stores/jobStore.svelte";

    let enableSchedule = $state(jobState.formData.data.schedule ? true : false);
    let scheduleType = $state("simple"); // 'simple' or 'advanced'
    let cronExpression = $state(jobState.formData.data.schedule || "");
    let frequency = $state("daily");
    let time = $state("00:00");
    let weekday = $state("1"); // Monday
    let dayOfMonth = $state("1");
    let errorMessage = $state("");
    let isValid = $state(true);
    let simpleConfigChanged = $state(false);
    let lastUpdateTimestamp = $state(Date.now()); // TRACK LAST UPDATE

    const frequencies = [
        { id: "hourly", label: "Hourly" },
        { id: "daily", label: "Daily" },
        { id: "weekly", label: "Weekly" },
        { id: "monthly", label: "Monthly" },
    ];

    const weekdays = [
        { id: "1", label: "Monday" },
        { id: "2", label: "Tuesday" },
        { id: "3", label: "Wednesday" },
        { id: "4", label: "Thursday" },
        { id: "5", label: "Friday" },
        { id: "6", label: "Saturday" },
        { id: "0", label: "Sunday" },
    ];

    const daysOfMonth = Array.from({ length: 31 }, (_, i) => ({
        id: String(i + 1),
        label: String(i + 1),
    }));

    onMount(() => {
        // SET DEFAULT VALUES BASED ON EXISTING SCHEDULE
        if (jobState.formData.data.schedule) {
            // TRY TO PARSE THE CRON EXPRESSION
            const parts = jobState.formData.data.schedule.split(" ");
            if (parts.length === 5) {
                const minute = parts[0];
                const hour = parts[1];
                const dom = parts[2];
                const month = parts[3];
                const dow = parts[4];

                // DETERMINE SCHEDULE TYPE AND SET VALUES
                if (dow === "*" && dom === "*") {
                    if (minute === "0" && hour !== "*") {
                        frequency = "daily";
                        time = hour.padStart(2, "0") + ":00";
                    } else if (minute === "0" && hour === "*") {
                        frequency = "hourly";
                    }
                } else if (dom === "*" && dow !== "*") {
                    frequency = "weekly";
                    weekday = dow;
                    if (minute === "0" && hour !== "*") {
                        time = hour.padStart(2, "0") + ":00";
                    }
                } else if (dom !== "*" && dow === "*") {
                    frequency = "monthly";
                    dayOfMonth = dom;
                    if (minute === "0" && hour !== "*") {
                        time = hour.padStart(2, "0") + ":00";
                    }
                } else {
                    scheduleType = "advanced";
                }
            } else {
                scheduleType = "advanced";
            }
        }

        updateCronExpression();
        updateFormData();
    });

    function updateCronExpression() {
        if (scheduleType === "simple") {
            // Parse time
            let hours = "0";
            let minutes = "0";
            if (time) {
                const timeParts = time.split(":");
                if (timeParts.length === 2) {
                    hours = timeParts[0].replace(/^0+/, "") || "0"; // Remove leading zeros
                    minutes = timeParts[1].replace(/^0+/, "") || "0";
                }
            }

            // Build cron expression based on frequency
            let newCronExpression;
            switch (frequency) {
                case "hourly":
                    newCronExpression = `0 * * * *`;
                    break;
                case "daily":
                    newCronExpression = `${minutes} ${hours} * * *`;
                    break;
                case "weekly":
                    newCronExpression = `${minutes} ${hours} * * ${weekday}`;
                    break;
                case "monthly":
                    newCronExpression = `${minutes} ${hours} ${dayOfMonth} * *`;
                    break;
                default:
                    newCronExpression = "";
            }
            
            // ONLY UPDATE IF CHANGED
            if (newCronExpression !== cronExpression) {
                cronExpression = newCronExpression;
            }
        }

        validate();
    }

    function validate() {
        if (!enableSchedule) {
            isValid = true;
            errorMessage = "";
            setStepValidity(5, true);
            return true;
        }

        if (scheduleType === "advanced") {
            // Validate cron expression format
            if (!cronExpression) {
                isValid = false;
                errorMessage = "Cron expression is required";
                setStepValidity(5, false);
                return false;
            }

            if (!isValidCron(cronExpression)) {
                isValid = false;
                errorMessage = "Invalid cron expression format";
                setStepValidity(5, false);
                return false;
            }
        }

        isValid = true;
        errorMessage = "";
        setStepValidity(5, true);
        return true;
    }

    function updateFormData() {
        const schedule = enableSchedule ? cronExpression : "";

        // ONLY UPDATE IF CHANGED
        if (jobState.formData.data.schedule !== schedule) {
            jobState.formData.data.schedule = schedule;
            // UPDATE TIMESTAMP TO TRACK CHANGES
            lastUpdateTimestamp = Date.now();
        }
        
        validate();
    }

    function handleSimpleConfigChange() {
        simpleConfigChanged = true;
    }
</script>

<div>
    <h2 class="text-xl font-semibold mb-4">Schedule</h2>
    <p class="text-dark-300 mb-6">
        Configure when this job should run automatically
    </p>

    <!-- Enable schedule toggle -->
    <div class="bg-base-800 rounded-lg p-4 mb-6">
        <div class="flex items-center">
            <input
                id="enable-schedule"
                type="checkbox"
                bind:checked={enableSchedule}
                class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-dark-500 rounded"
            />
            <label
                for="enable-schedule"
                class="ml-2 block text-sm font-medium text-white"
            >
                Enable Automatic Scheduling
            </label>
        </div>
    </div>

    {#if enableSchedule}
        <!-- Schedule type selector -->
        <div class="bg-base-800 rounded-lg p-4 mb-6">
            <h3 class="text-sm font-medium mb-4">Schedule Type</h3>

            <div class="flex space-x-4">
                <label class="inline-flex items-center">
                    <input
                        type="radio"
                        value="simple"
                        bind:group={scheduleType}
                        class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-dark-500"
                    />
                    <span class="ml-2 text-sm text-white">Simple Schedule</span>
                </label>

                <label class="inline-flex items-center">
                    <input
                        type="radio"
                        value="advanced"
                        bind:group={scheduleType}
                        class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-dark-500"
                    />
                    <span class="ml-2 text-sm text-white">Advanced (Cron)</span>
                </label>
            </div>
        </div>

        {#if scheduleType === "simple"}
            <!-- Simple schedule options -->
            <div class="bg-base-800 rounded-lg p-4 mb-6">
                <h3 class="text-sm font-medium mb-4">Schedule Options</h3>

                <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <div>
                        <label
                            for="frequency"
                            class="block text-sm font-medium text-dark-300 mb-1"
                        >
                            Frequency
                        </label>
                        <select
                            id="frequency"
                            bind:value={frequency}
                            onchange={handleSimpleConfigChange}
                            class="select select-bordered w-full"
                        >
                            {#each frequencies as option}
                                <option value={option.id}>{option.label}</option>
                            {/each}
                        </select>
                    
                    </div>

                    {#if frequency !== "hourly"}
                        <div>
                            <label
                                for="time"
                                class="block text-sm font-medium text-dark-300 mb-1"
                            >
                                Time
                            </label>
                            <input
                                id="time"
                                type="time"
                                bind:value={time}
                                onchange={handleSimpleConfigChange}
                                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                            />
                        </div>
                    {/if}

                    {#if frequency === "weekly"}
                        <div>
                            <label
                                for="weekday"
                                class="block text-sm font-medium text-dark-300 mb-1"
                            >
                                Day of Week
                            </label>
                            <select
                                id="weekday"
                                bind:value={weekday}
                                onchange={handleSimpleConfigChange}
                                class="select select-bordered w-full"
                            >
                                {#each weekdays as day}
                                    <option value={day.id}>{day.label}</option>
                                {/each}
                            </select>
                        </div>
                    {/if}

                    {#if frequency === "monthly"}
                        <div>
                            <label
                                for="day-of-month"
                                class="block text-sm font-medium text-dark-300 mb-1"
                            >
                                Day of Month
                            </label>
                            <select
                                id="day-of-month"
                                bind:value={dayOfMonth}
                                onchange={handleSimpleConfigChange}
                                class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                            >
                                {#each daysOfMonth as day}
                                    <option value={day.id}>{day.label}</option>
                                {/each}
                            </select>
                        </div>
                    {/if}
                </div>
            </div>
        {:else}
            <!-- Advanced cron schedule -->
            <div class="bg-base-800 rounded-lg p-4 mb-6">
                <h3 class="text-sm font-medium mb-4">Cron Expression</h3>

                <div>
                    <label
                        for="cron-expression"
                        class="block text-sm font-medium text-dark-300 mb-1"
                    >
                        Cron Expression
                    </label>
                    <input
                        id="cron-expression"
                        type="text"
                        bind:value={cronExpression}
                        placeholder="* * * * *"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="mt-1 text-xs text-dark-400">
                        Format: minute hour day-of-month month day-of-week
                    </p>

                    {#if errorMessage}
                        <p class="mt-1 text-xs text-danger-500">
                            {errorMessage}
                        </p>
                    {/if}
                </div>
            </div>
        {/if}

        <!-- Cron preview -->
        {#if isValid && cronExpression}
            <div class="bg-base-850 rounded-lg p-4 mb-6">
                <h3 class="text-sm font-medium mb-2">Schedule Preview</h3>
                <p class="text-sm font-mono text-primary-400">
                    {cronExpression}
                </p>
                <p class="mt-2 text-sm text-dark-300">
                    {getCronDescription(cronExpression)}
                </p>
            </div>
        {/if}

        <!-- Help section -->
        <div class="bg-base-850 rounded-lg p-4">
            <h3 class="text-sm font-medium mb-2">Schedule Examples</h3>
            <ul class="text-xs text-dark-300 list-disc pl-5 space-y-1">
                <li>
                    <code
                        class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                        >0 * * * *</code
                    > - Run every hour at the start of the hour
                </li>
                <li>
                    <code
                        class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                        >0 0 * * *</code
                    > - Run daily at midnight
                </li>
                <li>
                    <code
                        class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                        >0 9 * * 1-5</code
                    > - Run at 9:00 AM, Monday through Friday
                </li>
                <li>
                    <code
                        class="px-1 py-0.5 rounded bg-base-700 text-xs font-mono"
                        >0 0 1 * *</code
                    > - Run at midnight on the first day of each month
                </li>
            </ul>
        </div>
    {/if}
</div>
