# docker compose up kafka-1 kafka-2 kafka-3 kafka-ui --build

# docker compose up ingestion billing-api flink-jobmanager flink-taskmanager --build

# docker compose logs ingestion

# docker exec -it postgres bash

# psql -U billing -d billing

# select * from "usage_aggregates";

# Follow logs
# docker compose logs -f

# Run command in kafka
# /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 > /dev/null 2>&1
# docker exec -it project-kafka-1-1 /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server kafka-1:9092
# docker exec -it project-kafka-1-1 /opt/kafka/bin/kafka-metadata.sh --snapshot /var/kafka-logs/__cluster_metadata-0/00000000000000000000.log --command "cat -all" | grep -i controller

# Produce messages
# docker exec -it kafka1 /bin/kafka-console-producer --topic test-topic --bootstrap-server kafka1:9092