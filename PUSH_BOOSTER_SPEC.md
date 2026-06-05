# Push Booster Specification

## 1. Purpose

Push Booster is a web-push advertising platform for internal operator teams and media buyers.

The system must support:

- web-push subscriber collection;
- campaign and creative management;
- large-scale audience selection;
- fast push delivery over very large subscriber bases;
- campaign/event analytics;
- incoming and outgoing postbacks;
- publisher API;
- admin panel with JWT auth and roles.

## 2. Core Product Flow

1. A publisher installs the JS SDK and service worker on their website.
2. The browser subscribes to web push and sends subscription data to the public events API.
3. The platform stores subscriber credentials and targeting attributes.
4. An operator creates campaigns, creatives, filters, schedules and caps in the admin panel.
5. Scheduler creates a campaign launch.
6. Audience selector builds a campaign audience snapshot from ClickHouse.
7. Selector workers enqueue delivery tasks into Redpanda.
8. Sender workers consume tasks from Redpanda and send Web Push notifications.
9. Sender and service worker events are written to ClickHouse.
10. Postback service receives conversions and writes normalized events to ClickHouse.
11. Reports are built from ClickHouse raw events and aggregates.

## 3. High-Level Architecture

Main services:

- `admin-api`: management API for admin frontend.
- `admin-frontend`: React admin panel.
- `public-api`: subscriber ingestion, VAPID public key and service-worker event ingestion endpoints.
- `payload-api`: low-latency service-worker payload resolution and final notification decisioning.
- `scheduler`: campaign launch planner.
- `audience-selector`: builds campaign audience snapshots and publishes delivery tasks.
- `creative-sync`: periodically imports creatives from built-in creative provider APIs.
- `sender`: Redpanda consumer that sends Web Push notifications.
- `postback-service`: incoming and outgoing postbacks.
- `analytics-api`: reporting endpoints over ClickHouse.
- `automation-service`: rules, alerts and operator automations.
- `cost-import-service`: manual/API traffic cost ingestion.
- `maintenance-workers`: compaction, cleanup, subscriber invalidation, aggregate rebuilds.

Infrastructure:

- PostgreSQL for operational data.
- ClickHouse for subscribers, events, statistics and audience snapshots.
- Redpanda for delivery task streaming.
- Redis for short-lived state, rate limits, caps, locks and idempotency.

## 4. Storage Strategy

### 4.1 PostgreSQL

PostgreSQL stores operational and relational data:

- users;
- roles;
- publisher accounts;
- subscription sources;
- API keys;
- campaigns;
- creatives;
- creative provider configurations;
- DSP configurations;
- rates;
- payments;
- postback configurations;
- outgoing postback rules;
- system settings;
- audit log;
- campaign launch metadata;
- automation rules;
- alert subscriptions;
- operator notes;
- templates;
- replay job metadata.

PostgreSQL should not store the massive subscriber base as the primary selection store.

### 4.2 ClickHouse

ClickHouse stores:

- current subscriber delivery view;
- subscriber event history;
- campaign audience snapshots;
- push delivery events;
- impression/click/close events;
- unsubscribe/invalid endpoint events;
- postback/conversion events;
- traffic cost events;
- hourly and daily aggregates.

ClickHouse is used for audience selection because campaign delivery requires fast filtering over tens or hundreds of millions of rows.

### 4.3 Redpanda

Redpanda is required.

Primary use cases:

- decouple audience selection from sending;
- buffer delivery tasks;
- scale sender workers horizontally;
- support retries and DLQ;
- preserve delivery progress during sender restarts;
- provide backpressure between selector and sender.

### 4.4 Redis

Redis is used for:

- rate limiting;
- frequency caps;
- campaign locks;
- short-lived idempotency keys;
- refresh/session cache if needed;
- per-provider throttling state;
- temporary sender coordination.

## 5. Main Data Entities

### 5.1 User

Fields:

- `id`;
- `email`;
- `password_hash`;
- `status`;
- `role_id`;
- `created_at`;
- `updated_at`;
- `last_login_at`.

Roles:

- `admin`;
- `manager`;
- `finance`;
- `viewer`.

### 5.2 Publisher

Fields:

- `id`;
- `name`;
- `email`;
- `status`;
- `balance`;
- `api_key_id`;
- `postback_url`;
- `created_at`;
- `updated_at`.

### 5.2A Subscription Source

Subscription Source is a managed traffic/source/channel entity used to collect web-push subscribers and attribute downstream performance.

It replaces ad-hoc `channel` strings with admin-managed sources.

Fields:

- `id`;
- `publisher_id`;
- `name`;
- `slug`;
- `status`;
- `public_key`;
- `channel`;
- `subid_template`;
- `domain_allowlist`;
- `default_language`;
- `default_timezone`;
- `sdk_config`;
- `service_worker_config`;
- `post_subscribe_redirect_url`;
- `post_denied_redirect_url`;
- `created_at`;
- `updated_at`.

Requirements:

- sources are created and managed from admin panel;
- each source has a stable public identifier used by JS SDK;
- each source can generate an install snippet;
- source-level settings can control channel, subid, redirects and SDK behavior;
- source is stored on every subscriber as `source_id`;
- reports must support source-level subscriber, delivery and profit statistics.

### 5.3 Campaign

Fields:

- `id`;
- `name`;
- `description`;
- `type`;
- `status`;
- `publisher_id`, optional;
- `creative_selection_mode`;
- `creative_source_type`;
- `creative_provider_config_id`, optional;
- `creative_sync_interval`;
- `creative_sync_status`;
- `schedule`;
- `timezone_mode`;
- `filters`;
- `caps`;
- `frequency_caps`;
- `created_at`;
- `updated_at`.

Types:

- `self_served`;
- `dsp`;
- `custom`.

Statuses:

- `draft`;
- `active`;
- `paused`;
- `completed`;
- `archived`.

Creative selection modes:

- `fixed`;
- `weighted_random`;
- `round_robin`;
- `profit_ranked_rotation`;
- `adaptive_profit_weighted`.

Creative source types:

- `manual`;
- `provider_api`.

### 5.4 Creative

Fields:

- `id`;
- `campaign_id`;
- `name`;
- `external_id`;
- `source_type`;
- `provider_name`;
- `title`;
- `body`;
- `icon_url`;
- `image_url`;
- `badge_url`;
- `target_url`;
- `buttons`;
- `vibration`;
- `status`;
- `weight`;
- `priority`;
- `total_cap`;
- `daily_cap`;
- `daily_cap_per_subscription`;
- `total_cap_per_subscription`;
- `valid_from`;
- `valid_to`;
- `targeting_overrides`;
- `raw_provider_payload`;
- `last_synced_at`;
- `created_at`;
- `updated_at`.

