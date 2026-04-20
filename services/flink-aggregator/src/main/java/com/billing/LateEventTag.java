package com.billing;

import org.apache.flink.util.OutputTag;

public class LateEventTag {
    public static final OutputTag<UsageEvent> TAG = new OutputTag<UsageEvent>("late-events") {};
}
