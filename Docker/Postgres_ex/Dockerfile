# Use the official PostgreSQL image as a base
FROM postgres:16

# Set environment variables
ENV POSTGRES_USER=gds
ENV POSTGRES_PASSWORD=giuseppe
ENV POSTGRES_DB=mydatabase

# Install necessary packages
RUN apt-get update && apt-get install -y \
    wget \
    build-essential \
    golang-go \
    postgresql-server-dev-16 \
    git \
    && rm -rf /var/lib/apt/lists/*

# Download and install pgvector
RUN wget https://github.com/pgvector/pgvector/archive/refs/tags/v0.7.4.tar.gz \
    && tar -xzf v0.7.4.tar.gz \
    && cd pgvector-0.7.4 \
    && make \
    && make install

# Copy the SQL script to create the extension
COPY create_extension.sql /docker-entrypoint-initdb.d/

# Expose the PostgreSQL port
EXPOSE 5432