// todo check if the referrer works properly on SPAs
// todo check if the links get triggered on SPAs
const url = "http://localhost:8080/api/event"; // todo: change to production url

// Helper function to format goal names
function formatGoal(goal) {
  // Remove protocol
  goal = goal.replace(/(^\w+:|^)\/\//, "");

  // Remove trailing slashes
  goal = goal.replace(/\/$/, "");

  // Replace special characters with underscores
  goal = goal.replace(/[^\w\s]/g, "_");

  // Remove leading and trailing underscores
  goal = goal.replace(/^_+|_+$/g, "");

  return goal;
}

function trackDownload(event) {
  const link = event.currentTarget;
  const formattedGoal = formatGoal(link.getAttribute("download") || link.href);
  console.log(`Download initiated: ${formattedGoal}`);

  const eventData = {
    type: "download",
    timestamp: new Date().toISOString(),
    referrer: document.referrer || null,
    url: window.location.href,
    pathname: window.location.pathname,
    userAgent: navigator.userAgent,
    language: navigator.language,
    name: `download_${formattedGoal}`,
  };
  sendEventData(eventData);
}

function trackOutboundLink(event) {
  const link = event.currentTarget;
  const formattedGoal = formatGoal(link.href);
  console.log(`Outbound link clicked: ${formattedGoal}`);

  const eventData = {
    type: "outbound_link",
    timestamp: new Date().toISOString(),
    referrer: document.referrer || null,
    url: window.location.href,
    pathname: window.location.pathname,
    userAgent: navigator.userAgent,
    language: navigator.language,
    name: `outbound_${formattedGoal}`,
  };
  sendEventData(eventData);
}

function trackMailtoLink(event) {
  const link = event.currentTarget;
  const formattedGoal = formatGoal(link.href.replace("mailto:", ""));
  console.log(`Mailto link clicked: ${formattedGoal}`);

  const eventData = {
    type: "mailto_link",
    timestamp: new Date().toISOString(),
    referrer: document.referrer || null,
    url: window.location.href,
    pathname: window.location.pathname,
    userAgent: navigator.userAgent,
    language: navigator.language,
    name: `mailto_${formattedGoal}`,
  };
  sendEventData(eventData);
}

function sendEventData(eventData) {
  console.log(eventData);
  fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(eventData),
  })
    .then((response) => {
      if (response.ok) {
        console.log("Event sent successfully");
      } else {
        console.error("Failed to send event");
      }
    })
    .catch((error) => {
      console.error("Error:", error);
    });
}

document.addEventListener("DOMContentLoaded", () => {
  // Attach event listeners to download links
  const downloadLinks = document.querySelectorAll("a[download]");
  downloadLinks.forEach((link) => {
    link.addEventListener("click", trackDownload);
  });

  // Attach event listeners to outbound links
  const outboundLinks = document.querySelectorAll('a[href^="http"]');
  outboundLinks.forEach((link) => {
    // Ensure the link is outbound by checking the domain
    const url = new URL(link.href);
    if (url.origin !== window.location.origin) {
      link.addEventListener("click", trackOutboundLink);
    }
  });

  // Attach event listeners to mailto links
  const mailtoLinks = document.querySelectorAll('a[href^="mailto:"]');
  mailtoLinks.forEach((link) => {
    link.addEventListener("click", trackMailtoLink);
  });
});
