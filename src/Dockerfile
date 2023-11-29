# Use an official Golang runtime as a parent image
FROM golang:1.21.4 as build

# Set the working directory inside the container
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . /app

# Build the Go application
RUN go build -o main .

FROM golang:1.21.4 as run

WORKDIR /app
COPY --from=0 /app/main /app

# Expose port 8080 to the outside world
# EXPOSE 8080

# Command to run the executable
CMD ["./main"]

