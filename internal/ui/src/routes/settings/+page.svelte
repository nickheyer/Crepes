<script>
    import { onMount } from "svelte";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import ThemeController from "$lib/components/settings/ThemeController.svelte";
    import { addToast, availableThemes, state as uiState } from "$lib/stores/uiStore.svelte";
    import { settingsApi } from "$lib/utils/api";

    let loading = $state(false);
    let saving = $state(false);
    let settings = $state({
        appConfig: {
            port: 8080,
            storagePath: "./storage",
            thumbnailsPath: "./thumbnails",
            dataPath: "./data",
            maxConcurrent: 5,
            defaultTimeout: 5 * 60 * 1000, // 5 MINUTES IN MS
        },
        userConfig: {
            theme: uiState.theme,
            defaultView: "grid",
            notificationsEnabled: true
        }
    });
    let storageInfo = $state({
        totalSpace: "0 B",
        usedSpace: "0 B",
        freeSpace: "0 B",
        assetsSize: "0 B",
        thumbnailSize: "0 B",
        raw: {
            totalBytes: 0,
            usedBytes: 0,
            freeBytes: 0,
            assetsBytes: 0,
            thumbsBytes: 0
        }
    });
    
    onMount(async () => {
        loading = true;
        try {
            // FETCH SETTINGS FROM API
            try {
                const response = await settingsApi.getAll();
                if (response.success && response.data) {
                    settings = response.data;
                } else {
                    settings = response;
                }
            } catch (error) {
                console.error("ERROR LOADING SETTINGS:", error);
                // USE DEFAULT SETTINGS IF API FAILS
            }
            
            // FETCH STORAGE INFO
            try {
                const response = await settingsApi.getStorageInfo();
                if (response.success && response.data) {
                    storageInfo = response.data;
                }
            } catch (error) {
                console.error("ERROR LOADING STORAGE INFO:", error);
                // USE DEFAULT INFO IF API FAILS
            }
        } finally {
            loading = false;
        }
    });
    
    async function saveSettings() {
        saving = true;
        try {
            
            settings.userConfig.theme = uiState.theme;
            const response = await settingsApi.update(settings);
            if (response.success) {
                addToast("SETTINGS SAVED SUCCESSFULLY", "success");
            } else {
                throw new Error("FAILED TO SAVE SETTINGS");
            }

        } catch (error) {
            console.error("ERROR SAVING SETTINGS:", error);
            addToast("FAILED TO SAVE SETTINGS: " + error.message, "error");
        } finally {
            saving = false;
        }
    }
    
    async function handleClearCache() {
        try {
            const response = await settingsApi.clearCache();
            if (response.success) {
                addToast("CACHE CLEARED SUCCESSFULLY", "success");
            } else {
                throw new Error("FAILED TO CLEAR CACHE");
            }
        } catch (error) {
            console.error("ERROR CLEARING CACHE:", error);
            addToast("FAILED TO CLEAR CACHE: " + error.message, "error");
        }
    }
    
    function resetSettings() {
        settings = {
            appConfig: {
                port: 8080,
                storagePath: "./storage",
                thumbnailsPath: "./thumbnails",
                dataPath: "./data",
                maxConcurrent: 5,
                defaultTimeout: 5 * 60 * 1000, // 5 MINUTES IN MS
            },
            userConfig: {
                theme: uiState.theme,
                defaultView: "grid",
                notificationsEnabled: true
            }
        };
        addToast("SETTINGS RESET TO DEFAULTS", "info");
    }
    
    // CALCULATE STORAGE USAGE PERCENTAGE
    function getStorageUsagePercentage() {
        if (storageInfo.raw && storageInfo.raw.totalBytes > 0) {
            return Math.floor((storageInfo.raw.usedBytes / storageInfo.raw.totalBytes) * 100);
        }
        return 0;
    }
</script>

<svelte:head>
    <title>Settings | Crepes</title>
</svelte:head>