### 5.4A Creative Provider Config

Creative provider configs describe built-in API integrations that can automatically synchronize creatives into a campaign.

Fields:

- `id`;
- `name`;
- `provider_type`;
- `status`;
- `credentials_ref`;
- `endpoint_config`;
- `request_mapping`;
- `response_mapping`;
- `default_sync_interval`;
- `last_sync_at`;
- `last_success_at`;
- `last_error`;
- `created_at`;
- `updated_at`.

Provider configs must not store plain secrets in database fields. Store secret references or encrypted values according to the deployment secret strategy.

Built-in provider implementations are compiled into the backend. Each provider must normalize its API response into the internal `Creative` model.

### 5.4B Operator Tooling Entities

The product is an internal SaaS for operators, teams or a single media buyer. It must include productivity and decision-making entities, not only delivery entities.

Required operational entities:

- `CostImport`: traffic spend imported manually, by CSV or later by ad network API.
- `AutomationRule`: if/then rules for pausing, scaling, alerting and limiting.
- `AlertSubscription`: Telegram/email/webhook alert targets and preferences.
- `OperatorNote`: notes attached to sources, campaigns, creatives, launches or providers.
- `Template`: reusable source, campaign, postback, provider and cap presets.
- `ReplayJob`: controlled recovery/replay tasks for failed sends, postbacks and aggregates.
- `Segment`: reusable audience definition for campaigns and exclusions.

These entities are admin/operator features. They are not exposed as public publisher self-service by default.

### 5.5 Subscriber

Subscriber delivery data is stored in ClickHouse.

Required fields:

- `subscriber_id`;
- `publisher_id`;
- `source_id`;
- `endpoint`;
- `auth`;
- `p256dh`;
- `endpoint_hash`;
- `active`;
- `country`;
- `region`;
- `city`;
- `channel`;
- `subid`;
- `language`;
- `platform`;
- `browser`;
- `device_type`;
- `timezone`;
- `ip`;
- `user_agent`;
- `page_url`;
- `subscribed_at`;
- `updated_at`;
- `version`.

The unique Web Push fields `endpoint`, `auth` and `p256dh` are stored directly in ClickHouse string columns and are used by sender workers.

### 5.6 PostbackConfig

Fields:

- `id`;
- `name`;
- `publisher_id`, optional;
- `campaign_id`, optional;
- `token`;
- `signature_secret`, optional;
- `method`;
- `parameter_mapping`;
- `status`;
- `created_at`;
- `updated_at`.

Supported mapped fields:

- `click_id`;
- `campaign_id`;
- `publisher_id`;
- `subscriber_id`;
- `external_id`;
- `event_type`;
- `payout`;
- `currency`;
- `status`;
- `raw_payload`.

## 6. ClickHouse Tables

### 6.1 `subscribers_current`

Purpose: current delivery view used by audience selector.

Example schema:

```sql
CREATE TABLE subscribers_current
(
    subscriber_id UUID,
    publisher_id UUID,
    source_id UUID,

    endpoint String,
    auth String,
    p256dh String,
    endpoint_hash FixedString(32),

    active UInt8,

    country LowCardinality(String),
    region String,
    city String,
    channel LowCardinality(String),
    subid String,
    language LowCardinality(String),
    platform LowCardinality(String),
    browser LowCardinality(String),
    device_type LowCardinality(String),
    timezone LowCardinality(String),

    ip String,
    user_agent String,
    page_url String,

    subscribed_at DateTime,
    updated_at DateTime,
    version UInt64
)
ENGINE = ReplacingMergeTree(version)
PARTITION BY toYYYYMM(subscribed_at)
ORDER BY (active, country, channel, timezone, publisher_id, subscriber_id);
```

Notes:

- Avoid frequent row updates.
- Use append/versioned writes.
- `active = 0` is written as a new version when a subscriber becomes invalid.
- Avoid hot `FINAL` queries in campaign delivery paths when possible.
- Consider periodic compaction or a rebuilt current view for very large deployments.

### 6.2 `subscriber_events`

Append-only history:

- `subscribed`;
- `resubscribed`;
- `unsubscribed`;
- `invalidated`;
- `updated`.

### 6.3 `campaign_audience`

Purpose: fixed recipient snapshot for a campaign launch.

Example fields:

- `launch_id`;
- `campaign_id`;
- `subscriber_id`;
- `endpoint`;
- `auth`;
- `p256dh`;
- `country`;
- `channel`;
- `timezone`;
- `shard`;
- `created_at`.

The `shard` field is calculated as:

```sql
cityHash64(subscriber_id) % 256
```

### 6.4 `push_events`

Append-only event table:

- `delivery_enqueued`;
- `sent`;
- `success`;
- `fail`;
- `retry_scheduled`;
- `dlq`;
- `invalid_endpoint`;
- `impression`;
- `click`;
- `close`;
- `unsubscribe`.

Required dimensions:

- `event_time`;
- `event_type`;
- `launch_id`;
- `campaign_id`;
- `creative_id`;
- `publisher_id`;
- `source_id`;
- `subscriber_id`;
- `country`;
- `channel`;
- `device_type`;
- `browser`;
- `platform`;
- `provider`;
- `error_code`;
- `delivery_id`.

### 6.5 `postback_events`

Fields:

- `event_time`;
- `postback_config_id`;
- `campaign_id`;
- `publisher_id`;
- `source_id`;
- `subscriber_id`;
- `click_id`;
- `external_id`;
- `event_type`;
- `payout`;
- `currency`;
- `status`;
- `dedupe_key`;
- `raw_payload`.

### 6.6 `creative_exposure_state`

Purpose: compact per-subscriber state for creative rotation within a campaign.

This table is used to avoid showing the same creative repeatedly to the same subscription before the subscriber has seen all eligible creatives in the current rotation cycle.

Example fields:

- `campaign_id`;
- `subscriber_id`;
- `creative_id`;
- `cycle`;
- `shown_at`;
- `event_type`;
- `version`.

Recommended engine:

```sql
CREATE TABLE creative_exposure_state
(
    campaign_id UUID,
    subscriber_id UUID,
    creative_id UUID,
    cycle UInt32,
    shown_at DateTime,
    event_type LowCardinality(String),
    version UInt64
)
ENGINE = ReplacingMergeTree(version)
PARTITION BY cityHash64(campaign_id) % 64
ORDER BY (campaign_id, subscriber_id, cycle, creative_id);
```

