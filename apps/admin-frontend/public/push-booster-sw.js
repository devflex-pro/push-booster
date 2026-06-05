/* global Request, Response, caches, fetch, self */

const CONFIG_CACHE = "push-booster-config-v1";
const CONFIG_REQUEST = new Request("/push-booster-config");

self.addEventListener("install", (event) => {
  event.waitUntil(reportEvent("installed"));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(reportEvent("activated"));
});

self.addEventListener("message", (event) => {
  if (!event.data) {
    return;
  }
  if (event.data.type === "push_booster_config") {
    event.waitUntil(saveConfig(event.data.config).then(() => reportEvent("sdk_configured")));
    return;
  }
  if (event.data.type === "push_booster_subscription") {
    event.waitUntil(saveSubscriptionID(event.data.subscription_id));
  }
});

self.addEventListener("push", (event) => {
  event.waitUntil(showResolvedNotification(event));
});

self.addEventListener("pushsubscriptionchange", (event) => {
  event.waitUntil(reportEvent("subscription_changed"));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const data = event.notification.data || {};
  const url = data.click_url || data.url || "/";
  event.waitUntil(reportEvent("notification_click", data).then(() => self.clients.openWindow(url)));
});

self.addEventListener("notificationclose", (event) => {
  event.waitUntil(reportEvent("notification_close", event.notification.data || {}));
});

async function showResolvedNotification(event) {
  const config = await loadConfig();
  const subscription = await self.registration.pushManager.getSubscription();
  if (!config || !subscription) {
    return;
  }

  const subscriptionID = await loadSubscriptionID();
  if (!subscriptionID) {
    return;
  }

  const pushData = readPushData(event);
  const triggerID = pushData.trigger_id || subscriptionID;
  await reportEvent("push_received", {
    trigger_id: triggerID,
    subscription_id: subscriptionID
  });

  const response = await fetch(config.push_payload_endpoint, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ trigger_id: triggerID })
  });
  if (!response.ok) {
    await reportEvent("payload_failed", {
      trigger_id: triggerID,
      subscription_id: subscriptionID
    });
    return;
  }

  const data = await response.json();
  const notificationData = {
    url: data.url || "/",
    click_url: data.click_url || data.url || "/",
    trigger_id: data.trigger_id || triggerID,
    delivery_id: data.delivery_id,
    subscription_id: data.subscription_id || subscriptionID,
    source_id: data.source_id || config.source_id,
    campaign_id: data.campaign_id,
    creative_id: data.creative_id
  };
  await reportEvent("payload_resolved", notificationData);
  await self.registration.showNotification(data.title || "Push Booster", {
    body: data.body || "Notification received",
    icon: data.icon || "/vite.svg",
    data: notificationData
  });
  await reportEvent("notification_shown", notificationData);
}

async function saveConfig(config) {
  const cache = await caches.open(CONFIG_CACHE);
  await cache.put(CONFIG_REQUEST, new Response(JSON.stringify(config)));
}

async function loadConfig() {
  const cache = await caches.open(CONFIG_CACHE);
  const response = await cache.match(CONFIG_REQUEST);
  if (!response) {
    return null;
  }
  return response.json();
}

async function saveSubscriptionID(subscriptionID) {
  const cache = await caches.open(CONFIG_CACHE);
  await cache.put(new Request("/push-booster-subscription"), new Response(JSON.stringify({
    subscription_id: subscriptionID
  })));
}

async function loadSubscriptionID() {
  const cache = await caches.open(CONFIG_CACHE);
  const response = await cache.match(new Request("/push-booster-subscription"));
  if (!response) {
    return "";
  }
  const data = await response.json();
  return data.subscription_id || "";
}

function readPushData(event) {
  if (!event.data) {
    return {};
  }
  try {
    return event.data.json();
  } catch {
    try {
      return JSON.parse(event.data.text());
    } catch {
      return {};
    }
  }
}

async function reportEvent(eventType, metadata = {}) {
  try {
    const config = await loadConfig();
    if (!config || !config.events_endpoint) {
      return;
    }
    const subscriptionID = await loadSubscriptionID();
    if (!subscriptionID) {
      return;
    }
    const deliveryID = metadata.delivery_id || "";
    await fetch(config.events_endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        subscription_id: subscriptionID,
        delivery_id: deliveryID,
        campaign_id: metadata.campaign_id || "",
        creative_id: metadata.creative_id || "",
        event_id: eventID(eventType, subscriptionID, deliveryID),
        url: metadata.url || "",
        event_type: eventType
      })
    });
  } catch {
    // Event collection must never block notification display.
  }
}

function eventID(eventType, subscriptionID, deliveryID) {
  return [
    deliveryID || subscriptionID,
    eventType
  ].join(":");
}
