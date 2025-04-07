# This image is just for backups and cron jobs
FROM alpine:latest

RUN apk update && \
    apk add --no-cache postgresql17-client

# Create the log file directory and file for crond
RUN mkdir -p /var/log && touch /var/log/cron.log


# Create the directory for user crontabs if it doesn't exist
RUN mkdir -p /etc/crontabs

# Entrypoint and command will be handled by docker-compose.yaml
# The default user in Alpine is root, which matches the user specified in the crontab file.