<div class="container mx-auto p-6">
    <div class="mb-8">
        <h1 class="text-3xl font-bold mb-2">Settings</h1>
        <p class="text-dark-300">Configure application settings and preferences</p>
    </div>
    
    {#if loading}
        <div class="py-20 flex justify-center">
            <span class="loading loading-spinner loading-lg text-primary"></span>
        </div>
    {:else}
        <div class="grid grid-cols-1 lg:grid-cols-12 gap-8">
            <!-- LEFT COLUMN: STORAGE INFO -->
            <div class="lg:col-span-4 space-y-8">
                <Card title="Storage Information" class="card bg-base-200 shadow-xl">
                    <div class="card-body">
                        <div class="mb-6">
                            <div class="flex justify-between items-center mb-2">
                                <span class="text-sm font-semibold">Storage Usage</span>
                                <span class="badge badge-primary">{getStorageUsagePercentage()}%</span>
                            </div>
                            <progress 
                                class="progress progress-primary w-full" 
                                value={getStorageUsagePercentage()} 
                                max="100"
                            ></progress>
                            <div class="flex justify-between mt-2">
                                <span class="text-xs text-dark-400">{storageInfo.usedSpace} used</span>
                                <span class="text-xs text-dark-400">{storageInfo.freeSpace} free</span>
                            </div>
                        </div>
                        
                        <div class="stats stats-vertical shadow bg-base-300 w-full">
                            <div class="stat">
                                <div class="stat-title">Total Storage</div>
                                <div class="stat-value text-xl">{storageInfo.totalSpace}</div>
                            </div>
                            
                            <div class="stat">
                                <div class="stat-title">Used Space</div>
                                <div class="stat-value text-xl">{storageInfo.usedSpace}</div>
                            </div>
                        </div>
                        
                        <div class="divider"></div>
                        
                        <div class="stats stats-vertical shadow bg-base-300 w-full">
                            <div class="stat">
                                <div class="stat-title">Assets Storage</div>
                                <div class="stat-value text-lg">{storageInfo.assetsSize}</div>
                            </div>
                            
                            <div class="stat">
                                <div class="stat-title">Thumbnails Storage</div>
                                <div class="stat-value text-lg">{storageInfo.thumbnailSize}</div>
                            </div>
                        </div>
                        
                        <div class="card-actions justify-end mt-4">
                            <Button variant="outline" onclick={handleClearCache} class="btn btn-outline">
                                Clear Cache
                            </Button>
                        </div>
                    </div>
                </Card>
            </div>
            
            <!-- RIGHT COLUMN: SETTINGS -->
            <div class="lg:col-span-8 space-y-8">
                <Card title="Application Settings" class="card bg-base-200 shadow-xl">
                    <div class="card-body">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-8 mb-6">
                            <div>
                                <label for="port" class="label">
                                    <span class="label-text">Server Port</span>
                                </label>
                                <input
                                    id="port"
                                    type="number"
                                    bind:value={settings.appConfig.port}
                                    min="1"
                                    max="65535"
                                    class="input input-bordered w-full"
                                />
                                <label class="label" for="port">
                                    <span class="label-text-alt">The port the application server runs on</span>
                                </label>
                            </div>
                            
                            <div>
                                <label for="concurrent" class="label">
                                    <span class="label-text">Max Concurrent Connections</span>
                                </label>
                                <input
                                    id="concurrent"
                                    type="number"
                                    bind:value={settings.appConfig.maxConcurrent}
                                    min="1"
                                    max="100"
                                    class="input input-bordered w-full"
                                />
                                <label class="label" for="concurrent">
                                    <span class="label-text-alt">Maximum concurrent connections per job</span>
                                </label>
                            </div>
                            
                            <div>
                                <label for="timeout" class="label">
                                    <span class="label-text">Default Timeout (ms)</span>
                                </label>
                                <input
                                    id="timeout"
                                    type="number"
                                    bind:value={settings.appConfig.defaultTimeout}
                                    min="1000"
                                    step="1000"
                                    class="input input-bordered w-full"
                                />
                                <label class="label" for="timeout">
                                    <span class="label-text-alt">Default timeout for jobs (milliseconds)</span>
                                </label>
                            </div>
                        </div>
                        
                        <div class="divider">File Paths</div>
                        
                        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
                            <div>
                                <label for="storage-path" class="label">
                                    <span class="label-text">Storage Path</span>
                                </label>
                                <input
                                    id="storage-path"
                                    type="text"
                                    bind:value={settings.appConfig.storagePath}
                                    class="input input-bordered w-full"
                                />
                            </div>
                            
                            <div>
                                <label for="thumbs-path" class="label">
                                    <span class="label-text">Thumbnails Path</span>
                                </label>
                                <input
                                    id="thumbs-path"
                                    type="text"
                                    bind:value={settings.appConfig.thumbnailsPath}
                                    class="input input-bordered w-full"
                                />
                            </div>
                            
                            <div>
                                <label for="data-path" class="label">
                                    <span class="label-text">Data Path</span>
                                </label>
                                <input
                                    id="data-path"
                                    type="text"
                                    bind:value={settings.appConfig.dataPath}
                                    class="input input-bordered w-full"
                                />
                            </div>
                        </div>
                    </div>
                </Card>
                
                <Card title="User Preferences" class="card bg-base-200 shadow-xl">
                    <div class="card-body">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                            <div>
                                <label class="label" for="theme">
                                    <span class="label-text">App Theme</span>
                                </label>
                                <ThemeController bind:theme={settings.userConfig.theme} />
                            </div>
                            
                            <div>
                                <label for="default-view" class="label">
                                    <span class="label-text">Default Asset View</span>
                                </label>
                                <select
                                    id="default-view"
                                    bind:value={settings.userConfig.defaultView}
                                    class="select select-bordered w-full"
                                >
                                    <option value="grid">Grid</option>
                                    <option value="list">List</option>
                                </select>
                                <label class="label" for="default-view">
                                    <span class="label-text-alt">Default view mode for assets gallery</span>
                                </label>
                            </div>
                            
                            <div>
                                <label class="label" for="notifications">
                                    <span class="label-text">Notifications</span>
                                </label>
                                <div class="form-control">
                                    <label class="label cursor-pointer justify-start">
                                        <input
                                            id="notifications"
                                            type="checkbox"
                                            bind:checked={settings.userConfig.notificationsEnabled}
                                            class="checkbox checkbox-primary mr-2"
                                        />
                                        <span class="label-text">Enable Notifications</span>
                                    </label>
                                </div>
                                <label class="label" for="notifications">
                                    <span class="label-text-alt">Show system notifications for important events</span>
                                </label>
                            </div>
                        </div>
                    </div>
                </Card>
                
                <div class="flex justify-between mt-8">
                    <Button variant="outline" onclick={resetSettings} class="btn btn-outline">
                        Reset to Defaults
                    </Button>
                    <Button 
                        variant="primary" 
                        onclick={saveSettings} 
                        loading={saving}
                        class="btn btn-primary"
                    >
                        {#if saving}
                            <span class="loading loading-spinner loading-xs"></span>
                        {/if}
                        Save Settings
                    </Button>
                </div>
            </div>
        </div>
    {/if}
</div>
