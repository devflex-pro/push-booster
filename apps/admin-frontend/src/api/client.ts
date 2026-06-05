const API_BASE_URL = String(import.meta.env.VITE_API_BASE_URL ?? "");

export type AuthRole = "admin" | "user";
export type AuthStatus = "active" | "pending_approval";

export type AuthUser = {
  id: string;
  email: string;
  role: AuthRole;
  status: AuthStatus;
  email_verified: boolean;
  approved: boolean;
};

export type OTPChallenge = {
  email: string;
  otp?: string;
};

export type AuthSession = {
  token: string;
  user: AuthUser;
};

export type ListResponse<TItem> = {
  items: TItem[];
  total: number;
};

export type Publisher = {
  id: string;
  name: string;
  status: "active";
  created_at: string;
  updated_at: string;
};

export type Source = {
  id: string;
  publisher_id: string;
  name: string;
  domain: string;
  status: "active";
  vapid_key_id?: string;
  created_at: string;
  updated_at: string;
};

export type VAPIDKeyStatus = "active" | "deprecated" | "revoked";

export type VAPIDKey = {
  id: string;
  name: string;
  public_key: string;
  private_key?: string;
  status: VAPIDKeyStatus;
  created_at: string;
  updated_at: string;
};

export type SourceStats = {
  source_id: string;
  subscribers: number;
  subscribers_today: number;
  events_today: number;
  event_breakdown: Record<string, number>;
  last_event_at?: string;
  health: {
    status: "ok" | "attention";
    issues: string[];
  };
};

export type SourceSnippet = {
  snippet: string;
};

export type CampaignStatus = "draft" | "active" | "paused" | "archived";
export type CampaignAudienceScope = "all" | "selected_sources";

export type CreativeStatus = "active" | "paused" | "archived";

export type TargetingRules = {
  countries?: string[];
  languages?: string[];
  device_types?: string[];
  os_names?: string[];
  browser_names?: string[];
};

export type Campaign = {
  id: string;
  publisher_id?: string;
  source_id?: string;
  source_ids: string[];
  audience_scope: CampaignAudienceScope;
  name: string;
  status: CampaignStatus;
  targeting_rules: TargetingRules;
  daily_cap_per_subscription: number;
  total_cap_per_subscription: number;
  created_at: string;
  updated_at: string;
};

export type CreateCampaignPayload = {
  publisher_id?: string;
  source_id?: string;
  source_ids: string[];
  audience_scope: CampaignAudienceScope;
  name: string;
  targeting_rules: TargetingRules;
  daily_cap_per_subscription: number;
  total_cap_per_subscription: number;
};

export type Creative = {
  id: string;
  campaign_id: string;
  title: string;
  body: string;
  url: string;
  icon?: string;
  status: CreativeStatus;
  source_type: "manual" | "provider_api";
  provider_config_id?: string;
  provider_name?: string;
  provider_external_id?: string;
  last_synced_at?: string;
  sync_status?: "synced" | "invalid" | "stale";
  daily_cap_per_subscription: number;
  total_cap_per_subscription: number;
  created_at: string;
  updated_at: string;
};

export type CreativeProviderConfig = {
  id: string;
  campaign_id: string;
  name: string;
  provider_name: string;
  fetch_url: string;
  request_headers: Record<string, string>;
  status: "active" | "paused" | "archived";
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
};

export type CreativeProviderConfigStatus = "active" | "paused" | "archived";

export type CreateCreativeProviderConfigPayload = {
  campaign_id: string;
  name: string;
  provider_name: string;
  fetch_url: string;
  request_headers: Record<string, string>;
};

export type CreativeSyncLog = {
  id: string;
  provider_config_id: string;
  campaign_id: string;
  status: "running" | "completed" | "failed";
  fetched_total: number;
  upserted_total: number;
  error_message?: string;
  started_at: string;
  completed_at?: string;
};

export type CreativeExposureCount = {
  creative_id: string;
  count: number;
};