Notes:

- The authoritative history remains in `push_events`.
- This table is a compact operational view for rotation decisions.
- A cycle is complete when the subscriber has seen all currently eligible creatives for the campaign.
- After a full cycle, the next exposure starts `cycle + 1` and the creative history is treated as reset for selection purposes.

### 6.7 `creative_performance_hourly`

Purpose: fast profit-based ranking for creative selection.

Suggested fields:

- `hour`;
- `campaign_id`;
- `creative_id`;
- `sent`;
- `impressions`;
- `clicks`;
- `conversions`;
- `revenue`;
- `cost`;
- `profit`;
- `ctr`;
- `cvr`;
- `epc`;
- `score`.

The score is derived from postback and delivery statistics. The exact formula must be configurable.

### 6.8 Aggregates

Create materialized views for:

- campaign hourly stats;
- campaign daily stats;
- publisher hourly stats;
- publisher daily stats;
- subscription source hourly stats;
- subscription source daily stats;
- country/channel/device breakdowns;
- postback/conversion stats;
- creative performance stats;
- sender/provider health stats.

## 7. Audience Selection

Audience selection must support filters:

- country;
- region;
- city;
- channel;
- subid;
- language;
- platform;
- browser;
- device type;
- timezone;
- subscription age;
- subscription date range;
- active status;
- prior send/click/conversion exclusions;
- frequency cap exclusions.

Direct cursor selection is allowed for small and medium launches.

For large launches, the required flow is audience snapshot:

```sql
INSERT INTO campaign_audience
SELECT
    {launch_id} AS launch_id,
    {campaign_id} AS campaign_id,
    subscriber_id,
    endpoint,
    auth,
    p256dh,
    country,
    channel,
    timezone,
    cityHash64(subscriber_id) % 256 AS shard,
    now() AS created_at
FROM subscribers_current
WHERE active = 1
  AND ...
```

Benefits:

- stable campaign recipient set;
- resumable delivery;
- parallel shard processing;
- easier progress tracking;
- no drifting audience during long campaigns.

## 8. Delivery Pipeline

### 8.1 Campaign Launch

1. Admin starts or schedules campaign.
2. Scheduler creates `campaign_launch`.
3. Audience selector builds `campaign_audience`.
4. Creative selector assigns `creative_id` for every delivery task.
5. Selector publishes tasks to Redpanda.
6. Sender workers consume and send pushes.
7. Results are written to ClickHouse.

## 8A. Creative Selection And Rotation

Creative selection is part of delivery task preparation. Sender workers should receive a resolved `creative_id` in the Redpanda task and should not make random creative decisions during retry.

This guarantees:

- stable retry behavior;
- reproducible campaign delivery;
- accurate creative-level statistics;
- simpler sender workers;
- deterministic per-subscriber rotation.

### 8A.1 Eligible Creative Set

Before selecting a creative, build the eligible creative set for the campaign and subscriber.

A creative is eligible when:

- creative status is active;
- campaign status is active;
- current time is within `valid_from` and `valid_to`, if configured;
- creative caps are not exhausted;
- creative targeting overrides match the subscriber, if configured;
- the creative belongs to the campaign;
- the creative is not excluded by compliance or manual rules.

### 8A.2 Per-Subscriber Exposure History

The system must remember which creatives were already shown to each subscriber for each campaign.

Required behavior:

- if a subscriber has not seen all eligible creatives in the current cycle, select only from unseen creatives;
- if a subscriber has seen all eligible creatives, start a new cycle and allow all eligible creatives again;
- cycle reset is per `campaign_id + subscriber_id`;
- exposure is recorded when the notification is considered shown or sent, depending on campaign configuration.

Recommended default:

- record exposure on `sent` for strict delivery rotation;
- additionally track `impression` for real display analytics.

If exposure is recorded only on `impression`, users who receive but never display notifications may repeatedly receive the same creative. This may be useful for some campaigns but should not be the default.

### 8A.3 Selection Modes

#### Fixed

Always select the configured creative.

If the fixed creative is not eligible, use fallback behavior:

- skip delivery; or
- use campaign fallback creative.

The fallback behavior is campaign-configurable.

#### Weighted Random

Select from unseen eligible creatives using configured weights.

Selection must be deterministic for a delivery:

```text
hash(launch_id + subscriber_id + cycle) % total_weight
```

This avoids changing the creative during retry or Redpanda rebalance.

#### Round Robin

Select creatives in a stable sequence.

For large audiences, use deterministic distribution:

```text
cityHash64(subscriber_id + cycle) % eligible_creatives_count
```

When exposure history excludes already shown creatives, apply round-robin only to the unseen set.

#### Profit Ranked Rotation

Use postback and delivery performance to rank creatives from best to worst, then rotate through them in that order.

Behavior:

1. Calculate creative performance score from recent statistics.
2. Sort eligible creatives by score descending.
3. For each subscriber, select the highest-ranked creative not yet shown in the current cycle.
4. After the subscriber has seen all eligible creatives, start a new cycle.
5. In the next cycle, use the latest ranking, so the order may change as postback data changes.

This mode matches the requirement: after initial rotation/testing, show the most profitable creatives first, then continue from best to worst, while still preventing repeated exposure until the subscriber has gone through the full set.

#### Adaptive Profit Weighted

Use performance score to generate dynamic weights instead of a strict ranking.

Example:

- high-profit creatives receive higher weight;
- low-profit creatives still receive some traffic for continued testing;
- new creatives receive exploration traffic until enough data is collected.

This mode is useful when strict best-to-worst ordering would starve new creatives too quickly.

### 8A.4 Performance Score

Creative score must be configurable per campaign or globally.

Possible inputs:

- postback revenue;
- payout;
- cost;
- profit;
- conversions;
- CTR;
- CVR;
- EPC;
- fail rate;
- unsubscribe rate;
- minimum sample size.

Example formula:

```text
score = profit * 0.60 + revenue * 0.20 + conversions * 0.10 + ctr * 0.10
```

The actual formula must avoid over-optimizing on tiny samples. Use minimum sample thresholds and exploration rules.

### 8A.5 Cold Start And Exploration

New creatives without enough data must receive test traffic.

Recommended behavior:

- reserve an exploration percentage, for example 5-15%;
- distribute exploration traffic among new or under-sampled creatives;
- after minimum sample size is reached, include the creative in normal profit ranking;
- allow manual priority boost for new creatives.

