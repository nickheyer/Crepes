<script>
    import { onMount } from "svelte";
    import Card from "$lib/components/common/Card.svelte";
    import Button from "$lib/components/common/Button.svelte";
    import ThemeController from "$lib/components/settings/ThemeController.svelte";
    import { addToast, availableThemes } from "$lib/stores/uiStore";

    const defaultSettings = {
        appConfig: {
            port: 8080,
            storagePath: "./storage",
            thumbnailsPath: "./thumbnails",
            dataPath: "./data",
            maxConcurrent: 5,
            defaultTimeout: 5 * 60 * 1000, // 5 MINUTES IN MS
        },
        userConfig: {
            theme: "default",
            defaultView: "grid",
            notificationsEnabled: true
        }
    };

    // LOCAL STATE
    let loading = $state(false);
    let saving = $state(false);
    let settings = $state(defaultSettings);
    let storageInfo = $state({
        totalSpace: "0 B",
        usedSpace: "0 B",
        freeSpace: "0 B"
    });
    
    onMount(async () => {
        loading = true;
        try {
            // FETCH SETTINGS FROM API IF AVAILABLE
            try {
                const response = await fetch("/api/settings");
                if (response.ok) {
                    const data = await response.json();
                    if (data.success) {
                        settings = data.data;
                    }
                }
            } catch (error) {
                console.error("Error loading settings:", error);
                // USE DEFAULT SETTINGS IF API FAILS
            }
            
            // FETCH STORAGE INFO IF AVAILABLE
            try {
                const response = await fetch("/api/storage/info");
                if (response.ok) {
                    const data = await response.json();
                    if (data.success) {
                        storageInfo = data.data;
                    }
                }
            } catch (error) {
                console.error("Error loading storage info:", error);
                // USE DEFAULT INFO IF API FAILS
            }
        } finally {
            loading = false;
        }
    });
    
    async function saveSettings() {
        saving = true;
        try {
            // SAVE SETTINGS TO API IF AVAILABLE
            try {
                const response = await fetch("/api/settings", {
                    method: "PUT",
                    headers: {
                        "Content-Type": "application/json"
                    },
                    body: JSON.stringify(settings)
                });
                if (response.ok) {
                    addToast("Settings saved successfully", "success");
                } else {
                    throw new Error("Failed to save settings");
                }
            } catch (error) {
                console.error("Error saving settings:", error);
                addToast("Failed to save settings: " + error.message, "error");
            }
        } finally {
            saving = false;
        }
    }
    
    async function clearCache() {
        try {
            // CLEAR CACHE API IF AVAILABLE
            const response = await fetch("/api/cache/clear", {
                method: "POST"
            });
            if (response.ok) {
                addToast("Cache cleared successfully", "success");
            } else {
                throw new Error("Failed to clear cache");
            }
        } catch (error) {
            console.error("Error clearing cache:", error);
            addToast("Failed to clear cache: " + error.message, "error");
        }
    }
    
    function resetSettings() {
        // RESET TO DEFAULT SETTINGS
        settings = defaultSettings;
        addToast("Settings reset to defaults", "info");
    }

    // HANDLE THEME CHANGE
    function handleThemeChange(newTheme) {
        settings.userConfig.theme = newTheme;
    }
</script>

<svelte:head>
    <title>Settings | Crepes</title>
</svelte:head>

