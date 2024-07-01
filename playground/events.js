// ! just for testing purposes, it should be added to the base script.go if the user needs to track events.
const url = "http://localhost:8080/api/event";

function trackDownload(event) {
  const link = event.currentTarget;
  console.log(`Download initiated: ${link.href}`);

  const eventData = {
    type: "download",
    timestamp: new Date().toISOString(),
    referrer: document.referrer || null,
    url: window.location.href,
    pathname: window.location.pathname,
    userAgent: navigator.userAgent,
    language: navigator.language,
    goal: link.href,
  };
  sendEventData(eventData);
}

// function to track outbound links
function trackOutboundLink(event) {
  const link = event.currentTarget;
  console.log(`Outbound link clicked: ${link.href}`);

  const eventData = {
    type: "outbound_link",
    timestamp: new Date().toISOString(),
    referrer: document.referrer || null,
    url: window.location.href,
    pathname: window.location.pathname,
    userAgent: navigator.userAgent,
    language: navigator.language,
    goal: link.href,
  };
  sendEventData(eventData);
}

function sendEventData(eventData) {
  console.log(eventData);
  // fetch(url, {
  //   method: "POST",
  //   headers: {
  //     "Content-Type": "application/json",
  //   },
  //   body: JSON.stringify({ eventData }),
  // })
  //   .then((response) => {
  //     if (response.ok) {
  //       console.log("Event sent successfully");
  //     } else {
  //       console.error("Failed to send event");
  //     }
  //   })
  //   .catch((error) => {
  //     console.error("Error:", error);
  //   });
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
});