export type CampaignReport = {
  campaign_id: string;
  decisions_total: number;
  selected: number;
  suppressed: number;
  not_found: number;
  errors: number;
  tracked_events: number;
  shown: number;
  clicks: number;
  closed: number;
  events_by_type: Record<string, number>;
  decisions_by_result: Record<string, number>;
  decisions_by_reason: Record<string, number>;
  creative_exposures: CreativeExposureCount[];
  health: {
    status: "ok" | "attention";
    issues: string[];
  };
};

export type CampaignLaunchStatus = "building" | "completed" | "failed";
export type CampaignScheduleStatus = "active" | "paused" | "archived";

export type CampaignLaunch = {
  id: string;
  campaign_id: string;
  status: CampaignLaunchStatus;
  audience_total: number;
  processed_total: number;
  error_message?: string;
  enqueue_status: "pending" | "enqueuing" | "completed" | "failed";
  enqueued_total: number;
  enqueue_error?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  enqueue_started_at?: string;
  enqueue_completed_at?: string;
};

export type CampaignScheduleSlot = {
  id: string;
  schedule_id: string;
  local_time: string;
  days_of_week: number[];
  position: number;
  created_at: string;
};

export type CampaignSchedule = {
  id: string;
  campaign_id: string;
  status: CampaignScheduleStatus;
  timezone_mode: "subscriber_local";
  fallback_timezone: string;
  grace_minutes: number;
  slots: CampaignScheduleSlot[];
  created_at: string;
  updated_at: string;
};

export type CampaignScheduleRun = {
  id: string;
  schedule_id: string;
  slot_id: string;
  campaign_id: string;
  launch_id?: string;
  local_date: string;
  local_time: string;
  timezone: string;
  scheduled_utc_at: string;
  status: "pending" | "running" | "completed" | "failed";
  audience_total: number;
  enqueued_total: number;
  error_message?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
};

export type CreateCampaignSchedulePayload = {
  status?: CampaignScheduleStatus;
  fallback_timezone: string;
  grace_minutes: number;
  slots: Array<{
    local_time: string;
    days_of_week: number[];
    position: number;
  }>;
};

export type AudienceEstimate = {
  campaign_id: string;
  source_id?: string;
  source_ids: string[];
  audience_scope: CampaignAudienceScope;
  total: number;
};

export type CreateCreativePayload = {
  campaign_id: string;
  title: string;
  body: string;
  url: string;
  icon: string;
  daily_cap_per_subscription: number;
  total_cap_per_subscription: number;
};

export type SDKConfig = {
  source_id: string;
  vapid_public_key: string;
  subscribe_endpoint: string;
  push_payload_endpoint: string;
  events_endpoint: string;
  service_worker_url: string;
};

export type SubscribePayload = {
  source_id: string;
  endpoint: string;
  subid?: string;
  channel?: string;
  landing_url?: string;
  referrer?: string;
  timezone?: string;
  keys: {
    p256dh: string;
    auth: string;
  };
};

export type SubscribeResponse = {
  status: string;
  subscription_id: string;
};

export type PostbackConfigStatus = "active" | "paused" | "archived";

export type PostbackConfig = {
  id: string;
  name: string;
  source_id?: string;
  token?: string;
  status: PostbackConfigStatus;
  click_id_param: string;
  delivery_id_param: string;
  subscription_id_param: string;
  external_id_param: string;
  payout_param: string;
  currency_param: string;
  status_param: string;
  default_currency: string;
  created_at: string;
  updated_at: string;
};

export type CreatePostbackConfigPayload = {
  name: string;
  source_id?: string;
  token?: string;
  click_id_param?: string;
  delivery_id_param?: string;
  subscription_id_param?: string;
  external_id_param?: string;
  payout_param?: string;
  currency_param?: string;
  status_param?: string;
  default_currency?: string;
};

