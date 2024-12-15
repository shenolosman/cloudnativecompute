# Build stage
FROM python:3.11-slim-bookworm AS build-env

# Install system dependencies
RUN apt-get update && \
apt-get install -y libpq-dev gcc && \
apt-get clean && \
rm -rf /var/lib/apt/lists/*

# Create virtual environment
RUN python -m venv /opt/venv

# Activate virtual environment
ENV PATH="/opt/venv/bin:$PATH"
COPY . /app
WORKDIR /app
RUN pip install --no-cache-dir -r requirements.txt

# Run stage
FROM python:3.11-slim-bookworm

# Retain libq5 as it is required by psycopg2
RUN apt-get update && apt-get install -y libpq5

# Copy virtual environment from build stage
COPY --from=build-env /opt/venv /opt/venv

# Set environment variables to use the venv
ENV PATH="/opt/venv/bin:$PATH"
WORKDIR /app
COPY . /app

EXPOSE 5000

CMD ["python", "main.py"]
