#!/bin/bash

# Check if sharewoodserver is running
if pgrep -f sharewoodserver > /dev/null; then
    echo "sharewoodserver is running. Killing process..."
    # Kill the process
    pkill -f sharewoodserver
    # Give it a moment to terminate
    sleep 1
    echo "Process terminated."
else
    echo "No running sharewoodserver found."
fi

# Build sharewoodserver
echo "Building sharewoodserver..."
go build -o sharewoodserver main.go

# Check if build was successful
if [ $? -eq 0 ]; then
    echo "Build successful. Starting sharewoodserver..."
    # Start sharewoodserver in the background
    ./sharewoodserver &
    echo "sharewoodserver started with PID: $!"
else
    echo "Build failed. Please check for errors."
    exit 1
fi