export type PostbackEvent = {
  postback_config_id: string;
  dedupe_key: string;
  external_id: string;
  click_id: string;
  delivery_id: string;
  subscription_id: string;
  source_id: string;
  campaign_id: string;
  creative_id: string;
  payout: number;
  currency: string;
  status: string;
  attribution_status: string;
  raw_payload: string;
  received_at: string;
};

export type ReportGroupBy = "date" | "publisher" | "source" | "campaign" | "creative";

export type PerformanceRow = {
  key: string;
  name: string;
  group_by: ReportGroupBy;
  spend: number;
  revenue: number;
  profit: number;
  roi: number;
  conversions: number;
  sent: number;
  shown: number;
  clicks: number;
  closed: number;
  ctr: number;
  cr: number;
};

export type ReportQuery = {
  date_from?: string;
  date_to?: string;
  publisher_id?: string;
  campaign_id?: string;
  limit?: number;
  sort_by?: string;
};

export type DashboardReport = {
  spend: number;
  revenue: number;
  profit: number;
  roi: number;
  conversions: number;
  sent: number;
  shown: number;
  clicks: number;
  closed: number;
  ctr: number;
  cr: number;
  rows: PerformanceRow[];
};

export type CostEntry = {
  id: string;
  date: string;
  publisher_id?: string;
  publisher_name?: string;
  source_id?: string;
  source_name?: string;
  campaign_id?: string;
  campaign_name?: string;
  creative_id?: string;
  creative_name?: string;
  amount: number;
  currency: string;
  note: string;
  created_at: string;
};

export type CreateCostEntryPayload = {
  date: string;
  publisher_id?: string;
  source_id?: string;
  campaign_id?: string;
  creative_id?: string;
  amount: number;
  currency: string;
  note: string;
};

export type CostImportResult = {
  inserted: number;
  entries: CostEntry[];
};

type ApiErrorBody = {
  error?: string;
  status?: AuthStatus;
  user?: AuthUser;
};

export class ApiRequestError extends Error {
  readonly statusCode: number;
  readonly response?: ApiErrorBody;

  constructor(message: string, statusCode: number, response?: ApiErrorBody) {
    super(message);
    this.name = "ApiRequestError";
    this.statusCode = statusCode;
    this.response = response;
  }
}

async function request<TResponse>(
  path: string,
  token: string | null,
  options: RequestInit = {}
): Promise<TResponse> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  });

  if (!response.ok) {
    throw await buildRequestError(response);
  }

  if (response.status === 204) {
    return undefined as TResponse;
  }

  return response.json() as Promise<TResponse>;
}

async function requestRaw<TResponse>(
  path: string,
  token: string | null,
  options: RequestInit
): Promise<TResponse> {
  const headers = new Headers(options.headers);
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  });

  if (!response.ok) {
    throw await buildRequestError(response);
  }

  return response.json() as Promise<TResponse>;
}

async function buildRequestError(response: Response): Promise<ApiRequestError> {
  let message = `Request failed with status ${String(response.status)}`;
  let body: ApiErrorBody | undefined;
  try {
    body = (await response.json()) as ApiErrorBody;
    if (body.error) {
      message = body.error;
    }
  } catch (error: unknown) {
    if (error instanceof Error) {
      message = `${message}: ${error.message}`;
    }
  }
  return new ApiRequestError(message, response.status, body);
}

function jsonBody(payload: object): RequestInit {
  return {
    body: JSON.stringify(payload)
  };
}

function reportQuery(params: ReportQuery = {}): string {
  const query = new URLSearchParams();
  if (params.date_from) {
    query.set("date_from", params.date_from);
  }
  if (params.date_to) {
    query.set("date_to", params.date_to);
  }
  if (params.publisher_id) {
    query.set("publisher_id", params.publisher_id);
  }
  if (params.campaign_id) {
    query.set("campaign_id", params.campaign_id);
  }
  if (params.limit) {
    query.set("limit", String(params.limit));
  }
  if (params.sort_by) {
    query.set("sort_by", params.sort_by);
  }
  return query.toString();
}

