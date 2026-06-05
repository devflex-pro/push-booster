import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { ArrowLeft, Loader2, Plus, RefreshCw, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { type UseFormRegisterReturn, useForm } from "react-hook-form";
import { Link, useParams } from "react-router";
import { z } from "zod";

import {
  Campaign,
  CampaignAudienceScope,
  CampaignLaunch,
  CampaignReport,
  CampaignSchedule,
  CampaignScheduleRun,
  CampaignScheduleStatus,
  CampaignStatus,
  Creative,
  CreativeProviderConfig,
  CreativeProviderConfigStatus,
  CreativeSyncLog,
  CreativeStatus,
  TargetingRules,
  Publisher,
  Source,
  api
} from "../api/client";
import { useAuthStore } from "../store/auth";

const campaignSchema = z.object({
  audience_scope: z.enum(["all", "selected_sources"]),
  source_ids: z.array(z.string()),
  name: z.string().trim().min(2),
  countries: z.string(),
  languages: z.string(),
  device_types: z.string(),
  os_names: z.string(),
  browser_names: z.string(),
  daily_cap_per_subscription: z.number().int().min(0),
  total_cap_per_subscription: z.number().int().min(0)
}).refine((value) => value.audience_scope === "all" || value.source_ids.length > 0, {
  message: "Select at least one source",
  path: ["source_ids"]
});

const creativeSchema = z.object({
  campaign_id: z.string().trim().min(1),
  title: z.string().trim().min(2),
  body: z.string().trim().min(2),
  url: z.url(),
  icon: z.string().trim(),
  daily_cap_per_subscription: z.number().int().min(0),
  total_cap_per_subscription: z.number().int().min(0)
});

const providerConfigSchema = z.object({
  campaign_id: z.string().trim().min(1),
  name: z.string().trim().min(2),
  provider_name: z.string().trim().min(2),
  fetch_url: z.url(),
  request_headers: z.string()
});

const scheduleSchema = z.object({
  local_times: z.string().trim().min(5),
  days_of_week: z.string().trim(),
  fallback_timezone: z.string().trim().min(1),
  grace_minutes: z.number().int().min(1).max(120)
});

type CampaignForm = z.infer<typeof campaignSchema>;
type CreativeForm = z.infer<typeof creativeSchema>;
type ProviderConfigForm = z.infer<typeof providerConfigSchema>;
type ScheduleForm = z.infer<typeof scheduleSchema>;

const campaignStatuses: CampaignStatus[] = ["draft", "active", "paused", "archived"];
const campaignAudienceScopes: CampaignAudienceScope[] = ["all", "selected_sources"];
const creativeStatuses: CreativeStatus[] = ["active", "paused", "archived"];
const providerConfigStatuses: CreativeProviderConfigStatus[] = ["active", "paused", "archived"];
const scheduleStatuses: CampaignScheduleStatus[] = ["active", "paused", "archived"];
const creativeSourceFilters = ["all", "manual", "provider_api"] as const;
const creativeSyncFilters = ["all", "synced", "stale", "invalid"] as const;
const creativeStatusFilters = ["all", "active", "paused", "archived"] as const;

type CreativeSourceFilter = (typeof creativeSourceFilters)[number];
type CreativeSyncFilter = (typeof creativeSyncFilters)[number];
type CreativeStatusFilter = (typeof creativeStatusFilters)[number];

export function CampaignsPage() {
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();
  const [publisherFilter, setPublisherFilter] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [createCampaignOpen, setCreateCampaignOpen] = useState(false);

  const campaignForm = useForm<CampaignForm>({
    resolver: zodResolver(campaignSchema),
    defaultValues: {
      audience_scope: "all",
      source_ids: [],
      name: "",
      countries: "",
      languages: "",
      device_types: "",
      os_names: "",
      browser_names: "",
      daily_cap_per_subscription: 0,
      total_cap_per_subscription: 0
    }
  });

  const publishers = useQuery({
    queryKey: ["publishers", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.publishers(token);
    },
    enabled: Boolean(token)
  });

  const sources = useQuery({
    queryKey: ["sources", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.sources(token);
    },
    enabled: Boolean(token)
  });

  const campaigns = useQuery({
    queryKey: ["campaigns", token, sourceFilter],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.campaigns(token, sourceFilter || undefined);
    },
    enabled: Boolean(token)
  });

  const createCampaign = useMutation({
    mutationFn: (values: CampaignForm) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.createCampaign(token, {
        audience_scope: values.audience_scope,
        source_ids: values.audience_scope === "all" ? [] : values.source_ids,
        name: values.name,
        targeting_rules: targetingRulesFromForm(values),
        daily_cap_per_subscription: values.daily_cap_per_subscription,
        total_cap_per_subscription: values.total_cap_per_subscription
      });
    },
    onSuccess: async () => {
      campaignForm.reset();
      setCreateCampaignOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["campaigns", token] });
    }
  });

  const updateCampaignStatus = useMutation({
    mutationFn: (input: { id: string; status: CampaignStatus }) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.updateCampaignStatus(token, input.id, input.status);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["campaigns", token] });
    }
  });

  const filteredSourceOptions = useMemo(
    () =>
      (sources.data?.items ?? []).filter(
        (source) => !publisherFilter || source.publisher_id === publisherFilter
      ),
    [publisherFilter, sources.data?.items]
  );
  const filteredCampaignItems = useMemo(() => {
    const items = campaigns.data?.items ?? [];
    if (sourceFilter) {
      return items;
    }
    if (!publisherFilter) {
      return items;
    }
    const sourceIds = new Set(filteredSourceOptions.map((source) => source.id));
    return items.filter(
      (campaign) =>
        campaign.audience_scope === "all" ||
        campaign.source_ids.some((sourceID) => sourceIds.has(sourceID))
    );
  }, [campaigns.data?.items, filteredSourceOptions, publisherFilter, sourceFilter]);
  const campaignColumns = useMemo<ColumnDef<Campaign>[]>(
    () => [
      { header: "Name", accessorKey: "name" },
      {
        header: "Audience",
        cell: ({ row }) => audienceSummary(row.original, sources.data?.items ?? [])
      },
      {
        header: "Caps",
        cell: ({ row }) =>
          `${String(row.original.daily_cap_per_subscription)} daily / ${String(row.original.total_cap_per_subscription)} total`
      },
      {
        header: "Targeting",
        cell: ({ row }) => targetingSummary(row.original.targeting_rules)
      },
      {
        header: "Status",
        cell: ({ row }) => (
          <StatusSelect<CampaignStatus>
            disabled={updateCampaignStatus.isPending}
            options={campaignStatuses}
            value={row.original.status}
            onChange={(status) => {
              updateCampaignStatus.mutate({ id: row.original.id, status });
            }}
          />
        )
      },
      {
        header: "Creatives",
        cell: ({ row }) => (
          <Link
            className="inline-flex h-8 items-center rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
            to={`/campaigns/${row.original.id}`}
          >
            Open
          </Link>
        )
      }
    ],
    [sources.data?.items, updateCampaignStatus]
  );

  const campaignTable = useReactTable({
    data: filteredCampaignItems,
    columns: campaignColumns,
    getCoreRowModel: getCoreRowModel()
  });

  function submitCampaign(values: CampaignForm) {
    createCampaign.mutate(values);
  }

  return (
    <section className="space-y-4">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-zinc-950">Campaigns</h1>
          <p className="mt-1 text-sm text-zinc-600">Campaign, creative, caps and targeting setup.</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            className="inline-flex h-9 items-center gap-2 rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800"
            type="button"
            onClick={() => {
              setCreateCampaignOpen(true);
            }}
          >
            <Plus className="h-4 w-4" />
            Create campaign
          </button>
          <button
            className="inline-flex h-9 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
            type="button"
            onClick={() => {
              void queryClient.invalidateQueries({ queryKey: ["campaigns", token] });
            }}
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </button>
        </div>
      </div>

      {campaigns.isError ? <ErrorMessage message={campaigns.error.message} /> : null}
      {updateCampaignStatus.isError ? <ErrorMessage message={updateCampaignStatus.error.message} /> : null}

      <div className="flex flex-wrap items-center gap-3 rounded-md border border-zinc-200 bg-white p-4">
        <label className="text-sm font-medium text-zinc-800">
          Publisher
          <select
            className="ml-3 h-9 rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
            value={publisherFilter}
            onChange={(event) => {
              setPublisherFilter(event.target.value);
              setSourceFilter("");
            }}
          >
            <option value="">All publishers</option>
            {(publishers.data?.items ?? []).map((publisher) => (
              <option key={publisher.id} value={publisher.id}>
                {publisher.name}
              </option>
            ))}
          </select>
        </label>
        <label className="text-sm font-medium text-zinc-800">
          Source
          <select
            className="ml-3 h-9 rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
            disabled={!publisherFilter}
            value={sourceFilter}
            onChange={(event) => {
              setSourceFilter(event.target.value);
            }}
          >
            <option value="">{publisherFilter ? "All sources" : "Select publisher first"}</option>
            {filteredSourceOptions.map((source) => (
              <option key={source.id} value={source.id}>
                {source.name}
              </option>
            ))}
          </select>
        </label>
        <div className="text-sm text-zinc-500">{filteredCampaignItems.length} campaigns</div>
      </div>

      <DataTable<Campaign>
        emptyText="No campaigns yet"
        isLoading={campaigns.isLoading}
        table={campaignTable}
      />

      {createCampaignOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="max-h-[90vh] w-full max-w-3xl overflow-auto rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Create campaign</h2>
              <button
                className="inline-flex h-8 w-8 items-center justify-center rounded-md text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-950"
                type="button"
                onClick={() => {
                  campaignForm.reset();
                  setCreateCampaignOpen(false);
                }}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <form
              className="grid gap-3 p-4 sm:grid-cols-2"
              onSubmit={(event) => {
                void campaignForm.handleSubmit(submitCampaign)(event);
              }}
            >
              <SelectField
                label="Audience"
                options={campaignAudienceScopes.map((scope) => ({
                  label: scope === "all" ? "All sources" : "Selected sources",
                  value: scope
                }))}
                placeholder="Select audience"
                register={campaignForm.register("audience_scope")}
              />
              {campaignForm.watch("audience_scope") === "selected_sources" ? (
                <label className="space-y-1 text-sm font-medium text-zinc-800 sm:col-span-2">
                  <span>Sources</span>
                  <select
                    className="min-h-40 w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm outline-none transition focus:border-zinc-950"
                    multiple
                    {...campaignForm.register("source_ids")}
                  >
                    {(sources.data?.items ?? []).map((source) => (
                      <option key={source.id} value={source.id}>
                        {publisherName(publishers.data?.items ?? [], source.publisher_id)} / {source.name} ({source.domain})
                      </option>
                    ))}
                  </select>
                </label>
              ) : null}
              <TextField label="Name" placeholder="Campaign name" register={campaignForm.register("name")} />
              <NumberField
                label="Daily cap"
                register={campaignForm.register("daily_cap_per_subscription", { valueAsNumber: true })}
              />
              <NumberField
                label="Total cap"
                register={campaignForm.register("total_cap_per_subscription", { valueAsNumber: true })}
              />
              <TextField label="Countries" placeholder="us, ca" register={campaignForm.register("countries")} />
              <TextField label="Languages" placeholder="en, es" register={campaignForm.register("languages")} />
              <TextField
                label="Device types"
                placeholder="mobile, desktop"
                register={campaignForm.register("device_types")}
              />
              <TextField label="OS names" placeholder="ios, android" register={campaignForm.register("os_names")} />
              <TextField
                label="Browsers"
                placeholder="safari, chrome"
                register={campaignForm.register("browser_names")}
              />
              <div className="space-y-3 sm:col-span-2">
                <ErrorList errors={Object.values(campaignForm.formState.errors).map((error) => error.message)} />
                {createCampaign.isError ? <ErrorMessage message={createCampaign.error.message} /> : null}
                <div className="flex justify-end gap-2">
                  <button
                    className="inline-flex h-10 items-center justify-center rounded-md border border-zinc-300 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50"
                    type="button"
                    onClick={() => {
                      campaignForm.reset();
                      setCreateCampaignOpen(false);
                    }}
                  >
                    Cancel
                  </button>
                  <button
                    className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
                    disabled={createCampaign.isPending}
                    type="submit"
                  >
                    {createCampaign.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Plus className="h-4 w-4" />
                    )}
                    Create campaign
                  </button>
                </div>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </section>
  );
}

