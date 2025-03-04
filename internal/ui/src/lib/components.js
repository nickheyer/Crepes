
export const buttonClasses = {
    primary: "px-4 py-2 bg-indigo-600 rounded-md hover:bg-indigo-700 transition focus:outline-none focus:ring-2 focus:ring-indigo-500",
    secondary: "px-4 py-2 bg-gray-600 rounded-md hover:bg-gray-700 transition focus:outline-none focus:ring-2 focus:ring-gray-500",
    success: "px-4 py-2 bg-green-600 rounded-md hover:bg-green-700 transition focus:outline-none focus:ring-2 focus:ring-green-500",
    danger: "px-4 py-2 bg-red-600 rounded-md hover:bg-red-700 transition focus:outline-none focus:ring-2 focus:ring-red-500",
    warning: "px-4 py-2 bg-yellow-600 rounded-md hover:bg-yellow-700 transition focus:outline-none focus:ring-2 focus:ring-yellow-500"
};

export const badgeClasses = {
    idle: "px-2 py-1 bg-gray-500 text-white rounded-full text-xs font-semibold",
    running: "px-2 py-1 bg-green-500 text-white rounded-full text-xs font-semibold",
    completed: "px-2 py-1 bg-blue-500 text-white rounded-full text-xs font-semibold",
    failed: "px-2 py-1 bg-red-500 text-white rounded-full text-xs font-semibold",
    stopped: "px-2 py-1 bg-yellow-500 text-white rounded-full text-xs font-semibold"
};

export const cardClasses = "bg-gray-800 shadow rounded-lg p-6";

export const inputClasses = "w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500";
export const selectClasses = "w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500";
export const checkboxClasses = "rounded bg-gray-700 border-gray-600 text-indigo-600 focus:ring-indigo-500";

export function formatDate(dateString) {
    if (!dateString) return "N/A";
    return new Date(dateString).toLocaleString();
}

export function formatSize(bytes) {
    if (!bytes) return "Unknown";
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(2) + " " + sizes[i];
}

export function getAssetIcon(type) {
    const icons = {
        video: "🎬",
        image: "🖼️",
        audio: "🔊",
        document: "📄",
        unknown: "❓"
    };
    return icons[type] || icons["unknown"];
}

export function createToastSystem() {
    let toastQueue = [];
    let toastTimeout;

    function showToast(message, type = 'info', duration = 2000) {
        const id = Date.now();
        toastQueue = [...toastQueue, { id, message, type }];

        clearTimeout(toastTimeout);
        toastTimeout = setTimeout(() => {
            if (toastQueue.length > 0) {
                toastQueue = toastQueue.slice(1);
            }
        }, duration);
    }

    return { toastQueue, showToast };
}
