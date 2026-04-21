FROM apache/kafka:3.7.0

# Copy broker configuration overrides — these merge with the base image defaults
COPY server.properties /opt/kafka/config/kraft/server.properties

USER root
RUN mkdir -p /var/kafka-logs && chown 1000:1000 /var/kafka-logs
USER appuser

EXPOSE 9092 9093

# KRaft mode: the official apache/kafka image reads KAFKA_* env vars and
# auto-formats storage on first start if CLUSTER_ID is set.
# Required env vars at runtime:
#   KAFKA_NODE_ID            — unique integer per broker (1, 2, 3 …)
#   KAFKA_PROCESS_ROLES      — "broker,controller" for combined mode
#   KAFKA_CONTROLLER_QUORUM_VOTERS — e.g. "1@kafka-1:9093,2@kafka-2:9093,3@kafka-3:9093"
#   KAFKA_ADVERTISED_LISTENERS     — e.g. "PLAINTEXT://kafka-1:9092"
#   CLUSTER_ID               — stable UUID shared by all brokers in the cluster
