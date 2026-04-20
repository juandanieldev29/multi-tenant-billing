package com.billing;

import org.apache.flink.api.common.functions.AggregateFunction;

public class WindowAggregator implements AggregateFunction<UsageEvent, UsageAggregate, UsageAggregate> {

    @Override
    public UsageAggregate createAccumulator() {
        return new UsageAggregate();
    }

    @Override
    public UsageAggregate add(UsageEvent event, UsageAggregate acc) {
        if (acc.tenantId == null) {
            acc.tenantId = event.tenantId;
            acc.eventType = event.type;
        }
        acc.totalQuantity += event.quantity;
        acc.eventCount++;
        return acc;
    }

    @Override
    public UsageAggregate getResult(UsageAggregate acc) {
        return acc;
    }

    @Override
    public UsageAggregate merge(UsageAggregate a, UsageAggregate b) {
        return new UsageAggregate(
            a.tenantId, a.eventType, a.windowStart, a.windowEnd, a.windowDuration,
            a.totalQuantity + b.totalQuantity,
            a.eventCount + b.eventCount
        );
    }
}
