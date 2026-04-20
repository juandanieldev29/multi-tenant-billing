-- Run AFTER workers have been registered with the coordinator.
-- In docker-compose: run manually once all citus-worker-* containers are up.
-- In Kubernetes: run as a post-install Helm hook.

-- Reference tables are fully replicated to every worker node.
-- Queries against them never cross the network.
SELECT create_reference_table('tenants');
SELECT create_reference_table('pricing_plans');

-- Distributed tables are sharded on tenant_id (128 shards by default).
-- All tables sharing the same distribution column are co-located on the same
-- worker nodes, enabling efficient tenant-scoped JOINs without network hops.
SELECT create_distributed_table('usage_events',     'tenant_id');
SELECT create_distributed_table('usage_aggregates', 'tenant_id', colocate_with => 'usage_events');
SELECT create_distributed_table('invoices',         'tenant_id', colocate_with => 'usage_events');
