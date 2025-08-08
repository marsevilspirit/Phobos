#!/bin/bash

# Start the server in the background
go run server/main.go &
SERVER_PID=$!

# Wait for the server to start
sleep 2

# Run the client
go run client/main.go

# Kill the server
kill $SERVER_PID
