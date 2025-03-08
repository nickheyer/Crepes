import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import duration from 'dayjs/plugin/duration';

// Setup dayjs plugins
dayjs.extend(relativeTime);
dayjs.extend(duration);

/**
 * Format a date as a readable string
 * @param {string|Date} date - The date to format
 * @param {string} format - The format string (defaults to 'MMM D, YYYY h:mm A')
 * @returns {string} The formatted date string
 */
export function formatDate(date, format = 'MMM D, YYYY h:mm A') {
  if (!date) return 'N/A';
  return dayjs(date).format(format);
}

/**
 * Format a date as relative time (e.g. "2 hours ago")
 * @param {string|Date} date - The date to format
 * @returns {string} The relative time string
 */
export function formatRelativeTime(date) {
  if (!date) return 'N/A';
  return dayjs(date).fromNow();
}

/**
 * Format a duration in milliseconds to a readable string
 * @param {number} ms - Duration in milliseconds
 * @returns {string} Formatted duration
 */
export function formatDuration(ms) {
  if (!ms) return '0s';
  
  const duration = dayjs.duration(ms);
  const hours = Math.floor(duration.asHours());
  const minutes = duration.minutes();
  const seconds = duration.seconds();
  
  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  } else {
    return `${seconds}s`;
  }
}

/**
 * Format a file size in bytes to a human-readable string
 * @param {number} bytes - Size in bytes
 * @param {number} decimals - Number of decimal places (default: 2)
 * @returns {string} Formatted file size
 */
export function formatFileSize(bytes, decimals = 2) {
  if (bytes === 0 || !bytes) return '0 Bytes';
  
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i];
}

/**
 * Format a number with thousands separators
 * @param {number} num - The number to format
 * @returns {string} Formatted number
 */
export function formatNumber(num) {
  if (num === undefined || num === null) return '0';
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

/**
 * Truncate a string to a maximum length
 * @param {string} str - String to truncate
 * @param {number} maxLength - Maximum length
 * @returns {string} Truncated string
 */
export function truncateString(str, maxLength = 50) {
  if (!str || str.length <= maxLength) return str || '';
  return str.substring(0, maxLength) + '...';
}

/**
 * Format a URL to be display-friendly
 * @param {string} url - URL to format
 * @param {number} maxLength - Maximum display length
 * @returns {string} Formatted URL
 */
export function formatUrl(url, maxLength = 40) {
  if (!url) return '';
  
  try {
    const urlObj = new URL(url);
    let displayUrl = urlObj.hostname + urlObj.pathname;
    
    if (displayUrl.length > maxLength) {
      const hostLength = urlObj.hostname.length;
      const availablePathLength = maxLength - hostLength - 3; // 3 for "..."
      
      if (availablePathLength > 0) {
        displayUrl = urlObj.hostname + '/' + '...' + urlObj.pathname.slice(-availablePathLength);
      } else {
        displayUrl = urlObj.hostname.slice(0, maxLength - 3) + '...';
      }
    }
    
    return displayUrl;
  } catch (e) {
    // If URL parsing fails, just truncate the string
    return truncateString(url, maxLength);
  }
}

/**
 * Get appropriate icon/emoji for asset type
 * @param {string} type - Asset type
 * @returns {string} Icon representation
 */
export function getAssetTypeIcon(type) {
  const icons = {
    'image': '🖼️',
    'video': '🎬',
    'audio': '🔊',
    'document': '📄',
    'unknown': '❓'
  };
  
  return icons[type] || icons.unknown;
}

/**
 * Format a job status with appropriate emoji
 * @param {string} status - Job status
 * @returns {string} Formatted status with emoji
 */
export function formatJobStatus(status) {
  const statusFormats = {
    'idle': '⏸️ Idle',
    'running': '▶️ Running',
    'completed': '✅ Completed',
    'failed': '❌ Failed',
    'stopped': '⏹️ Stopped'
  };
  
  return statusFormats[status] || status;
}


export function getCronDescription(cronExpr) {
  if (!cronExpr) return "";

  const parts = cronExpr.split(" ");
  if (parts.length !== 5) return "Invalid cron expression";

  const minute = parts[0];
  const hour = parts[1];
  const dom = parts[2];
  const month = parts[3];
  const dow = parts[4];

  // Common patterns
  if (
      minute === "0" &&
      hour === "0" &&
      dom === "*" &&
      month === "*" &&
      dow === "*"
  ) {
      return "Runs daily at midnight";
  }

  if (
      minute === "0" &&
      hour === "*" &&
      dom === "*" &&
      month === "*" &&
      dow === "*"
  ) {
      return "Runs hourly at the start of the hour";
  }

  if (
      minute === "0" &&
      hour === "0" &&
      dom === "1" &&
      month === "*" &&
      dow === "*"
  ) {
      return "Runs at midnight on the first day of each month";
  }

  // Weekly patterns
  if (
      minute === "0" &&
      hour === "0" &&
      dom === "*" &&
      month === "*" &&
      dow !== "*"
  ) {
      const days = {
          "0": "Sunday",
          "1": "Monday",
          "2": "Tuesday",
          "3": "Wednesday",
          "4": "Thursday",
          "5": "Friday",
          "6": "Saturday",
      };

      if (dow.includes("-")) {
          const range = dow.split("-");
          return `Runs at midnight on ${days[range[0]]} through ${days[range[1]]}`;
      } else if (dow.includes(",")) {
          const dayList = dow
              .split(",")
              .map((d) => days[d])
              .join(", ");
          return `Runs at midnight on ${dayList}`;
      } else {
          return `Runs at midnight every ${days[dow]}`;
      }
  }

  // Daily at specific hour
  if (
      minute === "0" &&
      !hour.includes("*") &&
      dom === "*" &&
      month === "*" &&
      dow === "*"
  ) {
      return `Runs daily at ${hour}:00`;
  }

  // Complex pattern, give a generic description
  return "Custom schedule defined by cron expression";
}

/**
 * Format a job progress percentage
 * @param {number} progress - Progress value (0-100)
 * @returns {string} Formatted progress percentage
 */
export function formatProgress(progress) {
  if (progress === undefined || progress === null) return '0%';
  return `${Math.round(progress)}%`;
}