<section>
    <div class="mb-4">
        <h1 class="text-2xl font-bold mb-2">Settings</h1>
        <p class="text-dark-300">Configure application settings and preferences</p>
    </div>
    
    {#if loading}
        <div class="py-20 flex justify-center">
            <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary-500"></div>
        </div>
    {:else}
        <!-- STORAGE INFO CARD -->
        <Card title="Storage Information" class="mb-6">
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div class="bg-base-700 p-4 rounded-lg">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">Total Storage</h3>
                    <p class="text-2xl font-semibold">{storageInfo.totalSpace}</p>
                </div>
                <div class="bg-base-700 p-4 rounded-lg">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">Used Space</h3>
                    <p class="text-2xl font-semibold">{storageInfo.usedSpace}</p>
                </div>
                <div class="bg-base-700 p-4 rounded-lg">
                    <h3 class="text-sm font-medium text-dark-300 mb-1">Free Space</h3>
                    <p class="text-2xl font-semibold">{storageInfo.freeSpace}</p>
                </div>
            </div>
            <div class="mt-4">
                <div class="relative pt-1">
                    <div class="flex mb-2 items-center justify-between">
                        <div>
                            <span class="text-xs font-semibold inline-block py-1 px-2 uppercase rounded-full text-primary-600 bg-primary-200">
                                Storage Usage
                            </span>
                        </div>
                        <div class="text-right">
                            <span class="text-xs font-semibold inline-block text-primary-600">
                                {storageInfo.usedSpace} / {storageInfo.totalSpace}
                            </span>
                        </div>
                    </div>
                    <div class="overflow-hidden h-2 mb-4 text-xs flex rounded bg-base-600">
                        <div style="width: 30%" class="shadow-none flex flex-col text-center whitespace-nowrap text-white justify-center bg-primary-500"></div>
                    </div>
                </div>
            </div>
        </Card>

        <!-- APPLICATION SETTINGS CARD -->
        <Card title="Application Settings" class="mb-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <label for="port" class="block text-sm font-medium text-dark-300 mb-1">
                        Server Port
                    </label>
                    <input
                        id="port"
                        type="number"
                        bind:value={settings.appConfig.port}
                        min="1"
                        max="65535"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="mt-1 text-xs text-dark-400">The port the application server runs on</p>
                </div>
                <div>
                    <label for="concurrent" class="block text-sm font-medium text-dark-300 mb-1">
                        Max Concurrent Connections
                    </label>
                    <input
                        id="concurrent"
                        type="number"
                        bind:value={settings.appConfig.maxConcurrent}
                        min="1"
                        max="100"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="mt-1 text-xs text-dark-400">Maximum number of concurrent connections per job</p>
                </div>
                <div>
                    <label for="timeout" class="block text-sm font-medium text-dark-300 mb-1">
                        Default Timeout (ms)
                    </label>
                    <input
                        id="timeout"
                        type="number"
                        bind:value={settings.appConfig.defaultTimeout}
                        min="1000"
                        step="1000"
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="mt-1 text-xs text-dark-400">Default timeout for scraping jobs (in milliseconds)</p>
                </div>
                <div>
                    <label for="storage-path" class="block text-sm font-medium text-dark-300 mb-1">
                        Storage Path
                    </label>
                    <input
                        id="storage-path"
                        type="text"
                        bind:value={settings.appConfig.storagePath}
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="mt-1 text-xs text-dark-400">Directory where downloaded assets are stored</p>
                </div>
                <div>
                    <label for="thumbs-path" class="block text-sm font-medium text-dark-300 mb-1">
                        Thumbnails Path
                    </label>
                    <input
                        id="thumbs-path"
                        type="text"
                        bind:value={settings.appConfig.thumbnailsPath}
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="mt-1 text-xs text-dark-400">Directory where asset thumbnails are stored</p>
                </div>
                <div>
                    <label for="data-path" class="block text-sm font-medium text-dark-300 mb-1">
                        Data Path
                    </label>
                    <input
                        id="data-path"
                        type="text"
                        bind:value={settings.appConfig.dataPath}
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    />
                    <p class="mt-1 text-xs text-dark-400">Directory where application data is stored</p>
                </div>
            </div>
        </Card>

        <!-- USER PREFERENCES CARD -->
        <Card title="User Preferences" class="mb-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <!-- THEME PREVIEW -->
                    <div class="mt-4">
                        <span class="block text-sm font-medium text-dark-300 mb-2">App Theme</span>
                        <ThemeController bind:theme={settings.userConfig.theme} availableThemes={availableThemes} />
                        <input id="theme" class="invisible" bind:value={settings.userConfig.theme}/>
                    </div>
                </div>
                <div>
                    <label for="default-view" class="block text-sm font-medium text-dark-300 mb-1">
                        Default Asset View
                    </label>
                    <select
                        id="default-view"
                        bind:value={settings.userConfig.defaultView}
                        class="w-full px-3 py-2 bg-base-700 border border-dark-600 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    >
                        <option value="grid">Grid</option>
                        <option value="list">List</option>
                    </select>
                    <p class="mt-1 text-xs text-dark-400">Default view mode for assets gallery</p>
                </div>
                <div>
                    <legend class="block text-sm font-medium text-dark-300 mb-1">
                        Notifications
                    </legend>
                    <div class="flex items-center mt-2">
                        <input
                            id="enable-notifications"
                            type="checkbox"
                            bind:checked={settings.userConfig.notificationsEnabled}
                            class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-dark-500 rounded"
                        />
                        <label for="enable-notifications" class="ml-2 block text-sm text-white">
                            Enable Notifications
                        </label>
                    </div>
                </div>
            </div>
        </Card>

        <!-- ACTION BUTTONS -->
        <Card title="Action Buttons" class="mb-6">
            <div class="flex justify-between">
                <div class="space-x-3">
                    <Button variant="outline" onclick={resetSettings}>
                        Reset to Defaults
                    </Button>
                    <Button variant="outline" onclick={clearCache}>
                        Clear Cache
                    </Button>
                </div>
                <Button variant="primary" onclick={saveSettings} loading={saving}>
                    Save Settings
                </Button>
            </div>
        </Card>
    {/if}
</section>
