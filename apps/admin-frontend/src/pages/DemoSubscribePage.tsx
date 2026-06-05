import { useMutation } from "@tanstack/react-query";
import { Bell, Loader2 } from "lucide-react";
import { useParams } from "react-router";

import { api } from "../api/client";

function urlBase64ToUint8Array(value: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  const base64 = `${value}${padding}`.replace(/-/g, "+").replace(/_/g, "/");
  const raw = window.atob(base64);
  const output = new Uint8Array(raw.length);
  for (let index = 0; index < raw.length; index += 1) {
    output[index] = raw.charCodeAt(index);
  }
  return output;
}

export function DemoSubscribePage() {
  const params = useParams();
  const sourceId = params.sourceId ?? "";

  const subscribe = useMutation({
    mutationFn: async () => {
      if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
        throw new Error("Web Push is not supported in this browser");
      }
      if (Notification.permission === "denied") {
        throw new Error("Notifications are blocked");
      }
      const config = await api.sdkConfig(sourceId);
      const registration = await navigator.serviceWorker.register(config.service_worker_url);
      const readyRegistration = await navigator.serviceWorker.ready;
      readyRegistration.active?.postMessage({ type: "push_booster_config", config });
      const permission = await Notification.requestPermission();
      if (permission !== "granted") {
        throw new Error("Notification permission was not granted");
      }
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(config.vapid_public_key)
      });
      const payload = subscription.toJSON();
      const keys = payload.keys;
      if (!payload.endpoint || !keys?.p256dh || !keys.auth) {
        throw new Error("Browser returned an incomplete push subscription");
      }
      const result = await api.subscribe({
        source_id: config.source_id,
        endpoint: payload.endpoint,
        subid: new URLSearchParams(window.location.search).get("subid") ?? "",
        channel: new URLSearchParams(window.location.search).get("channel") ?? "demo",
        landing_url: window.location.href,
        referrer: document.referrer,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        keys: {
          p256dh: keys.p256dh,
          auth: keys.auth
        }
      });
      readyRegistration.active?.postMessage({
        type: "push_booster_subscription",
        subscription_id: result.subscription_id
      });
      return result;
    }
  });

  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-100 px-4 py-10">
      <section className="w-full max-w-md rounded-md border border-zinc-200 bg-white p-6 shadow-sm">
        <div className="mb-6">
          <div className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
            Push Booster Demo
          </div>
          <h1 className="mt-2 text-2xl font-semibold text-zinc-950">Subscribe demo</h1>
          <p className="mt-2 break-all text-sm text-zinc-600">Source: {sourceId}</p>
        </div>

        <button
          className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
          disabled={subscribe.isPending || !sourceId}
          type="button"
          onClick={() => {
            subscribe.mutate();
          }}
        >
          {subscribe.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Bell className="h-4 w-4" />
          )}
          Subscribe
        </button>

        {subscribe.isSuccess ? (
          <div className="mt-4 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800">
            Subscription stored.
          </div>
        ) : null}
        {subscribe.isError ? (
          <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
            {subscribe.error.message}
          </div>
        ) : null}
      </section>
    </main>
  );
}
