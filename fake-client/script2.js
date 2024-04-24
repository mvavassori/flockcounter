// Prepare payload data
const now = new Date();
const formattedStamp = now.toISOString();
const url = "http://localhost:8080/api/visit";

// Get the current time in milliseconds when the page loads
let startTime = performance.now();
let totalElapsedTime = 0;
let currentUrl = window.location.href;

let currentReferrer = document.referrer || null;

console.log("window.location.host", window.location.host);

let previousPathname = window.location.pathname;

console.log("Page loaded, startTime:", startTime);

function isUniqueVisitor() {
  const uniqueIdKey = "unique_id";

  // Check if the unique_id exists in localStorage
  if (localStorage.getItem(uniqueIdKey)) {
    // The unique_id exists, so it's a returning visitor.
    return false;
  } else {
    // The unique_id does not exist, so it's a new visitor.

    // Generate a new unique identifier and store it in localStorage
    const uniqueId =
      Date.now().toString(36) + Math.random().toString(36).slice(2);
    console.log("uniqueId generated:", uniqueId);
    localStorage.setItem(uniqueIdKey, uniqueId);

    return true;
  }
}

// Function to handle sending the visit data
function sendVisit(elapsedTime) {
  const payloadData = {
    timestamp: formattedStamp,
    referrer: currentReferrer,
    url: currentUrl,
    // pathname: window.location.pathname,
    pathname: previousPathname,
    userAgent: navigator.userAgent,
    language: navigator.language,
    isUniqueVisit: isUniqueVisitor(),
    timeSpentOnPage: Math.round(elapsedTime),
  };
  let data = JSON.stringify(payloadData);
  console.log("Sending visit data:", payloadData);
  navigator.sendBeacon(url, data);
}

// Event listener for page visibility changes
window.addEventListener("visibilitychange", (event) => {
  if (document.visibilityState === "visible") {
    // Page became visible, restart the timer
    startTime = performance.now();
    console.log("Page became visible, startTime updated:", startTime);
  } else {
    // Page became hidden
    const elapsedTime = performance.now() - startTime;
    console.log("Page became hidden, elapsed time:", elapsedTime);
    totalElapsedTime += elapsedTime;
    console.log("Total elapsed time:", totalElapsedTime);

    if (totalElapsedTime < 2000) {
      console.log("Visit time less than 2 seconds, not sending data.");
      // Reset the timer without sending the visit data
      startTime = 0;
      totalElapsedTime = 0;
      console.log(
        "Timer reset, startTime:",
        startTime,
        "totalElapsedTime:",
        totalElapsedTime
      );
    } else {
      sendVisit(totalElapsedTime);
      // Reset the timer after sending the visit data
      startTime = 0;
      totalElapsedTime = 0;
      console.log(
        "Timer reset, startTime:",
        startTime,
        "totalElapsedTime:",
        totalElapsedTime
      );
    }
  }
});

// Event listener for route changes
window.addEventListener("popstate", handleRouteChange);
window.history.pushState = overridePushStateFunction(window.history.pushState);
window.history.replaceState = overrideReplaceStateFunction(
  window.history.replaceState
);

function handleRouteChange() {
  const newUrl = window.location.href;
  if (newUrl !== currentUrl) {
    // Store the current URL as the previous referrer
    currentReferrer = currentUrl;

    console.log("URL changed from", currentUrl, "to", newUrl);
    const elapsedTime = performance.now() - startTime;
    console.log("Page changed, elapsed time:", elapsedTime);
    totalElapsedTime += elapsedTime;
    console.log("Total elapsed time:", totalElapsedTime);
    if (totalElapsedTime < 5000) {
      console.log("Visit time less than 5 seconds, not sending data.");
      // Reset the timer without sending the visit data
      startTime = performance.now();
      totalElapsedTime = 0;
      console.log(
        "New page, startTime updated:",
        startTime,
        "totalElapsedTime:",
        totalElapsedTime
      );
    } else {
      sendVisit(totalElapsedTime);
      currentUrl = newUrl;
      startTime = performance.now();
      totalElapsedTime = 0;
      console.log(
        "New page, startTime updated:",
        startTime,
        "totalElapsedTime:",
        totalElapsedTime
      );
    }
  }
}

function overridePushStateFunction(originalPushState) {
  return function overridenPushState(...args) {
    previousPathname = window.location.pathname; // Store the previous pathname before calling handleRouteChange
    const result = originalPushState.apply(this, args);
    handleRouteChange();
    return result;
  };
}

function overrideReplaceStateFunction(originalReplaceState) {
  return function overridenReplaceState(...args) {
    previousPathname = window.location.pathname;
    const result = originalReplaceState.apply(this, args);
    handleRouteChange();
    return result;
  };
}
