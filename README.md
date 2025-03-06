# FlockCounter - Privacy-Focused Web Analytics

FlockCounter is a web analytics platform designed with privacy at its core. It provides valuable insights into website traffic and user behavior without compromising the privacy of your visitors. It's built in Go, and uses a PostgreSQL database. It offers a self-hosted solution, giving you full control over your data.

## Features

- **Privacy-Preserving:** FlockCounter avoids using cookies by default. It uses a daily rotating salt combined with IP address, user agent, and website domain to generate a unique identifier for counting unique visitors, without permanently storing any PII (Personally Identifiable Information).
- **Real-time Analytics:** Track live page views and see how users interact with your site in real time.
- **Detailed Metrics:** Get insights into page views, referrers, visit duration, user agents, languages, and countries (using GeoIP).
- **Event Tracking:** Track custom events like downloads, outbound link clicks, mailto links, and form submissions. Easily track custom events by adding a `data-event-name` class to any HTML element.
- **Time-on-Page Tracking:** Accurately measure how long users spend on each page, with consideration for tab switching and inactivity.
- **Referrer Tracking:** Understand where your traffic is coming from, distinguishing between direct visits, search engines, and other websites. Referrers are displayed without query parameters for increased privacy.
- **Single-Page Application (SPA) Support:** Handles route changes in single-page applications correctly, ensuring accurate page view tracking.
- **Self-Hosted:** Maintain full control over your data by hosting FlockCounter on your own infrastructure.
- **REST API:** Access your data programmatically through a comprehensive REST API.
- **Dashboard:** Visualize your data with a user-friendly dashboard (implementation details may vary).
- **Easy Integration:** Integrate the tracking script into your website with a simple JavaScript snippet.
- **GDPR Compliant:** Designed to be compliant with GDPR and other privacy regulations.

## Technology Stack

- **Backend:** Go
- **Database:** PostgreSQL
- **GeoIP:** MaxMind GeoLite2 City database
- **Frontend:** Available at https://github.com/mvavassori/flockcounter-frontend

## Setup and Installation

This section provides a high-level overview of the setup process. More detailed instructions would be included in a production-ready README.

1.  **Prerequisites:**

    - Go (version 1.21 or later)
    - PostgreSQL (version compatible with `pq` driver)
    - Docker (optional, but recommended)
    - MaxMind GeoLite2 City database (`GeoLite2-City.mmdb`)

2.  **Clone the repository:**

    ```bash
    git clone https://github.com/mvavassori/flockcounter.git
    cd flockcounter
    ```

3.  **Install Dependencies:**

    ```bash
    go mod download
    ```

4.  **Database Setup:**

    - Create a PostgreSQL database.
    - Create the necessary tables (schema not provided in context, but would be included here). The schema includes tables like `visits`, `events`, and `daily_unique_identifiers`.

5.  **GeoIP Database:**

    - Download the `GeoLite2-City.mmdb` file from MaxMind.
    - Place the `GeoLite2-City.mmdb` file in the `/app/data/geoip` directory (or adjust the `GEOIP_DB_PATH` environment variable accordingly).

6.  **Environment Variables:**

    - Set the `GEOIP_DB_PATH` environment variable to the path of your `GeoLite2-City.mmdb` file (e.g., `/app/data/geoip/GeoLite2-City.mmdb`).
    - Set other necessary environment variables, such as database connection strings (not provided in context, but would be included here).

7.  **Build and Run (without Docker):**

    ```bash
    go build -o main .
    ./main
    ```

8.  **Build and Run (with Docker):**

    ```bash
    docker build -t flockcounter .
    docker run -p 8080:8080 flockcounter
    ```

9.  **Nginx Configuration (Example):**
    The provided `nginx.conf` shows an example of how to proxy requests to a frontend application (presumably running on port 3000) and configure security headers and gzip compression. This would need to be adapted to your specific setup.

10. **Air (for development):**
    The `.air.toml` file configures the `air` tool for hot-reloading during development. This allows for faster development cycles.