export const api = {
  requestOTP: (email: string): Promise<OTPChallenge> =>
    request<OTPChallenge>("/api/auth/request-otp", null, {
      method: "POST",
      ...jsonBody({ email })
    }),

  verifyOTP: (email: string, otp: string): Promise<AuthSession> =>
    request<AuthSession>("/api/auth/verify-otp", null, {
      method: "POST",
      ...jsonBody({ email, otp })
    }),

  me: (token: string): Promise<AuthUser> =>
    request<AuthUser>("/api/me", token),

  users: (token: string): Promise<ListResponse<AuthUser>> =>
    request<ListResponse<AuthUser>>("/api/users", token),

  approveUser: (token: string, id: string): Promise<AuthUser> =>
    request<AuthUser>(`/api/users/${id}/approve`, token, {
      method: "POST"
    }),

  publishers: (token: string): Promise<ListResponse<Publisher>> =>
    request<ListResponse<Publisher>>("/api/publishers", token),

  createPublisher: (token: string, name: string): Promise<Publisher> =>
    request<Publisher>("/api/publishers", token, {
      method: "POST",
      ...jsonBody({ name })
    }),

  sources: (token: string, publisherId?: string): Promise<ListResponse<Source>> => {
    const query = publisherId ? `?publisher_id=${encodeURIComponent(publisherId)}` : "";
    return request<ListResponse<Source>>(`/api/sources${query}`, token);
  },

  createSource: (
    token: string,
    payload: { publisher_id: string; name: string; domain: string }
  ): Promise<Source> =>
    request<Source>("/api/sources", token, {
      method: "POST",
      ...jsonBody(payload)
    }),

  sourceSnippet: (token: string, id: string): Promise<SourceSnippet> =>
    request<SourceSnippet>(`/api/sources/${id}/snippet`, token),

  sourceStats: (token: string, id: string): Promise<SourceStats> =>
    request<SourceStats>(`/api/sources/${id}/stats`, token),

  vapidKeys: (token: string): Promise<ListResponse<VAPIDKey>> =>
    request<ListResponse<VAPIDKey>>("/api/vapid-keys", token),

  createVAPIDKey: (token: string, name: string): Promise<VAPIDKey> =>
    request<VAPIDKey>("/api/vapid-keys", token, {
      method: "POST",
      ...jsonBody({ name })
    }),

  updateVAPIDKeyStatus: (
    token: string,
    id: string,
    status: VAPIDKeyStatus
  ): Promise<VAPIDKey> =>
    request<VAPIDKey>(`/api/vapid-keys/${id}/status`, token, {
      method: "POST",
      ...jsonBody({ status })
    }),

  attachVAPIDKeyToSource: (
    token: string,
    sourceId: string,
    vapidKeyId: string
  ): Promise<Source> =>
    request<Source>(`/api/sources/${sourceId}/vapid-key`, token, {
      method: "POST",
      ...jsonBody({ vapid_key_id: vapidKeyId })
    }),

  vapidPublicKey: (): Promise<{ public_key: string }> =>
    request<{ public_key: string }>("/api/web-push/vapid-public-key", null),

  sdkConfig: (sourceId: string): Promise<SDKConfig> =>
    request<SDKConfig>(`/api/sdk/config?source_id=${encodeURIComponent(sourceId)}`, null),

  campaigns: (token: string, sourceId?: string): Promise<ListResponse<Campaign>> => {
    const query = sourceId ? `?source_id=${encodeURIComponent(sourceId)}` : "";
    return request<ListResponse<Campaign>>(`/api/campaigns${query}`, token);
  },

  createCampaign: (
    token: string,
    payload: CreateCampaignPayload
  ): Promise<Campaign> =>
    request<Campaign>("/api/campaigns", token, {
      method: "POST",
      ...jsonBody(payload)
    }),

  updateCampaignStatus: (
    token: string,
    id: string,
    status: CampaignStatus
  ): Promise<Campaign> =>
    request<Campaign>(`/api/campaigns/${id}/status`, token, {
      method: "POST",
      ...jsonBody({ status })
    }),

  creatives: (token: string, campaignId?: string): Promise<ListResponse<Creative>> => {
    const query = campaignId ? `?campaign_id=${encodeURIComponent(campaignId)}` : "";
    return request<ListResponse<Creative>>(`/api/creatives${query}`, token);
  },

  createCreative: (
    token: string,
    payload: CreateCreativePayload
  ): Promise<Creative> =>
    request<Creative>("/api/creatives", token, {
      method: "POST",
      ...jsonBody(payload)
    }),

  updateCreativeStatus: (
    token: string,
    id: string,
    status: CreativeStatus
  ): Promise<Creative> =>
    request<Creative>(`/api/creatives/${id}/status`, token, {
      method: "POST",
      ...jsonBody({ status })
    }),

  creativeProviderConfigs: (
    token: string,
    campaignId?: string
  ): Promise<ListResponse<CreativeProviderConfig>> => {
    const query = campaignId ? `?campaign_id=${encodeURIComponent(campaignId)}` : "";
    return request<ListResponse<CreativeProviderConfig>>(
      `/api/creative-provider-configs${query}`,
      token
    );
  },

  createCreativeProviderConfig: (
    token: string,
    payload: CreateCreativeProviderConfigPayload
  ): Promise<CreativeProviderConfig> =>
    request<CreativeProviderConfig>("/api/creative-provider-configs", token, {
      method: "POST",
      ...jsonBody(payload)
    }),

  syncCreativeProviderConfig: (
    token: string,
    id: string
  ): Promise<CreativeSyncLog> =>
    request<CreativeSyncLog>(`/api/creative-provider-configs/${id}/sync`, token, {
      method: "POST"
    }),

  updateCreativeProviderConfigStatus: (
    token: string,
    id: string,
    status: CreativeProviderConfigStatus
  ): Promise<CreativeProviderConfig> =>
    request<CreativeProviderConfig>(`/api/creative-provider-configs/${id}/status`, token, {
      method: "POST",
      ...jsonBody({ status })
    }),

  creativeSyncLogs: (
    token: string,
    params: { campaign_id?: string; provider_config_id?: string } = {}
  ): Promise<ListResponse<CreativeSyncLog>> => {
    const query = new URLSearchParams();
    if (params.campaign_id) {
      query.set("campaign_id", params.campaign_id);
    }
    if (params.provider_config_id) {
      query.set("provider_config_id", params.provider_config_id);
    }
    return request<ListResponse<CreativeSyncLog>>(
      `/api/creative-sync-logs${query.toString() ? `?${query.toString()}` : ""}`,
      token
    );
  },

  campaignReport: (token: string, campaignId: string): Promise<CampaignReport> =>
    request<CampaignReport>(
      `/api/reports/campaigns?campaign_id=${encodeURIComponent(campaignId)}`,
      token
    ),

  dashboardReport: (token: string, params: ReportQuery = {}): Promise<DashboardReport> => {
    const query = reportQuery(params);
    return request<DashboardReport>(`/api/reports/dashboard${query ? `?${query}` : ""}`, token);
  },

  performanceReport: (
    token: string,
    groupBy: ReportGroupBy,
    params: ReportQuery = {}
  ): Promise<ListResponse<PerformanceRow>> => {
    const query = new URLSearchParams(reportQuery(params));
    query.set("group_by", groupBy);
    return request<ListResponse<PerformanceRow>>(
      `/api/reports/performance?${query.toString()}`,
      token
    );
  },

  costs: (
    token: string,
    limit = 50,
    offset = 0
  ): Promise<ListResponse<CostEntry>> =>
    request<ListResponse<CostEntry>>(
      `/api/costs?limit=${String(limit)}&offset=${String(offset)}`,
      token
    ),

  createCost: (
    token: string,
    payload: CreateCostEntryPayload
  ): Promise<CostEntry> =>
    request<CostEntry>("/api/costs", token, {
      method: "POST",
      ...jsonBody(payload)
    }),

  importCosts: (token: string, file: File): Promise<CostImportResult> => {
    const form = new FormData();
    form.set("file", file);
    return requestRaw<CostImportResult>("/api/costs/import", token, {
      method: "POST",
      body: form
    });
  },

  campaignLaunches: (
    token: string,
    campaignId: string
  ): Promise<ListResponse<CampaignLaunch>> =>
    request<ListResponse<CampaignLaunch>>(
      `/api/campaigns/${campaignId}/launches`,
      token
    ),

  campaignSchedules: (
    token: string,
    campaignId: string
  ): Promise<ListResponse<CampaignSchedule>> =>
    request<ListResponse<CampaignSchedule>>(
      `/api/campaigns/${campaignId}/schedules`,
      token
    ),

  createCampaignSchedule: (
    token: string,
    campaignId: string,
    payload: CreateCampaignSchedulePayload
  ): Promise<CampaignSchedule> =>
    request<CampaignSchedule>(`/api/campaigns/${campaignId}/schedules`, token, {
      method: "POST",
      ...jsonBody(payload)
    }),

  updateCampaignScheduleStatus: (
    token: string,
    campaignId: string,
    scheduleId: string,
    status: CampaignScheduleStatus
  ): Promise<CampaignSchedule> =>
    request<CampaignSchedule>(
      `/api/campaigns/${campaignId}/schedules/${scheduleId}/status`,
      token,
      {
        method: "POST",
        ...jsonBody({ status })
      }
    ),

  campaignScheduleRuns: (
    token: string,
    campaignId: string
  ): Promise<ListResponse<CampaignScheduleRun>> =>
    request<ListResponse<CampaignScheduleRun>>(
      `/api/campaigns/${campaignId}/schedule-runs`,
      token
    ),

  estimateCampaignAudience: (
    token: string,
    campaignId: string
  ): Promise<AudienceEstimate> =>
    request<AudienceEstimate>(`/api/campaigns/${campaignId}/launches/estimate`, token, {
      method: "POST"
    }),

  createCampaignLaunch: (
    token: string,
    campaignId: string
  ): Promise<CampaignLaunch> =>
    request<CampaignLaunch>(`/api/campaigns/${campaignId}/launches`, token, {
      method: "POST"
    }),

  enqueueCampaignLaunch: (
    token: string,
    campaignId: string,
    launchId: string
  ): Promise<CampaignLaunch> =>
    request<CampaignLaunch>(
      `/api/campaigns/${campaignId}/launches/${launchId}/enqueue`,
      token,
      {
        method: "POST"
      }
    ),

  subscribe: (payload: SubscribePayload): Promise<SubscribeResponse> =>
    request<SubscribeResponse>("/api/subscribe", null, {
      method: "POST",
      ...jsonBody(payload)
    }),

  postbackConfigs: (token: string): Promise<ListResponse<PostbackConfig>> =>
    request<ListResponse<PostbackConfig>>("/api/postback-configs", token),

  createPostbackConfig: (
    token: string,
    payload: CreatePostbackConfigPayload
  ): Promise<PostbackConfig> =>
    request<PostbackConfig>("/api/postback-configs", token, {
      method: "POST",
      ...jsonBody(payload)
    }),

  updatePostbackConfigStatus: (
    token: string,
    id: string,
    status: PostbackConfigStatus
  ): Promise<PostbackConfig> =>
    request<PostbackConfig>(`/api/postback-configs/${id}/status`, token, {
      method: "POST",
      ...jsonBody({ status })
    }),

  postbacks: (
    token: string,
    limit = 50,
    postbackConfigId?: string
  ): Promise<ListResponse<PostbackEvent>> => {
    const params = new URLSearchParams({ limit: String(limit) });
    if (postbackConfigId) {
      params.set("postback_config_id", postbackConfigId);
    }
    return request<ListResponse<PostbackEvent>>(`/api/postbacks?${params.toString()}`, token);
  }
};
