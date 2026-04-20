package com.billing;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.connector.jdbc.JdbcConnectionOptions;
import org.apache.flink.connector.jdbc.JdbcExecutionOptions;
import org.apache.flink.connector.jdbc.JdbcSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.CheckpointingMode;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.windowing.ProcessWindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.TumblingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.time.Time;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;
import org.apache.kafka.clients.consumer.OffsetResetStrategy;

import java.time.Duration;
import java.time.Instant;

public class AggregationJob {

    private static final ObjectMapper MAPPER = new ObjectMapper()
        .registerModule(new JavaTimeModule());

    public static void main(String[] args) throws Exception {
        final StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        env.enableCheckpointing(30_000);
        env.getCheckpointConfig().setCheckpointingMode(CheckpointingMode.EXACTLY_ONCE);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(10_000);
        env.getCheckpointConfig().setCheckpointTimeout(60_000);
        env.getCheckpointConfig().setTolerableCheckpointFailureNumber(3);

        String brokers  = System.getenv().getOrDefault("KAFKA_BROKERS",  "kafka-1:9092");
        String dbUrl    = System.getenv().getOrDefault("DATABASE_URL",   "jdbc:postgresql://citus-coordinator:5432/billing");
        String dbUser   = System.getenv().getOrDefault("DB_USER",        "billing");
        String dbPass   = System.getenv().getOrDefault("DB_PASSWORD",    "billing_secret");

        KafkaSource<String> kafkaSource = KafkaSource.<String>builder()
            .setBootstrapServers(brokers)
            .setTopics("billing.events")
            .setGroupId("flink-aggregator")
            .setStartingOffsets(OffsetsInitializer.committedOffsets(OffsetResetStrategy.EARLIEST))
            .setValueOnlyDeserializer(new SimpleStringSchema())
            // read_committed skips messages from aborted Kafka transactions
            .setProperty("isolation.level", "read_committed")
            .build();

        WatermarkStrategy<UsageEvent> watermarkStrategy = WatermarkStrategy
            .<UsageEvent>forBoundedOutOfOrderness(Duration.ofSeconds(30))
            .withTimestampAssigner((event, ts) -> event.timestamp.toEpochMilli())
            .withIdleness(Duration.ofMinutes(1));

        DataStream<UsageEvent> events = env
            .fromSource(kafkaSource, WatermarkStrategy.noWatermarks(), "Kafka Source")
            .map(json -> MAPPER.readValue(json, UsageEvent.class))
            .assignTimestampsAndWatermarks(watermarkStrategy);

        buildPipeline(events, Time.minutes(1),  "1 minute", dbUrl, dbUser, dbPass);
        buildPipeline(events, Time.hours(1),    "1 hour",   dbUrl, dbUser, dbPass);
        buildPipeline(events, Time.hours(24),   "1 day",    dbUrl, dbUser, dbPass);

        env.execute("multi-tenant-billing-aggregation");
    }

    private static void buildPipeline(
        DataStream<UsageEvent> events,
        Time windowSize,
        final String windowDuration,
        String dbUrl, String dbUser, String dbPass
    ) {
        events
            .keyBy(e -> e.tenantId + "|" + e.type)
            .window(TumblingEventTimeWindows.of(windowSize))
            .allowedLateness(Time.minutes(5))
            .sideOutputLateData(LateEventTag.TAG)
            .aggregate(new WindowAggregator(), new ProcessWindowFunction<UsageAggregate, UsageAggregate, String, TimeWindow>() {
                @Override
                public void process(String key, Context ctx, Iterable<UsageAggregate> elements, Collector<UsageAggregate> out) {
                    UsageAggregate agg = elements.iterator().next();
                    agg.windowStart    = Instant.ofEpochMilli(ctx.window().getStart());
                    agg.windowEnd      = Instant.ofEpochMilli(ctx.window().getEnd());
                    agg.windowDuration = windowDuration;
                    out.collect(agg);
                }
            })
            .addSink(JdbcSink.sink(
                "INSERT INTO usage_aggregates " +
                "(tenant_id, event_type, window_start, window_end, window_duration, total_quantity, event_count) " +
                "VALUES (?, ?, ?, ?, ?, ?, ?) " +
                "ON CONFLICT (tenant_id, event_type, window_start, window_duration) " +
                "DO UPDATE SET total_quantity = EXCLUDED.total_quantity, " +
                "              event_count    = EXCLUDED.event_count, " +
                "              updated_at     = NOW()",
                (stmt, agg) -> {
                    stmt.setString(1, agg.tenantId);
                    stmt.setString(2, agg.eventType);
                    stmt.setTimestamp(3, java.sql.Timestamp.from(agg.windowStart));
                    stmt.setTimestamp(4, java.sql.Timestamp.from(agg.windowEnd));
                    stmt.setString(5, agg.windowDuration);
                    stmt.setDouble(6, agg.totalQuantity);
                    stmt.setLong(7, agg.eventCount);
                },
                JdbcExecutionOptions.builder()
                    .withBatchSize(1000)
                    .withBatchIntervalMs(200)
                    .withMaxRetries(3)
                    .build(),
                new JdbcConnectionOptions.JdbcConnectionOptionsBuilder()
                    .withUrl(dbUrl)
                    .withDriverName("org.postgresql.Driver")
                    .withUsername(dbUser)
                    .withPassword(dbPass)
                    .build()
            ));
    }
}