### 8A.6 Exposure State Update

For every delivery task, write or enqueue an exposure event:

- `campaign_id`;
- `subscriber_id`;
- `creative_id`;
- `launch_id`;
- `delivery_id`;
- `cycle`;
- `event_type`;
- `event_time`.

The compact `creative_exposure_state` table is updated from these events.

For very high throughput:

- write raw events first;
- update compact state asynchronously;
- tolerate eventual consistency within a launch;
- use deterministic task assignment to reduce duplicate creative selection.

### 8A.7 Concurrency And Idempotency

Creative selection must be idempotent.

Use:

```text
delivery_id = hash(campaign_id + launch_id + subscriber_id)
```

If the same delivery task is retried, it must keep the same `creative_id`.

If two selectors process the same subscriber by mistake, dedupe by `delivery_id` and record only one exposure.

### 8A.8 Reporting

Reports must support creative-level breakdowns:

- sent;
- success;
- fail;
- impressions;
- clicks;
- conversions;
- revenue;
- cost;
- profit;
- CTR;
- CVR;
- EPC;
- unsubscribe rate;
- score;
- rank history.

## 8B. Creative Provider API Sync

Campaign creatives can be managed manually or synchronized from built-in creative provider APIs.

This is controlled by campaign fields:

- `creative_source_type = manual`;
- `creative_source_type = provider_api`.

When `provider_api` is selected, the campaign references a `creative_provider_config_id` and a sync interval.

### 8B.1 Provider Model

Creative providers are backend integrations implemented in code.

Each provider must expose a common interface:

```text
FetchCreatives(ctx, providerConfig, campaignConfig) -> []NormalizedCreative
```

Provider responsibilities:

- authenticate against the source API;
- request available creatives;
- handle provider-specific pagination;
- handle provider-specific rate limits;
- normalize the response into internal creative fields;
- preserve raw provider payload for debugging;
- return stable `external_id` values.

Examples of provider-specific differences:

- different auth methods;
- different endpoint URLs;
- different creative schemas;
- different image/icon fields;
- different click URL macros;
- different payout/revenue fields;
- different status values.

### 8B.2 Sync Scheduling

The `creative-sync` service periodically synchronizes provider-backed campaigns.

Required behavior:

- each provider-backed campaign has `creative_sync_interval`;
- scheduler enqueues sync jobs;
- sync jobs are idempotent;
- concurrent sync for the same campaign is prevented with a Redis lock or PostgreSQL advisory lock;
- sync results are stored in audit/sync logs;
- failed sync does not stop the currently active campaign creatives.

Minimum interval must be configurable globally to avoid provider abuse.

### 8B.3 Upsert And Deduplication

Provider creatives are deduplicated by:

```text
campaign_id + provider_name + external_id
```

Sync behavior:

- new provider creative creates an internal `Creative`;
- existing provider creative updates normalized fields;
- missing provider creative is not deleted immediately;
- missing provider creative can be marked `inactive` after configurable grace period;
- manual edits to provider-owned fields may be overwritten on next sync;
- manual override fields must be separated if needed.

Recommended fields:

- `source_type = provider_api`;
- `provider_name`;
- `external_id`;
- `raw_provider_payload`;
- `last_synced_at`;
- `sync_status`.

### 8B.4 Validation

Every synced creative must be validated before activation:

- title is present;
- body is present if required;
- target URL is valid;
- icon/image URLs are valid or fetchable if validation is enabled;
- provider status allows serving;
- campaign-level compliance rules pass.

Invalid creatives are stored with inactive/error status and visible in admin UI.

### 8B.5 Interaction With Creative Selection

Provider-synced creatives participate in the same selection and rotation modes as manual creatives:

- fixed;
- weighted random;
- round-robin;
- profit-ranked rotation;
- adaptive profit weighted.

For provider-backed campaigns, the default selection mode should be `profit_ranked_rotation` or `adaptive_profit_weighted`.

Per-subscriber exposure history applies equally to provider-synced creatives.

If a provider sync adds new creatives:

- new creatives enter the eligible set;
- cold-start exploration rules allocate test traffic;
- subscribers who already completed a cycle can see them in the next cycle;
- subscribers mid-cycle can receive newly added creatives if campaign config allows dynamic cycle expansion.

Dynamic cycle expansion is configurable:

- `strict_cycle`: finish the current creative set before considering newly synced creatives;
- `dynamic_cycle`: newly synced creatives can be added to the subscriber's current unseen set.

### 8B.6 Provider Creative Caps And Status

Provider APIs may return their own caps, status or availability.

The normalized creative model must support:

- provider status;
- internal status;
- provider cap fields, optional;
- internal cap overrides;
- last provider update time.

Internal status has final authority. Admins can pause a provider creative locally even if the provider marks it active.

### 8B.7 Admin UI Requirements

Admin panel must support:

- creating creative provider configs;
- selecting a creative provider in campaign form;
- setting sync interval;
- running manual sync;
- viewing sync history;
- viewing provider errors;
- viewing raw provider payload for debugging;
- enabling/disabling synced creatives locally;
- showing whether a creative is manual or provider-synced.

### 8.2 Redpanda Topics

Required topics:

- `push.delivery.tasks`;
- `push.delivery.retry.1m`;
- `push.delivery.retry.10m`;
- `push.delivery.retry.1h`;
- `push.delivery.dlq`;
- `push.delivery.results`, optional if results are not written directly to ClickHouse;
- `subscriber.invalidations`, optional;
- `postback.events`, optional.

Start with 64 partitions for delivery tasks. For higher traffic, use 128 or 256 partitions.

### 8.3 Delivery Task Message

Key:

```text
campaign_id:subscriber_id
```

or:

```text
subscriber_id
```

Value:

```json
{
  "delivery_id": "hash(campaign_id + launch_id + subscriber_id)",
  "launch_id": "uuid",
  "campaign_id": "uuid",
  "campaign_version": 12,
  "creative_id": "uuid",
  "subscriber_id": "uuid",
  "endpoint": "https://...",
  "auth": "...",
  "p256dh": "...",
  "country": "US",
  "channel": "news",
  "timezone": "America/New_York"
}
```

Campaign and creative payloads should be cached by sender workers using `campaign_id` and `campaign_version`. Do not duplicate the full campaign payload in every task unless there is a clear operational reason.

### 8.3A Trigger-Only Web Push

Web Push messages sent by sender workers must be trigger-only.

The encrypted push sent to the browser must not contain the final notification title, body, URL, image, creative payload or campaign decision payload. It may be empty or contain only a minimal opaque trigger/version marker required by push provider compatibility.

