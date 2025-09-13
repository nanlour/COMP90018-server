#!/bin/bash

# Start the server and database using docker-compose
echo "Starting Mobile Bill App Server..."
docker-compose up -d

echo "Server is running on http://localhost:8080"