export function CampaignDetailsPage() {
  const { campaignId } = useParams<{ campaignId: string }>();
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();
  const [creativeSourceFilter, setCreativeSourceFilter] = useState<CreativeSourceFilter>("all");
  const [creativeSyncFilter, setCreativeSyncFilter] = useState<CreativeSyncFilter>("all");
  const [creativeStatusFilter, setCreativeStatusFilter] = useState<CreativeStatusFilter>("all");
  const [createCreativeOpen, setCreateCreativeOpen] = useState(false);
  const [createProviderOpen, setCreateProviderOpen] = useState(false);
  const [createScheduleOpen, setCreateScheduleOpen] = useState(false);

  const creativeForm = useForm<CreativeForm>({
    resolver: zodResolver(creativeSchema),
    defaultValues: {
      campaign_id: campaignId ?? "",
      title: "",
      body: "",
      url: "",
      icon: "",
      daily_cap_per_subscription: 0,
      total_cap_per_subscription: 0
    }
  });
  const providerConfigForm = useForm<ProviderConfigForm>({
    resolver: zodResolver(providerConfigSchema),
    defaultValues: {
      campaign_id: campaignId ?? "",
      name: "",
      provider_name: "generic",
      fetch_url: "",
      request_headers: "{}"
    }
  });
  const scheduleForm = useForm<ScheduleForm>({
    resolver: zodResolver(scheduleSchema),
    defaultValues: {
      local_times: "10:00, 10:45",
      days_of_week: "1,2,3,4,5,6,7",
      fallback_timezone: "UTC",
      grace_minutes: 10
    }
  });

  useEffect(() => {
    if (campaignId) {
      creativeForm.setValue("campaign_id", campaignId);
      providerConfigForm.setValue("campaign_id", campaignId);
    }
  }, [campaignId, creativeForm, providerConfigForm]);

  const campaigns = useQuery({
    queryKey: ["campaigns", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.campaigns(token);
    },
    enabled: Boolean(token)
  });

  const creatives = useQuery({
    queryKey: ["creatives", token, campaignId],
    queryFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing creative query input");
      }
      return api.creatives(token, campaignId);
    },
    enabled: Boolean(token && campaignId)
  });

  const campaignReport = useQuery({
    queryKey: ["campaign-report", token, campaignId],
    queryFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign report input");
      }
      return api.campaignReport(token, campaignId);
    },
    enabled: Boolean(token && campaignId)
  });

  const campaignLaunches = useQuery({
    queryKey: ["campaign-launches", token, campaignId],
    queryFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign launch input");
      }
      return api.campaignLaunches(token, campaignId);
    },
    enabled: Boolean(token && campaignId)
  });

  const campaignSchedules = useQuery({
    queryKey: ["campaign-schedules", token, campaignId],
    queryFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign schedule input");
      }
      return api.campaignSchedules(token, campaignId);
    },
    enabled: Boolean(token && campaignId)
  });

  const campaignScheduleRuns = useQuery({
    queryKey: ["campaign-schedule-runs", token, campaignId],
    queryFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign schedule run input");
      }
      return api.campaignScheduleRuns(token, campaignId);
    },
    enabled: Boolean(token && campaignId)
  });

  const providerConfigs = useQuery({
    queryKey: ["creative-provider-configs", token, campaignId],
    queryFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing provider config query input");
      }
      return api.creativeProviderConfigs(token, campaignId);
    },
    enabled: Boolean(token && campaignId)
  });

  const providerSyncLogs = useQuery({
    queryKey: ["creative-sync-logs", token, campaignId],
    queryFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing provider sync log query input");
      }
      return api.creativeSyncLogs(token, { campaign_id: campaignId });
    },
    enabled: Boolean(token && campaignId)
  });

  const createCreative = useMutation({
    mutationFn: (values: CreativeForm) => {
      if (!token || !campaignId) {
        throw new Error("Missing creative input");
      }
      return api.createCreative(token, { ...values, campaign_id: campaignId });
    },
    onSuccess: async () => {
      creativeForm.reset({
        campaign_id: campaignId ?? "",
        title: "",
        body: "",
        url: "",
        icon: "",
        daily_cap_per_subscription: 0,
        total_cap_per_subscription: 0
      });
      setCreateCreativeOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["creatives", token, campaignId] });
    }
  });

  const updateCreativeStatus = useMutation({
    mutationFn: (input: { id: string; status: CreativeStatus }) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.updateCreativeStatus(token, input.id, input.status);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["creatives", token, campaignId] });
    }
  });

  const createProviderConfig = useMutation({
    mutationFn: (values: ProviderConfigForm) => {
      if (!token || !campaignId) {
        throw new Error("Missing provider config input");
      }
      return api.createCreativeProviderConfig(token, {
        campaign_id: campaignId,
        name: values.name,
        provider_name: values.provider_name,
        fetch_url: values.fetch_url,
        request_headers: parseHeaders(values.request_headers)
      });
    },
    onSuccess: async (config) => {
      providerConfigForm.reset({
        campaign_id: config.campaign_id,
        name: "",
        provider_name: config.provider_name,
        fetch_url: "",
        request_headers: "{}"
      });
      setCreateProviderOpen(false);
      await queryClient.invalidateQueries({
        queryKey: ["creative-provider-configs", token, config.campaign_id]
      });
    }
  });

  const syncProviderConfig = useMutation({
    mutationFn: (id: string) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.syncCreativeProviderConfig(token, id);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["creative-provider-configs", token, campaignId]
      });
      await queryClient.invalidateQueries({
        queryKey: ["creative-sync-logs", token, campaignId]
      });
      await queryClient.invalidateQueries({ queryKey: ["creatives", token, campaignId] });
    }
  });

  const updateProviderConfigStatus = useMutation({
    mutationFn: (input: { id: string; status: CreativeProviderConfigStatus }) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.updateCreativeProviderConfigStatus(token, input.id, input.status);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["creative-provider-configs", token, campaignId]
      });
    }
  });

  const estimateAudience = useMutation({
    mutationFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign launch input");
      }
      return api.estimateCampaignAudience(token, campaignId);
    }
  });

  const createLaunch = useMutation({
    mutationFn: () => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign launch input");
      }
      return api.createCampaignLaunch(token, campaignId);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["campaign-launches", token, campaignId]
      });
    }
  });

  const enqueueLaunch = useMutation({
    mutationFn: (launchId: string) => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign launch input");
      }
      return api.enqueueCampaignLaunch(token, campaignId, launchId);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ["campaign-launches", token, campaignId]
      });
    }
  });

  const createSchedule = useMutation({
    mutationFn: (values: ScheduleForm) => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign schedule input");
      }
      return api.createCampaignSchedule(token, campaignId, {
        fallback_timezone: values.fallback_timezone,
        grace_minutes: values.grace_minutes,
        slots: parseScheduleTimes(values.local_times).map((localTime, index) => ({
          local_time: localTime,
          days_of_week: parseScheduleDays(values.days_of_week),
          position: index + 1
        }))
      });
    },
    onSuccess: async () => {
      scheduleForm.reset({
        local_times: "10:00, 10:45",
        days_of_week: "1,2,3,4,5,6,7",
        fallback_timezone: "UTC",
        grace_minutes: 10
      });
      setCreateScheduleOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["campaign-schedules", token, campaignId] });
    }
  });

  const updateScheduleStatus = useMutation({
    mutationFn: (input: { id: string; status: CampaignScheduleStatus }) => {
      if (!token || !campaignId) {
        throw new Error("Missing campaign schedule input");
      }
      return api.updateCampaignScheduleStatus(token, campaignId, input.id, input.status);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["campaign-schedules", token, campaignId] });
    }
  });

  const selectedCampaign = (campaigns.data?.items ?? []).find(
    (campaign) => campaign.id === campaignId
  );
  const creativeItems = useMemo(() => creatives.data?.items ?? [], [creatives.data?.items]);
  const filteredCreatives = useMemo(
    () =>
      creativeItems.filter((creative) => {
        const sourceMatches =
          creativeSourceFilter === "all" || creative.source_type === creativeSourceFilter;
        const syncMatches =
          creativeSyncFilter === "all" ||
          (creative.source_type === "provider_api" && creative.sync_status === creativeSyncFilter);
        const statusMatches =
          creativeStatusFilter === "all" || creative.status === creativeStatusFilter;
        return sourceMatches && syncMatches && statusMatches;
      }),
    [creativeItems, creativeSourceFilter, creativeStatusFilter, creativeSyncFilter]
  );
  const creativeSummary = useMemo(() => creativeWorkspaceSummary(creativeItems), [creativeItems]);

  const creativeColumns = useMemo<ColumnDef<Creative>[]>(
    () => [
      { header: "Title", accessorKey: "title" },
      { header: "URL", accessorKey: "url" },
      {
        header: "Source",
        cell: ({ row }) =>
          row.original.source_type === "provider_api"
            ? `${row.original.provider_name ?? "provider"}:${row.original.provider_external_id ?? ""}`
            : "manual"
      },
      {
        header: "Caps",
        cell: ({ row }) =>
          `${String(row.original.daily_cap_per_subscription)} daily / ${String(row.original.total_cap_per_subscription)} total`
      },
      {
        header: "Status",
        cell: ({ row }) => (
          <StatusSelect<CreativeStatus>
            disabled={updateCreativeStatus.isPending}
            options={creativeStatuses}
            value={row.original.status}
            onChange={(status) => {
              updateCreativeStatus.mutate({ id: row.original.id, status });
            }}
          />
        )
      }
    ],
    [updateCreativeStatus]
  );

  const creativeTable = useReactTable({
    data: filteredCreatives,
    columns: creativeColumns,
    getCoreRowModel: getCoreRowModel()
  });

  function submitCreative(values: CreativeForm) {
    createCreative.mutate(values);
  }

  function submitProviderConfig(values: ProviderConfigForm) {
    createProviderConfig.mutate(values);
  }

  function submitSchedule(values: ScheduleForm) {
    createSchedule.mutate(values);
  }

  return (
    <section className="space-y-4">
      <div>
        <Link
          className="mb-3 inline-flex h-8 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
          to="/campaigns"
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </Link>
        <h1 className="text-xl font-semibold text-zinc-950">
          {selectedCampaign?.name ?? "Campaign details"}
        </h1>
        <p className="mt-1 text-sm text-zinc-600">Creative, launch and provider setup.</p>
      </div>

      {campaigns.isError ? <ErrorMessage message={campaigns.error.message} /> : null}

      <div className="space-y-3">
        <div className="space-y-3">
          <CampaignReportPanel
            creatives={creativeItems}
            isLoading={campaignReport.isLoading}
            report={campaignReport.data}
          />
          <CampaignLaunchPanel
            estimate={estimateAudience.data}
            isBuilding={createLaunch.isPending}
            isEstimating={estimateAudience.isPending}
            launches={campaignLaunches.data?.items ?? []}
            selected={Boolean(campaignId)}
            onBuild={() => {
              createLaunch.mutate();
            }}
            onEnqueue={(launchId) => {
              enqueueLaunch.mutate(launchId);
            }}
            onEstimate={() => {
              estimateAudience.mutate();
            }}
          />
          <CampaignSchedulePanel
            isUpdatingStatus={updateScheduleStatus.isPending}
            runs={campaignScheduleRuns.data?.items ?? []}
            schedules={campaignSchedules.data?.items ?? []}
            selected={Boolean(campaignId)}
            onAdd={() => {
              setCreateScheduleOpen(true);
            }}
            onStatusChange={(id, status) => {
              updateScheduleStatus.mutate({ id, status });
            }}
          />
          {campaignReport.isError ? <ErrorMessage message={campaignReport.error.message} /> : null}
          {campaignLaunches.isError ? <ErrorMessage message={campaignLaunches.error.message} /> : null}
          {campaignSchedules.isError ? <ErrorMessage message={campaignSchedules.error.message} /> : null}
          {campaignScheduleRuns.isError ? <ErrorMessage message={campaignScheduleRuns.error.message} /> : null}
          {estimateAudience.isError ? <ErrorMessage message={estimateAudience.error.message} /> : null}
          {createLaunch.isError ? <ErrorMessage message={createLaunch.error.message} /> : null}
          {enqueueLaunch.isError ? <ErrorMessage message={enqueueLaunch.error.message} /> : null}
          {createSchedule.isError ? <ErrorMessage message={createSchedule.error.message} /> : null}
          {updateScheduleStatus.isError ? <ErrorMessage message={updateScheduleStatus.error.message} /> : null}
          {updateCreativeStatus.isError ? <ErrorMessage message={updateCreativeStatus.error.message} /> : null}
          <ProviderSyncPanel
            configs={providerConfigs.data?.items ?? []}
            isSyncing={syncProviderConfig.isPending}
            isUpdatingStatus={updateProviderConfigStatus.isPending}
            logs={providerSyncLogs.data?.items ?? []}
            selected={Boolean(campaignId)}
            onAdd={() => {
              setCreateProviderOpen(true);
            }}
            onStatusChange={(id, status) => {
              updateProviderConfigStatus.mutate({ id, status });
            }}
            onSync={(id) => {
              syncProviderConfig.mutate(id);
            }}
          />
          {providerConfigs.isError ? <ErrorMessage message={providerConfigs.error.message} /> : null}
          {providerSyncLogs.isError ? <ErrorMessage message={providerSyncLogs.error.message} /> : null}
          {syncProviderConfig.isError ? <ErrorMessage message={syncProviderConfig.error.message} /> : null}
          {updateProviderConfigStatus.isError ? <ErrorMessage message={updateProviderConfigStatus.error.message} /> : null}
          <CreativeWorkspaceFilters
            filteredTotal={filteredCreatives.length}
            sourceFilter={creativeSourceFilter}
            statusFilter={creativeStatusFilter}
            summary={creativeSummary}
            syncFilter={creativeSyncFilter}
            total={creativeItems.length}
            onCreate={() => {
              setCreateCreativeOpen(true);
            }}
            onSourceFilterChange={setCreativeSourceFilter}
            onStatusFilterChange={setCreativeStatusFilter}
            onSyncFilterChange={setCreativeSyncFilter}
          />
          <DataTable<Creative>
            emptyText="No creatives yet"
            isLoading={creatives.isLoading}
            table={creativeTable}
          />
        </div>
      </div>

      {createCreativeOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="max-h-[90vh] w-full max-w-xl overflow-auto rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Create creative</h2>
              <button
                className="inline-flex h-8 w-8 items-center justify-center rounded-md text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-950"
                type="button"
                onClick={() => {
                  creativeForm.reset({
                    campaign_id: campaignId ?? "",
                    title: "",
                    body: "",
                    url: "",
                    icon: "",
                    daily_cap_per_subscription: 0,
                    total_cap_per_subscription: 0
                  });
                  setCreateCreativeOpen(false);
                }}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <form
              className="space-y-3 p-4"
              onSubmit={(event) => {
                void creativeForm.handleSubmit(submitCreative)(event);
              }}
            >
              <input type="hidden" {...creativeForm.register("campaign_id")} />
              <TextField label="Title" placeholder="Notification title" register={creativeForm.register("title")} />
              <TextField label="Body" placeholder="Notification body" register={creativeForm.register("body")} />
              <TextField label="URL" placeholder="https://example.com" register={creativeForm.register("url")} />
              <TextField label="Icon" placeholder="https://example.com/icon.png" register={creativeForm.register("icon")} />
              <NumberField
                label="Daily cap"
                register={creativeForm.register("daily_cap_per_subscription", { valueAsNumber: true })}
              />
              <NumberField
                label="Total cap"
                register={creativeForm.register("total_cap_per_subscription", { valueAsNumber: true })}
              />
              <ErrorList errors={Object.values(creativeForm.formState.errors).map((error) => error.message)} />
              {createCreative.isError ? <ErrorMessage message={createCreative.error.message} /> : null}
              <div className="flex justify-end gap-2">
                <button
                  className="inline-flex h-10 items-center justify-center rounded-md border border-zinc-300 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50"
                  type="button"
                  onClick={() => {
                    creativeForm.reset({
                      campaign_id: campaignId ?? "",
                      title: "",
                      body: "",
                      url: "",
                      icon: "",
                      daily_cap_per_subscription: 0,
                      total_cap_per_subscription: 0
                    });
                    setCreateCreativeOpen(false);
                  }}
                >
                  Cancel
                </button>
                <button
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
                  disabled={createCreative.isPending}
                  type="submit"
                >
                  {createCreative.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  Create creative
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {createScheduleOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="max-h-[90vh] w-full max-w-xl overflow-auto rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Create schedule</h2>
              <button
                className="inline-flex h-8 w-8 items-center justify-center rounded-md text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-950"
                type="button"
                onClick={() => {
                  setCreateScheduleOpen(false);
                }}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <form
              className="space-y-3 p-4"
              onSubmit={(event) => {
                void scheduleForm.handleSubmit(submitSchedule)(event);
              }}
            >
              <TextField
                label="Local times"
                placeholder="10:00, 10:45, 18:30"
                register={scheduleForm.register("local_times")}
              />
              <TextField
                label="Days of week"
                placeholder="1,2,3,4,5,6,7"
                register={scheduleForm.register("days_of_week")}
              />
              <TextField
                label="Fallback timezone"
                placeholder="UTC"
                register={scheduleForm.register("fallback_timezone")}
              />
              <NumberField
                label="Grace minutes"
                register={scheduleForm.register("grace_minutes", { valueAsNumber: true })}
              />
              <ErrorList errors={Object.values(scheduleForm.formState.errors).map((error) => error.message)} />
              {createSchedule.isError ? <ErrorMessage message={createSchedule.error.message} /> : null}
              <div className="flex justify-end gap-2">
                <button
                  className="inline-flex h-10 items-center justify-center rounded-md border border-zinc-300 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50"
                  type="button"
                  onClick={() => {
                    setCreateScheduleOpen(false);
                  }}
                >
                  Cancel
                </button>
                <button
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
                  disabled={createSchedule.isPending}
                  type="submit"
                >
                  {createSchedule.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  Create schedule
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {createProviderOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="max-h-[90vh] w-full max-w-xl overflow-auto rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Add provider</h2>
              <button
                className="inline-flex h-8 w-8 items-center justify-center rounded-md text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-950"
                type="button"
                onClick={() => {
                  providerConfigForm.reset({
                    campaign_id: campaignId ?? "",
                    name: "",
                    provider_name: "generic",
                    fetch_url: "",
                    request_headers: "{}"
                  });
                  setCreateProviderOpen(false);
                }}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <form
              className="space-y-3 p-4"
              onSubmit={(event) => {
                void providerConfigForm.handleSubmit(submitProviderConfig)(event);
              }}
            >
              <input type="hidden" {...providerConfigForm.register("campaign_id")} />
              <TextField label="Name" placeholder="Provider feed" register={providerConfigForm.register("name")} />
              <TextField label="Provider" placeholder="generic" register={providerConfigForm.register("provider_name")} />
              <TextField label="Fetch URL" placeholder="https://example.com/creatives.json" register={providerConfigForm.register("fetch_url")} />
              <TextField label="Headers JSON" placeholder='{"Authorization":"Bearer ..."}' register={providerConfigForm.register("request_headers")} />
              <ErrorList errors={Object.values(providerConfigForm.formState.errors).map((error) => error.message)} />
              {createProviderConfig.isError ? <ErrorMessage message={createProviderConfig.error.message} /> : null}
              <div className="flex justify-end gap-2">
                <button
                  className="inline-flex h-10 items-center justify-center rounded-md border border-zinc-300 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50"
                  type="button"
                  onClick={() => {
                    providerConfigForm.reset({
                      campaign_id: campaignId ?? "",
                      name: "",
                      provider_name: "generic",
                      fetch_url: "",
                      request_headers: "{}"
                    });
                    setCreateProviderOpen(false);
                  }}
                >
                  Cancel
                </button>
                <button
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
                  disabled={createProviderConfig.isPending}
                  type="submit"
                >
                  {createProviderConfig.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  Add provider
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </section>
  );
}

function CampaignReportPanel({
  creatives,
  isLoading,
  report
}: {
  creatives: Creative[];
  isLoading: boolean;
  report?: CampaignReport;
}) {
  if (!report) {
    return (
      <div className="rounded-md border border-zinc-200 bg-white p-4 text-sm text-zinc-600">
        {isLoading ? "Loading campaign stats" : "Select a campaign to inspect stats."}
      </div>
    );
  }
  const creativeNameById = new Map(
    creatives.map((creative) => [creative.id, creativeLabel(creative)])
  );
  return (
    <div className="space-y-3 rounded-md border border-zinc-200 bg-white p-4">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-zinc-950">Campaign stats</h2>
        <HealthBadge status={report.health.status} />
      </div>
      <div className="grid gap-2 sm:grid-cols-5">
        <Metric label="Decisions" value={report.decisions_total} />
        <Metric label="Selected" value={report.selected} />
        <Metric label="Suppressed" value={report.suppressed} />
        <Metric label="Not found" value={report.not_found} />
        <Metric label="Errors" value={report.errors} />
      </div>
      <div className="grid gap-2 sm:grid-cols-4">
        <Metric label="Shown" value={report.shown} />
        <Metric label="Clicks" value={report.clicks} />
        <Metric label="Closed" value={report.closed} />
        <Metric label="Tracked" value={report.tracked_events} />
      </div>
      {report.health.issues.length > 0 ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          {report.health.issues.join(", ")}
        </div>
      ) : null}
      <div className="grid gap-3 lg:grid-cols-4">
        <Breakdown title="By result" values={report.decisions_by_result} />
        <Breakdown title="By reason" values={report.decisions_by_reason} />
        <Breakdown title="Tracking" values={report.events_by_type} />
        <CreativeExposureList
          creativeNameById={creativeNameById}
          exposures={report.creative_exposures}
        />
      </div>
    </div>
  );
}

function CreativeWorkspaceFilters({
  filteredTotal,
  onCreate,
  onSourceFilterChange,
  onStatusFilterChange,
  onSyncFilterChange,
  sourceFilter,
  statusFilter,
  summary,
  syncFilter,
  total
}: {
  filteredTotal: number;
  onCreate: () => void;
  onSourceFilterChange: (value: CreativeSourceFilter) => void;
  onStatusFilterChange: (value: CreativeStatusFilter) => void;
  onSyncFilterChange: (value: CreativeSyncFilter) => void;
  sourceFilter: CreativeSourceFilter;
  statusFilter: CreativeStatusFilter;
  summary: ReturnType<typeof creativeWorkspaceSummary>;
  syncFilter: CreativeSyncFilter;
  total: number;
}) {
  return (
    <div className="space-y-3 rounded-md border border-zinc-200 bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-zinc-950">Creative workspace</h2>
          <p className="mt-1 text-sm text-zinc-600">
            {filteredTotal}/{total} creatives visible
          </p>
        </div>
        <button
          className="inline-flex h-9 items-center gap-2 rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800"
          type="button"
          onClick={onCreate}
        >
          <Plus className="h-4 w-4" />
          Create creative
        </button>
      </div>
      <div className="grid gap-2 sm:grid-cols-4">
        <Metric label="Manual" value={summary.manual} />
        <Metric label="Provider" value={summary.provider} />
        <Metric label="Stale" value={summary.stale} />
        <Metric label="Active" value={summary.active} />
        </div>
      <div className="grid gap-3 md:grid-cols-3">
        <SelectValueField
          label="Source type"
          options={creativeSourceFilters.map((value) => ({ label: value, value }))}
          value={sourceFilter}
          onChange={(value) => {
            onSourceFilterChange(value as CreativeSourceFilter);
          }}
        />
        <SelectValueField
          label="Sync status"
          options={creativeSyncFilters.map((value) => ({ label: value, value }))}
          value={syncFilter}
          onChange={(value) => {
            onSyncFilterChange(value as CreativeSyncFilter);
          }}
        />
        <SelectValueField
          label="Creative status"
          options={creativeStatusFilters.map((value) => ({ label: value, value }))}
          value={statusFilter}
          onChange={(value) => {
            onStatusFilterChange(value as CreativeStatusFilter);
          }}
        />
      </div>
    </div>
  );
}

function ProviderSyncPanel({
  configs,
  isSyncing,
  isUpdatingStatus,
  logs,
  onAdd,
  onSync,
  onStatusChange,
  selected
}: {
  configs: CreativeProviderConfig[];
  isSyncing: boolean;
  isUpdatingStatus: boolean;
  logs: CreativeSyncLog[];
  onAdd: () => void;
  onSync: (id: string) => void;
  onStatusChange: (id: string, status: CreativeProviderConfigStatus) => void;
  selected: boolean;
}) {
  const latestLogByConfig = new Map<string, CreativeSyncLog>();
  logs.forEach((log) => {
    if (!latestLogByConfig.has(log.provider_config_id)) {
      latestLogByConfig.set(log.provider_config_id, log);
    }
  });

  return (
    <div className="space-y-3 rounded-md border border-zinc-200 bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-zinc-950">Provider sync</h2>
          <p className="mt-1 text-sm text-zinc-600">Fetch creatives from a JSON endpoint.</p>
        </div>
        <button
          className="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
          disabled={!selected}
          type="button"
          onClick={onAdd}
        >
          <Plus className="h-4 w-4" />
          Add provider
        </button>
      </div>
      <div className="space-y-2">
        {configs.length === 0 ? (
          <div className="text-sm text-zinc-500">No provider configs yet</div>
        ) : (
          configs.map((config) => {
            const latestLog = latestLogByConfig.get(config.id);
            return (
              <div
                className="space-y-2 rounded-md border border-zinc-200 px-3 py-2"
                key={config.id}
              >
                <div className="grid gap-2 sm:grid-cols-[1fr_auto_auto_auto]">
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium text-zinc-900">{config.name}</div>
                    <div className="truncate text-xs text-zinc-500">
                      {config.provider_name} | {config.fetch_url}
                    </div>
                  </div>
                  <StatusSelect<CreativeProviderConfigStatus>
                    disabled={isUpdatingStatus}
                    options={providerConfigStatuses}
                    value={config.status}
                    onChange={(status) => {
                      onStatusChange(config.id, status);
                    }}
                  />
                  <span className="self-center text-xs text-zinc-500">
                    {config.last_sync_at ? new Date(config.last_sync_at).toLocaleString() : "never synced"}
                  </span>
                  <button
                    className="inline-flex h-8 items-center justify-center rounded-md border border-zinc-300 bg-white px-2 text-xs text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-60"
                    disabled={isSyncing || config.status !== "active"}
                    type="button"
                    onClick={() => {
                      onSync(config.id);
                    }}
                  >
                    Sync
                  </button>
                </div>
                {latestLog ? <SyncLogSummary log={latestLog} /> : null}
              </div>
            );
          })
        )}
        <div className="rounded-md border border-zinc-200 p-3">
          <div className="text-sm font-medium text-zinc-900">Sync history</div>
          <div className="mt-2 space-y-1 text-sm text-zinc-700">
            {logs.length === 0 ? (
              <div className="text-zinc-500">No sync runs yet</div>
            ) : (
              logs.slice(0, 5).map((log) => <SyncLogSummary key={log.id} log={log} />)
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function SyncLogSummary({ log }: { log: CreativeSyncLog }) {
  return (
    <div className="rounded-md border border-zinc-100 bg-zinc-50 px-2 py-1 text-xs text-zinc-700">
      <div className="flex flex-wrap justify-between gap-2">
        <span>{new Date(log.started_at).toLocaleString()}</span>
        <span className="font-medium">{log.status}</span>
        <span>
          fetched {log.fetched_total} / upserted {log.upserted_total}
        </span>
      </div>
      {log.error_message ? (
        <div className="mt-1 truncate text-red-700">{log.error_message}</div>
      ) : null}
    </div>
  );
}

function CampaignLaunchPanel({
  estimate,
  isBuilding,
  isEstimating,
  launches,
  onEnqueue,
  onBuild,
  onEstimate,
  selected
}: {
  estimate?: { total: number };
  isBuilding: boolean;
  isEstimating: boolean;
  launches: CampaignLaunch[];
  onEnqueue: (launchId: string) => void;
  onBuild: () => void;
  onEstimate: () => void;
  selected: boolean;
}) {
  return (
    <div className="space-y-3 rounded-md border border-zinc-200 bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-zinc-950">Audience launch</h2>
          <p className="mt-1 text-sm text-zinc-600">
            Freeze matching subscribers before delivery queueing.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            className="inline-flex h-9 items-center justify-center rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-60"
            disabled={!selected || isEstimating}
            type="button"
            onClick={onEstimate}
          >
            {isEstimating ? "Estimating" : "Estimate"}
          </button>
          <button
            className="inline-flex h-9 items-center justify-center rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
            disabled={!selected || isBuilding}
            type="button"
            onClick={onBuild}
          >
            {isBuilding ? "Building" : "Build snapshot"}
          </button>
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-3">
        <Metric label="Estimate" value={estimate?.total ?? 0} />
        <Metric label="Launches" value={launches.length} />
        <Metric
          label="Last audience"
          value={launches[0]?.audience_total ?? 0}
        />
      </div>
      <div className="space-y-1 text-sm text-zinc-700">
        {launches.length === 0 ? (
          <div className="text-zinc-500">No launches yet</div>
        ) : (
          launches.slice(0, 5).map((launch) => (
            <div
              className="grid gap-2 rounded-md border border-zinc-200 px-3 py-2 sm:grid-cols-[1fr_auto_auto_auto]"
              key={launch.id}
            >
              <span className="truncate">{launch.id}</span>
              <span>
                {launch.status} / {launch.enqueue_status}
              </span>
              <span className="font-medium">
                {launch.enqueued_total}/{launch.audience_total}
              </span>
              <button
                className="inline-flex h-8 items-center justify-center rounded-md border border-zinc-300 bg-white px-2 text-xs text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-60"
                disabled={launch.status !== "completed" || launch.enqueue_status === "completed"}
                type="button"
                onClick={() => {
                  onEnqueue(launch.id);
                }}
              >
                Enqueue
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function CampaignSchedulePanel({
  isUpdatingStatus,
  onAdd,
  onStatusChange,
  runs,
  schedules,
  selected
}: {
  isUpdatingStatus: boolean;
  onAdd: () => void;
  onStatusChange: (id: string, status: CampaignScheduleStatus) => void;
  runs: CampaignScheduleRun[];
  schedules: CampaignSchedule[];
  selected: boolean;
}) {
  return (
    <div className="space-y-3 rounded-md border border-zinc-200 bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-zinc-950">Campaign schedule</h2>
          <p className="mt-1 text-sm text-zinc-600">
            Subscriber-local delivery slots by timezone.
          </p>
        </div>
        <button
          className="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
          disabled={!selected}
          type="button"
          onClick={onAdd}
        >
          <Plus className="h-4 w-4" />
          Add schedule
        </button>
      </div>
      <div className="space-y-2">
        {schedules.length === 0 ? (
          <div className="text-sm text-zinc-500">No schedules yet</div>
        ) : (
          schedules.map((schedule) => (
            <div
              className="grid gap-2 rounded-md border border-zinc-200 px-3 py-2 lg:grid-cols-[1fr_auto_auto]"
              key={schedule.id}
            >
              <div className="min-w-0">
                <div className="text-sm font-medium text-zinc-900">
                  {schedule.slots.map((slot) => slot.local_time).join(", ")}
                </div>
                <div className="truncate text-xs text-zinc-500">
                  days {formatDays(schedule.slots[0]?.days_of_week ?? [])} | fallback {schedule.fallback_timezone} | grace {schedule.grace_minutes}m
                </div>
              </div>
              <StatusSelect<CampaignScheduleStatus>
                disabled={isUpdatingStatus}
                options={scheduleStatuses}
                value={schedule.status}
                onChange={(status) => {
                  onStatusChange(schedule.id, status);
                }}
              />
              <span className="self-center text-xs text-zinc-500">
                {new Date(schedule.created_at).toLocaleString()}
              </span>
            </div>
          ))
        )}
      </div>
      <div className="rounded-md border border-zinc-200 p-3">
        <div className="text-sm font-medium text-zinc-900">Recent schedule runs</div>
        <div className="mt-2 space-y-1 text-sm text-zinc-700">
          {runs.length === 0 ? (
            <div className="text-zinc-500">No scheduled runs yet</div>
          ) : (
            runs.slice(0, 5).map((run) => (
              <div
                className="grid gap-2 rounded-md bg-zinc-50 px-2 py-1 text-xs sm:grid-cols-[1fr_auto_auto_auto]"
                key={run.id}
              >
                <span>
                  {run.local_date} {run.local_time} {run.timezone}
                </span>
                <span className="font-medium">{run.status}</span>
                <span>
                  {run.enqueued_total}/{run.audience_total}
                </span>
                <span>{new Date(run.scheduled_utc_at).toLocaleString()}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function parseScheduleTimes(value: string): string[] {
  return value
    .split(",")
    .map((part) => part.trim())
    .filter(Boolean);
}

function parseScheduleDays(value: string): number[] {
  const days = value
    .split(",")
    .map((part) => Number.parseInt(part.trim(), 10))
    .filter((day) => Number.isInteger(day) && day >= 1 && day <= 7);
  return days.length > 0 ? days : [1, 2, 3, 4, 5, 6, 7];
}

function formatDays(days: number[]): string {
  return days.length > 0 ? days.join(",") : "1,2,3,4,5,6,7";
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-zinc-200 p-3">
      <div className="text-xs uppercase text-zinc-500">{label}</div>
      <div className="mt-1 text-lg font-semibold text-zinc-950">{value}</div>
    </div>
  );
}

function Breakdown({ title, values }: { title: string; values: Record<string, number> }) {
  const entries = Object.entries(values);
  return (
    <div className="rounded-md border border-zinc-200 p-3">
      <div className="text-sm font-medium text-zinc-900">{title}</div>
      <div className="mt-2 space-y-1 text-sm text-zinc-700">
        {entries.length === 0 ? (
          <div className="text-zinc-500">No data</div>
        ) : (
          entries.map(([key, value]) => (
            <div className="flex justify-between gap-3" key={key}>
              <span className="truncate">{key}</span>
              <span className="font-medium">{value}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function CreativeExposureList({
  creativeNameById,
  exposures
}: {
  creativeNameById: Map<string, string>;
  exposures: { creative_id: string; count: number }[];
}) {
  return (
    <div className="rounded-md border border-zinc-200 p-3">
      <div className="text-sm font-medium text-zinc-900">Creative exposures</div>
      <div className="mt-2 space-y-1 text-sm text-zinc-700">
        {exposures.length === 0 ? (
          <div className="text-zinc-500">No data</div>
        ) : (
          exposures.map((item) => (
            <div className="flex justify-between gap-3" key={item.creative_id}>
              <span className="truncate" title={item.creative_id}>
                {creativeNameById.get(item.creative_id) ?? shortID(item.creative_id)}
              </span>
              <span className="font-medium">{item.count}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function creativeLabel(creative: Creative): string {
  if (creative.provider_external_id) {
    return `${creative.title} (${creative.provider_external_id})`;
  }
  return creative.title;
}

function shortID(value: string): string {
  return value.length > 12 ? `${value.slice(0, 8)}...${value.slice(-4)}` : value;
}

function HealthBadge({ status }: { status: "ok" | "attention" }) {
  return (
    <span className="inline-flex h-7 items-center rounded-md border border-zinc-200 px-2 text-xs font-medium text-zinc-700">
      {status}
    </span>
  );
}

function targetingRulesFromForm(values: CampaignForm): TargetingRules {
  return {
    countries: splitList(values.countries),
    languages: splitList(values.languages),
    device_types: splitList(values.device_types),
    os_names: splitList(values.os_names),
    browser_names: splitList(values.browser_names)
  };
}

function splitList(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim().toLowerCase())
    .filter((item, index, items) => item.length > 0 && items.indexOf(item) === index);
}

function creativeWorkspaceSummary(creatives: Creative[]) {
  return creatives.reduce(
    (summary, creative) => {
      if (creative.source_type === "provider_api") {
        summary.provider += 1;
      } else {
        summary.manual += 1;
      }
      if (creative.sync_status === "stale") {
        summary.stale += 1;
      }
      if (creative.status === "active") {
        summary.active += 1;
      }
      return summary;
    },
    {
      active: 0,
      manual: 0,
      provider: 0,
      stale: 0
    }
  );
}

function parseHeaders(value: string): Record<string, string> {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }
  const parsed: unknown = JSON.parse(trimmed);
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("Headers JSON must be an object");
  }
  const headers: Record<string, string> = {};
  Object.entries(parsed).forEach(([key, headerValue]) => {
    if (typeof headerValue === "string" && key.trim() && headerValue.trim()) {
      headers[key.trim()] = headerValue.trim();
    }
  });
  return headers;
}

function audienceSummary(campaign: Campaign, sources: Source[]): string {
  if (campaign.audience_scope === "all") {
    return "All sources";
  }
  const sourceIDs = campaign.source_ids.length > 0 ? campaign.source_ids : campaign.source_id ? [campaign.source_id] : [];
  if (sourceIDs.length === 0) {
    return "No sources";
  }
  if (sourceIDs.length <= 2) {
    return sourceIDs.map((sourceID) => sourceName(sources, sourceID)).join(", ");
  }
  return `${String(sourceIDs.length)} sources`;
}

function sourceName(sources: Source[], sourceId: string): string {
  return sources.find((source) => source.id === sourceId)?.name ?? sourceId;
}

function publisherName(publishers: Publisher[], publisherId: string): string {
  return publishers.find((publisher) => publisher.id === publisherId)?.name ?? publisherId;
}

function targetingSummary(rules: TargetingRules): string {
  const parts = [
    labelList("geo", rules.countries),
    labelList("lang", rules.languages),
    labelList("device", rules.device_types),
    labelList("os", rules.os_names),
    labelList("browser", rules.browser_names)
  ].filter((part) => part !== "");
  return parts.length > 0 ? parts.join(" | ") : "unrestricted";
}

function labelList(label: string, values?: string[]): string {
  return values && values.length > 0 ? `${label}: ${values.join(", ")}` : "";
}

function TextField({
  label,
  placeholder,
  register
}: {
  label: string;
  placeholder: string;
  register: UseFormRegisterReturn;
}) {
  return (
    <label className="text-sm font-medium text-zinc-800">
      {label}
      <input
        className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
        placeholder={placeholder}
        {...register}
      />
    </label>
  );
}

function NumberField({
  label,
  register
}: {
  label: string;
  register: UseFormRegisterReturn;
}) {
  return (
    <label className="text-sm font-medium text-zinc-800">
      {label}
      <input
        className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
        min={0}
        type="number"
        {...register}
      />
    </label>
  );
}

function SelectField({
  label,
  options,
  placeholder,
  register
}: {
  label: string;
  options: { label: string; value: string }[];
  placeholder: string;
  register: UseFormRegisterReturn;
}) {
  return (
    <label className="text-sm font-medium text-zinc-800">
      {label}
      <select
        className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
        {...register}
      >
        <option value="">{placeholder}</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function SelectValueField({
  label,
  onChange,
  options,
  value
}: {
  label: string;
  onChange: (value: string) => void;
  options: { label: string; value: string }[];
  value: string;
}) {
  return (
    <label className="text-sm font-medium text-zinc-800">
      {label}
      <select
        className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
        value={value}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function StatusSelect<TStatus extends string>({
  disabled,
  options,
  value,
  onChange
}: {
  disabled: boolean;
  options: TStatus[];
  value: TStatus;
  onChange: (value: TStatus) => void;
}) {
  return (
    <select
      className="h-8 rounded-md border border-zinc-300 bg-white px-2 text-sm outline-none transition focus:border-zinc-950 disabled:opacity-60"
      disabled={disabled}
      value={value}
      onChange={(event) => {
        onChange(event.target.value as TStatus);
      }}
    >
      {options.map((option) => (
        <option key={option} value={option}>
          {option}
        </option>
      ))}
    </select>
  );
}

function ErrorList({ errors }: { errors: (string | undefined)[] }) {
  return (
    <>
      {errors
        .filter((message): message is string => Boolean(message))
        .map((message) => (
          <ErrorMessage key={message} message={message} />
        ))}
    </>
  );
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800">
      {message}
    </div>
  );
}

function DataTable<TItem>({
  table,
  isLoading,
  emptyText
}: {
  table: ReturnType<typeof useReactTable<TItem>>;
  isLoading: boolean;
  emptyText: string;
}) {
  return (
    <div className="overflow-hidden rounded-md border border-zinc-200 bg-white">
      <table className="w-full border-collapse text-left text-sm">
        <thead className="border-b border-zinc-200 bg-zinc-50 text-xs uppercase text-zinc-500">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th className="px-4 py-3 font-medium" key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr className="border-b border-zinc-100 last:border-0" key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td className="px-4 py-3 align-middle" key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      {table.getRowModel().rows.length === 0 ? (
        <div className="p-6 text-sm text-zinc-600">{isLoading ? "Loading" : emptyText}</div>
      ) : null}
    </div>
  );
}