When the browser receives the push, the service worker must:

1. read the opaque trigger marker from the push event, or the local compatibility trigger value stored after subscribe;
2. call the payload resolution API;
3. let the backend decide the final notification payload at display time;
4. show the notification returned by the backend;
5. report push, notification shown, click and close events with the resolved delivery metadata.

Payload resolution is server-side because the final notification can depend on:

- subscriber endpoint and source;
- campaign state and caps at receive time;
- geo, IP-derived attributes and user agent;
- device/browser capability;
- source and publisher settings;
- creative availability, compliance rules and rotation state;
- fraud/risk rules and suppression lists.

This keeps delivery tasks and sender workers simple. Sender workers wake eligible subscribers; they do not make final creative or message decisions during retry.

Payload resolution is served by `payload-api`, not by `admin-api` or subscriber ingestion. The service worker must resolve payloads by `trigger_id`, not by the Web Push endpoint. The endpoint is transport data for sender workers and must not be the browser-facing subscriber lookup key.

Target chain:

```text
trigger_id -> delivery_id -> subscription_id -> source_id/campaign/caps/creative
```

Trigger context is short-lived hot-path state and is stored in Redis with TTL. Sender workers create opaque trigger IDs and store server-side trigger context before sending trigger-only push messages. Local compatibility paths may temporarily use the platform `subscription_id` as `trigger_id`.

Internal trigger creation shape:

```http
POST /v1/push/triggers
Content-Type: application/json

{
  "subscription_id": "uuid",
  "source_id": "uuid",
  "campaign_id": "uuid",
  "ttl_seconds": 300
}
```

Response:

```json
{
  "trigger_id": "opaque-id",
  "delivery_id": "uuid",
  "subscription_id": "uuid",
  "source_id": "uuid",
  "campaign_id": "uuid",
  "expires_at": "timestamp"
}
```

Current creative decisioning is intentionally simple:

- if trigger context contains `campaign_id`, check active creatives from that active campaign;
- otherwise check active creatives from an active campaign attached to `source_id`;
- query ClickHouse exposure history for `subscription_id + campaign_id` within the configured exposure window;
- try creatives not yet shown in the current exposure window first;
- when all active creatives were shown in the window, start a new cycle and allow the full active set again;
- load subscriber targeting snapshot from ClickHouse when campaign targeting rules are configured;
- suppress campaigns whose targeting rules do not match the subscriber;
- return the first creative that passes campaign and creative caps;
- if a creative cap is exhausted, try the next active creative;
- return `404` when no active campaign/creative is eligible;
- later selection modes replace this simple exposure-aware stable-order rule.

Endpoint shape:

```http
POST /v1/push/payload
Content-Type: application/json

{
  "trigger_id": "opaque-id"
}
```

Response:

```json
{
  "title": "string",
  "body": "string",
  "icon": "https://...",
  "url": "https://...",
  "source_id": "uuid",
  "campaign_id": "uuid",
  "creative_id": "uuid",
  "delivery_id": "uuid"
}
```

Caps:

- `daily_cap_per_subscription = 0` means unlimited daily exposures;
- `total_cap_per_subscription = 0` means unlimited total exposures;
- campaign caps are stored by `subscription_id + campaign_id`;
- creative caps are stored by `subscription_id + campaign_id + creative_id`;
- if a creative cap is exceeded, `payload-api` tries the next active creative;
- if campaign caps or all eligible creative caps are exceeded, `payload-api` returns `204 No Content` and no notification payload.

Exposure tracking:

- `creative_exposures` is append-only ClickHouse history;
- rows are written after a creative is selected for a payload response;
- the exposure window defaults to 24 hours;
- this history is used to avoid repeating a creative for a subscription before the active creative set has completed a cycle.

Campaign targeting:

- campaign `targeting_rules` is PostgreSQL JSONB operational config;
- targeting allow-lists are `countries`, `languages`, `device_types`, `os_names` and `browser_names`;
- empty lists mean unrestricted;
- payload decisioning compares rules with the latest subscriber targeting snapshot in ClickHouse;
- language matching accepts exact values and base language values, for example `en` matches `en-US`;
- targeting mismatch returns no payload and records `campaign_targeting_mismatch`.

Payload decision observability:

- `payload_decisions` is append-only ClickHouse debug history;
- rows contain `trigger_id`, `subscription_id`, `source_id`, `campaign_id`, selected `creative_id`, result, reason and error text;
- final notification title, body, URL, icon and creative payload are not stored in this table;
- decision results are `selected`, `suppressed`, `not_found` and `error`;
- decision reasons include `selected`, `source_lookup_failed`, `creative_lookup_failed`, `no_active_creative`, `targeting_lookup_failed`, `campaign_targeting_mismatch`, `exposure_lookup_failed`, `cap_check_failed`, `all_eligible_creatives_capped` and `exposure_record_failed`.

The endpoint must be low latency, observable and cache-aware, but it must not rely on client-side trust for targeting decisions.

### 8.4 Sender Mechanics

Sender workers must support:

- horizontal scaling through Redpanda consumer groups;
- batch consume;
- controlled concurrency per instance;
- per-provider throttling;
- retry with backoff;
- DLQ for fatal errors;
- idempotent delivery IDs;
- Web Push VAPID signing;
- invalid endpoint detection;
- structured result events.

Provider-aware throttling should group by endpoint host, for example:

- FCM;
- Mozilla Autopush;
- Apple/WebKit push;
- other providers.

## 9. Batching Rules

Never use `OFFSET` for massive audience scans.

Use keyset/cursor batching:

```sql
SELECT subscriber_id, endpoint, auth, p256dh
FROM campaign_audience
WHERE launch_id = {launch_id}
  AND shard = {shard}
  AND subscriber_id > {last_subscriber_id}
ORDER BY subscriber_id
LIMIT 10000;
```

Repeat until an empty result is returned.

For parallel processing:

- split by `shard`;
- each selector worker owns one or more shards;
- store shard progress in campaign launch state;
- resume from the last cursor after restart.

## 10. Caps And Rate Limits

Campaigns must support:

- total campaign cap;
- daily campaign cap;
- hourly campaign cap;
- publisher cap;
- country cap;
- channel cap;
- per-subscriber frequency cap;
- provider throughput limits.

Recommended approach:

- apply broad exclusions in ClickHouse during audience snapshot;
- enforce short-window caps in Redis;
- write all delivery events to ClickHouse;
- use ClickHouse for future exclusion queries and reporting.

