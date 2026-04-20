package com.billing;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;

public class UsageEvent {
    @JsonProperty("event_id")
    public String eventId;

    @JsonProperty("tenant_id")
    public String tenantId;

    @JsonProperty("type")
    public String type;

    @JsonProperty("quantity")
    public double quantity;

    @JsonProperty("resource_id")
    public String resourceId;

    @JsonProperty("timestamp")
    public Instant timestamp;
}
