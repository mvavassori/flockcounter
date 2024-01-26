// Function to initiate a tracking request
const trackVisit = () => {
    // Prepare payload data
    const now = new Date();
    
    // Use toISOString for ISO 8601 format with UTC timezone
    const formattedStamp = now.toISOString();

    console.log(formattedStamp)
    
    const payloadData = {
        timestamp: formattedStamp,
        referrer: document.referrer || null,
        url: window.location.href,
        pathname: window.location.pathname,
        hash: window.location.hash,
        userAgent: navigator.userAgent,
        language: navigator.language,
        screenWidth: window.screen.width,
        screenHeight: window.screen.height,
        location: Intl.DateTimeFormat().resolvedOptions().timeZone,
    };
  
    console.log(payloadData);
  
    // Construct full API endpoint URL
    const apiUrl = "http://localhost:8080/api/visit";
  
    try {
      fetch(apiUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payloadData),
      })
    
    } catch (error) {
      console.error("Error sending visit data:", error);
    }
  };
  
  // Callback triggered upon DOM ready state
  document.addEventListener("DOMContentLoaded", () => {
    // Trigger initial visit tracking
    trackVisit();
    // sendTimestamp();
  });


  const sendTimestamp = () => {
    // Prepare payload data
    const now = new Date();
    // Use toISOString for ISO 8601 format with UTC timezone
    const formattedStamp = now.toISOString();
    
    // Construct full API endpoint URL
    const apiUrl = "http://localhost:5003/api/v1/TestTimestamp";
  
    // Initiate XMLHttpRequest
    const req = new XMLHttpRequest();
    req.open("POST", apiUrl, true);
    req.setRequestHeader("Content-Type", "application/json");
  
    try {
      req.send(JSON.stringify(formattedStamp));
    } catch (error) {
      console.error("Error sending timestamp:", error);
    }
  };