## 11. Public API

### 11.1 Subscribe

Endpoint:

```http
POST /v1/subscribe
```

Payload:

```json
{
  "publisher_key": "...",
  "source_id": "...",
  "subscription_id": "uuid returned by the platform",
  "channel": "news",
  "subid": "subid_1",
  "page_url": "https://example.com",
  "timezone": "Europe/Moscow",
  "language": "ru-RU",
  "user_agent": "...",
  "endpoint": "https://...",
  "auth": "...",
  "p256dh": "..."
}
```

Behavior:

- validate publisher key;
- validate subscription source and domain allowlist;
- derive IP from trusted proxy headers;
- parse user agent;
- enrich geo;
- calculate endpoint hash;
- generate a platform `subscription_id` for the subscriber;
- write a new subscriber version;
- write `subscriber_events`;
- update `subscribers_current`.

### 11.2 Tracking

Endpoints:

```http
GET /v1/events/impression/{delivery_id}
GET /v1/events/click/{delivery_id}
GET /v1/events/close/{delivery_id}
GET /v1/events/unsubscribe/{delivery_id}
```

Requirements:

- accept no-cors browser calls where needed;
- write raw event to ClickHouse;
- support idempotency for repeated browser events;
- redirect click to target URL if required by product flow.

## 12. JS SDK And Service Worker

Provide:

- versioned JS SDK;
- versioned service worker template;
- admin-generated install snippet per subscription source;
- publisher configuration generator;
- VAPID public key injection;
- channel/subid support;
- safe event tracking;
- graceful handling when push is unavailable.

The SDK should not contain private keys or internal secrets.

The service worker must treat push events as triggers only. It must not expect the Web Push event payload to contain the final notification. The final payload is fetched from the backend payload resolution API at push receive time.

## 12A. Subscription Sources And Script Generation

Subscription sources are managed from the admin panel and define how subscribers are collected from a specific site, channel, landing page, placement or traffic source.

### 12A.1 Admin Management

Admin panel must support:

- source CRUD;
- source status: active, paused, archived;
- binding source to publisher;
- source public key generation;
- VAPID key management and binding to sources;
- channel and subid defaults;
- domain allowlist;
- SDK behavior settings;
- service worker settings;
- redirect settings after subscribe/deny/error;
- generated installation snippet;
- source-level statistics and profit reports.

### 12A.1A VAPID Key Management

VAPID keys must become first-class admin-managed entities.

Admin panel must support:

- generating VAPID key pairs;
- listing VAPID public keys and metadata;
- storing private keys only server-side;
- binding one active VAPID key pair to one or more subscription sources;
- rotating source VAPID keys with an explicit migration/re-subscription strategy;
- marking keys as active, deprecated or revoked;
- auditing who generated, attached, detached or revoked a key.

Subscription source config must expose the active VAPID public key to the SDK. The runtime path is `GET /api/sdk/config?source_id=...`, which resolves the active source VAPID public key and public endpoints. Generated snippets may inline the active source VAPID public key to remove one network round-trip. `GET /api/web-push/vapid-public-key` may remain as a fallback endpoint for local compatibility.

Private VAPID keys must never be sent to browsers or publishers. Sender services use private keys for Web Push signing; public-api/SDK only expose public keys.

### 12A.2 Generated Snippet

For every source, admin panel generates a copy-paste snippet.

Example:

```html
<script
  src="https://cdn.example.com/sdk/push-booster.js"
  data-source-id="src_xxx"
  data-public-key="pub_xxx"
  data-channel="news"
  data-subid="{SUBID}">
</script>
```

The exact snippet format can evolve, but it must include:

- SDK URL;
- source identifier;
- public source key;
- active source VAPID public key, inline or resolvable from config;
- optional channel override;
- optional subid macro;
- optional debug flag for testing.

### 12A.3 SDK Config Endpoint

The SDK may load source config from public API:

```http
GET /api/sdk/config?source_id={source_id}
```

Response includes:

- VAPID public key;
- subscribe endpoint;
- service worker event endpoint;
- push payload resolution endpoint;
- service worker URL;
- source settings;
- redirects;
- feature flags.

This allows changing source behavior from admin without requiring publishers to replace the snippet.

Public SDK endpoints must validate browser `Origin`/`Referer` against the subscription source domain allowlist before returning source config or accepting browser-originated subscribe/event writes. Requests without browser origin headers may be accepted for controlled server-side/internal tooling, but browser requests from foreign domains must be rejected.

Subscribe ingestion returns a platform `subscription_id`. Service workers must store this value for event reporting. Subscriber identity, source stats, caps and downstream attribution must use `subscription_id`; the Web Push `endpoint` remains delivery transport data and must not be used as the subscriber lookup fallback. Payload resolution uses `trigger_id`, which resolves server-side to `subscription_id`. Initial attribution fields include `subid`, `channel`, `landing_url` and `referrer`.

Subscriber targeting attributes must be normalized before writing to ClickHouse. The subscriber table stores IP, geo placeholders, language, browser, OS, device and Client Hints fields. Raw User-Agent must not be stored in `subscribers`; it may be used transiently during ingestion to derive normalized attributes.

### 12A.4 Service Worker Generation

The system must support either:

- a generic service worker with source config loaded at runtime; or
- generated service worker files per source.

Preferred approach:

- generic versioned service worker;
- source-specific config delivered by API.
- trigger-only push handling with backend payload resolution.

Generated service workers can be added later if provider/domain constraints require it.

### 12A.5 Source Attribution

Every subscription and all downstream events must be attributable to `source_id`. Browser-originated service worker events may send only `subscription_id`; backend services resolve `source_id` from subscriber metadata.

Attribution chain:

```text
source_id -> subscriber_id -> delivery_id -> click_id -> postback/conversion
```

This enables source-level reporting:

- subscriptions;
- unique subscriptions;
- unsubscribes;
- active subscribers;
- sent;
- impressions;
- clicks;
- conversions;
- revenue;
- cost;
- profit;
- profit per subscriber;
- profit per unique subscriber;
- conversion rate by source.

### 12A.6 Source-Level Profit

Profit from postbacks must be attributable to the original subscription source.

When a postback arrives:

1. resolve `click_id`, `delivery_id` or `subscriber_id`;
2. resolve `source_id` from subscriber/delivery metadata;
3. write `source_id` into `postback_events`;
4. update source-level ClickHouse aggregates.

If attribution cannot be resolved, write the postback with empty source and expose it in unresolved attribution reports.

### 12A.7 Source Reports

