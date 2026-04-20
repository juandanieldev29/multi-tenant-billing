# syntax=docker/dockerfile:1.7

# Stage 1: build the fat JAR with Maven
FROM maven:3.9-eclipse-temurin-11 AS builder

WORKDIR /src

# Cache Maven dependencies before copying source (layer cache-friendly)
COPY pom.xml .
RUN mvn dependency:go-offline -q

COPY src ./src
RUN mvn clean package -DskipTests -q

# Stage 2: Flink runtime image with the job JAR
FROM flink:1.18-java11

RUN mkdir -p /opt/flink/usrlib

COPY --from=builder /src/target/flink-aggregator-1.0.0.jar /opt/flink/usrlib/flink-aggregator.jar

# Flink config — override via FLINK_PROPERTIES env var in docker-compose / Helm
ENV KAFKA_BROKERS=kafka-1:9092,kafka-2:9093,kafka-3:9094
ENV DATABASE_URL=jdbc:postgresql://citus-coordinator:5432/billing
ENV DB_USER=billing
ENV DB_PASSWORD=billing_secret

# The official Flink image entrypoint reads the CMD to decide jobmanager vs taskmanager.
# Override CMD at runtime: docker run ... jobmanager   or   docker run ... taskmanager
