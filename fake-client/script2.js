const trackVisit = (isUniqueVisit, timeSpentOnPage) => {
    // Prepare payload data
    const now = new Date();
  
    // Use toISOString for ISO 8601 format with UTC timezone
    const formattedStamp = now.toISOString();
  
    const payloadData = {
      timestamp: formattedStamp,
      referrer: document.referrer || null,
      url: window.location.href,
      pathname: window.location.pathname,
      userAgent: navigator.userAgent,
      language: navigator.language,
      country: getCountry(),
      state: getState(),
      isUnique: isUniqueVisit,
      timeSpentOnPage: timeSpentOnPage
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
      }).then(response => {
        if (!response.ok) {
          throw new Error('Network response was not ok');
        }
      }).catch(error => {
        console.error("Error sending visit data:", error);
      });
    } catch (error) {
      console.error("Error sending visit data:", error);
    }
  };