Admin source reports must show:

- subscribed count;
- unique subscribed count;
- unsubscribed count;
- active subscriber count;
- delivery stats generated by subscribers from this source;
- postback/conversion stats generated by subscribers from this source;
- total revenue;
- total cost;
- total profit;
- profit per subscriber;
- profit per unique subscriber;
- breakdown by country, channel, subid, device, browser and campaign.

## 13. Admin Frontend

Frontend stack:

- React;
- TypeScript;
- Vite;
- Tailwind CSS;
- shadcn/ui style conventions;
- React Router;
- TanStack Query;
- TanStack Table;
- Recharts;
- Zustand;
- react-hook-form;
- zod.

Required sections:

- login;
- arbitrage cockpit;
- dashboard;
- publishers;
- subscription sources;
- campaigns;
- creatives;
- creative providers;
- cost import;
- cohort LTV;
- DSPs;
- rates;
- payments;
- subscribers;
- reports;
- postbacks;
- postback inbox;
- rules and alerts;
- replay tools;
- templates;
- users and roles;
- system settings.

UI requirements:

- dense operational interface;
- tables with filters, sorting and pagination;
- CSV export;
- clear form validation;
- audit-visible destructive actions;
- no marketing-style landing page inside admin.

## 14. Admin API

Requirements:

- REST API;
- latest available stable Go toolchain;
- Go services use `net/http` with `chi` as the default HTTP router;
- OpenAPI specification;
- email OTP login;
- JWT Bearer access token;
- local mode may return OTP in API response when explicitly enabled;
- RBAC;
- audit log;
- request validation;
- structured errors;
- pagination/filtering conventions;
- health/readiness endpoints.

Auth requirements:

- fixed JWT algorithm;
- `iss` and `aud` validation;
- short access token TTL;
- OTP TTL and request rate limit;
- admin email from environment is auto-approved;
- non-admin verified users remain pending until admin approval;
- logout/revocation support can be added after token sessions are in place.

## 15. Postbacks

### 15.1 Incoming Postbacks

Endpoint pattern:

```http
GET  /v1/postbacks/{postback_config_id}
POST /v1/postbacks/{postback_config_id}
```

Requirements:

- configurable parameter mapping;
- token or signature validation;
- raw payload logging;
- normalized event writing;
- deduplication;
- conversion attribution to click/campaign/publisher/subscriber when possible.

### 15.2 Outgoing Postbacks

Optional but should be designed:

- publisher-specific outgoing URLs;
- macro replacement;
- retry policy;
- result logging;
- DLQ for failed outgoing postbacks.

## 16. Reporting

Reports must support:

- campaign stats;
- publisher stats;
- subscription source stats;
- creative stats;
- channel stats;
- country stats;
- device/browser/platform stats;
- postback/conversion stats;
- sender/provider health;
- date range filtering;
- timezone-aware grouping;
- CSV export.

Metrics:

- enqueued;
- sent;
- success;
- fail;
- invalid endpoints;
- impressions;
- clicks;
- closes;
- CTR;
- conversions;
- payout;
- revenue;
- cost;
- profit if billing model is defined.
- source-level profit.

## 17. Security

Requirements:

- no hardcoded secrets;
- all secrets through env or secret manager;
- JWT best practices;
- API keys for publishers;
- postback token/signature;
- rate limits on public endpoints;
- CORS separation for admin and public APIs;
- audit log for critical admin actions;
- input validation on all endpoints;
- idempotency for delivery and postbacks;
- safe proxy IP handling;
- private VAPID keys only on sender side.

## 18. Observability

Every service must expose:

```http
GET /healthz
GET /readyz
GET /metrics
```

Logs:

- structured JSON logs;
- request ID;
- campaign ID;
- launch ID;
- delivery ID where applicable.

Metrics:

- Redpanda consumer lag;
- selector throughput;
- sender throughput;
- send success/fail rate;
- provider response codes;
- ClickHouse insert latency;
- API latency;
- postback ingest rate;
- DLQ rate.

## 19. Operations And Deployment

Local runtime:

- Docker Compose with PostgreSQL, ClickHouse, Redpanda, Redis and services;
- database migrations;
- seed data;
- local JS SDK demo page.

Production:

- separate Docker images per service;
- non-root containers;
- graceful shutdown;
- config through env;
- migrations as explicit deployment action;
- no automatic destructive migrations.

CI:

- Go tests;
- frontend tests/build;
- lint;
- static checks;
- migration validation;
- Docker image build.

## 19A. Internal SaaS Operator Tools

This product is intended for internal use by an operator team or a single media buyer. The UI should be optimized for fast decisions, traffic control and profit management rather than external self-service.

The following tools are required as product modules.

### 19A.1 Source Manager

Purpose: manage subscription sources and understand their quality and profitability.

Required features:

- source CRUD;
- generated JS snippet;
- domain allowlist;
- channel/subid defaults;
- SDK config;
- service worker config;
- opt-in funnel metrics;
- active subscribers;
- unsubscribes;
- source revenue;
- source cost;
- source profit;
- LTV by source;
- country/device/subid breakdown;
- quick actions: pause source, clone source, regenerate key.

Key metrics:

- prompt shown;
- permission accepted;
- permission denied;
- subscribed;
- unique subscribed;
- active subscribers;
- unsubscribed;
- opt-in rate;
- unsubscribe rate;
- revenue;
- cost;
- profit;
- ROI;
- profit per subscriber.

### 19A.2 Campaign Workspace

Purpose: single workspace for creating, launching and monitoring a campaign.

Required features:

- campaign settings;
- audience filters;
- estimated audience size;
- estimated send time;
- schedule;
- caps;
- frequency strategy;
- creative set;
- creative source mode: manual or provider API;
- postback binding;
- launch preview;
- quick actions: pause, resume, clone, scale, archive.

The workspace should show current launch progress:

- audience total;
- enqueued;
- sent;
- success;
- fail;
- impressions;
- clicks;
- conversions;
- revenue;
- cost;
- profit.

### 19A.3 Creative Lab

Purpose: manage and optimize manual and provider-synced creatives.

Required features:

- manual creative editor;
- provider-synced creative list;
- push notification preview;
- bulk import;
- status management;
- performance ranking;
- winner/loser labels;
- auto-pause candidates;
- unsubscribe/fatigue metrics;
- raw provider payload view;
- creative history by campaign and source.

Creative Lab must support the profit-ranked and adaptive rotation modes defined in `8A. Creative Selection And Rotation`.

### 19A.4 Postback Inbox And Debugger

