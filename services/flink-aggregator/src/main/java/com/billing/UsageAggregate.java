package com.billing;

import java.time.Instant;

public class UsageAggregate {
    public String tenantId;
    public String eventType;
    public Instant windowStart;
    public Instant windowEnd;
    public String windowDuration;
    public double totalQuantity;
    public long eventCount;

    public UsageAggregate() {}

    public UsageAggregate(String tenantId, String eventType, Instant windowStart,
                          Instant windowEnd, String windowDuration,
                          double totalQuantity, long eventCount) {
        this.tenantId = tenantId;
        this.eventType = eventType;
        this.windowStart = windowStart;
        this.windowEnd = windowEnd;
        this.windowDuration = windowDuration;
        this.totalQuantity = totalQuantity;
        this.eventCount = eventCount;
    }
}
