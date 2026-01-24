FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY ml-service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY ml-service/ .

# Create non-root user
RUN useradd -m -u 1000 mlservice && \
    chown -R mlservice:mlservice /app

USER mlservice

EXPOSE 5000

# Run with gunicorn for production
CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "2", "--threads", "4", "--timeout", "120", "api.app:app"]