Purpose: make postback ingestion observable and debuggable.

Required features:

- recent postbacks list;
- raw request viewer;
- normalized mapped fields;
- attribution status;
- duplicate/dedupe reason;
- unresolved postbacks;
- test postback generator;
- mapping preview;
- provider/source/campaign filters;
- retry or replay postback processing.

Attribution trace:

```text
source -> subscriber -> delivery -> click -> postback
```

### 19A.5 Cost Import

Purpose: capture traffic spend so ROI and profit are real.

Required features:

- manual spend entry;
- CSV import;
- cost by date;
- cost by source;
- cost by subid;
- cost by country;
- cost by campaign, optional;
- later API connectors for ad networks;
- validation and import preview.

ClickHouse should store cost events so reports can calculate:

- spend;
- revenue;
- profit;
- ROI;
- ROAS;
- payback period.

### 19A.6 Arbitrage Cockpit

Purpose: main decision dashboard for traffic arbitrage.

Required features:

- today/yesterday/7d overview;
- spend;
- revenue;
- profit;
- ROI;
- top winning sources;
- top losing sources;
- top creatives;
- top countries;
- postback health;
- sender health;
- quick actions from tables.

Quick actions:

- pause source;
- pause creative;
- pause campaign;
- reduce cap;
- increase cap;
- clone campaign;
- open trace/debug view.

### 19A.7 Cohort LTV

Purpose: understand long-term value of subscriber acquisition.

Required features:

- cohorts by subscription date;
- cohorts by source;
- cohorts by subid;
- cohorts by country;
- cohorts by device/browser;
- day 0, day 1, day 3, day 7 revenue;
- cumulative LTV;
- unsubscribe curve;
- payback period;
- cohort ROI.

Cohort reports must attribute revenue and profit back to the original `source_id` and subscription date.

### 19A.8 Rules And Alerts

Purpose: automate routine operator decisions.

Rules engine examples:

- pause source if ROI < X after spend > Y;
- pause creative if unsubscribe rate > X;
- reduce campaign cap if fail rate > X;
- increase cap if profit > X and ROI > Y;
- alert if postbacks drop to zero;
- alert if provider sync fails;
- alert if Redpanda lag grows;
- alert if ClickHouse insert latency grows.

Alert channels:

- Telegram;
- email;
- webhook.

Rules should support dry-run mode before enforcement.

### 19A.9 Replay And Recovery Tools

Purpose: allow safe operational recovery without manual database work.

Required features:

- requeue failed delivery tasks;
- rebuild campaign audience;
- replay postbacks;
- rebuild ClickHouse aggregates;
- process invalid subscriber events again;
- inspect one delivery by `delivery_id`;
- inspect one subscriber timeline;
- trace source/subscriber/delivery/click/postback chain.

Replay jobs must be auditable and idempotent.

### 19A.10 Templates

Templates are required to speed up repeat workflows:

- source templates;
- campaign templates;
- postback mapping templates;
- creative provider templates;
- cap/frequency templates;
- alert/rule templates.

Templates should be editable from admin UI and usable as starting points.

## 20. Product Scope

Core modules:

1. Admin JWT auth and RBAC.
2. Publishers CRUD.
3. Subscription sources CRUD and source script generation.
4. Campaigns CRUD.
5. Creatives CRUD.
6. Creative provider configs and provider-backed campaign creative sync.
7. Creative selection modes: fixed, weighted random, round-robin, profit-ranked rotation.
8. Per-subscriber creative exposure history and cycle reset.
9. Basic rates.
10. Public subscribe endpoint.
11. JS SDK and service worker.
12. ClickHouse `subscribers_current`.
13. Audience snapshot generation.
14. Redpanda `push.delivery.tasks`.
15. Sender workers.
16. Impression/click/close tracking.
17. ClickHouse raw events.
18. Daily campaign, publisher, source and creative aggregates.
19. Basic dashboard and reports.
20. Incoming postback endpoint with dedupe.
21. Docker Compose local environment.

Operator modules:

1. Source Manager with source profit reports.
2. Campaign Workspace.
3. Creative Lab.
4. Postback Inbox and debugger.
5. Cost Import.
6. Arbitrage Cockpit.
7. Cohort LTV.
8. Rules and Alerts.
9. Replay and Recovery Tools.
10. Templates.

Out of scope for the current product:

- complex billing;
- white label;
- ML optimization;
- advanced A/B testing;
- full BI report builder;
- multi-region deployment;
- complex outgoing postback retry UI.

## 21. Key Technical Decisions

Decisions already made:

- admin frontend uses React and TypeScript;
- backend uses the latest available stable Go toolchain;
- Go HTTP services use `net/http` with `chi` by default;
- auth uses email OTP with JWT Bearer sessions, not BasicAuth;
- operational data uses PostgreSQL;
- massive subscribers and analytics use ClickHouse;
- Redpanda is required;
- Redis is used for fast transient state;
- campaign delivery is based on audience snapshots and Redpanda tasks.

Open decisions:

- exact expected traffic: subscribes/sec, sends/day, events/sec;
- initial Redpanda partition count;
- exact ClickHouse cluster topology;
- whether to use direct ClickHouse inserts from services or event topics for all analytics;
- exact billing and payout model;
- first postback partners and required mapping formats;
- whether outgoing postbacks are included in the current product scope.

## 22. Data And Migration Risks

Historical data migration is not required unless later requested.

If migration is needed later:

- subscriber migration must be batched;
- legacy subscriber fields must be mapped carefully;
- historical stats should be backfilled into ClickHouse separately;
- duplicate subscribers must be handled by endpoint hash;
- legacy VAPID keys and exposed secrets should be treated as compromised and rotated.

## 23. Performance Targets

Recommended starting targets:

- subscriber base: 100M+ rows supported by design;
- audience snapshot generation: millions of rows per minute, depending on cluster size;
- Redpanda delivery topic: 64 partitions minimum for high-volume deployments;
- sender workers: horizontally scalable;
- delivery task batch size: configurable, default 10,000 records per selector query;
- ClickHouse event inserts: batched;
- API p95 latency targets defined per endpoint.

## 24. Engineering Principles

- Keep services small and explicit.
- Prefer append-only event writes for high-volume data.
- Avoid per-row updates in ClickHouse hot paths.
- Avoid `OFFSET` for large scans.
- Use keyset/cursor batching.
- Make campaign launches resumable.
- Make delivery idempotent.
- Keep secrets out of source code.
- Add migrations from the beginning.
- Add observability from the beginning.
