import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { AlertTriangle, ArrowLeft, CheckCircle2, Code2, Copy, Loader2, Plus, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link, Navigate, useParams } from "react-router";
import { Bar, BarChart, ResponsiveContainer, XAxis, YAxis } from "recharts";
import { z } from "zod";

import { Source, SourceStats, api } from "../api/client";
import { useAuthStore } from "../store/auth";

const sourceSchema = z.object({
  name: z.string().trim().min(2),
  domain: z.string().trim().min(3)
});

type SourceForm = z.infer<typeof sourceSchema>;

export function SourcesPage() {
  const { publisherId } = useParams<{ publisherId: string }>();
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();
  const [snippetSourceId, setSnippetSourceId] = useState<string | null>(null);
  const [detailsSourceId, setDetailsSourceId] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const form = useForm<SourceForm>({
    resolver: zodResolver(sourceSchema),
    defaultValues: { name: "", domain: "" }
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
    queryKey: ["sources", token, publisherId],
    queryFn: () => {
      if (!token || !publisherId) {
        throw new Error("Missing source page input");
      }
      return api.sources(token, publisherId);
    },
    enabled: Boolean(token && publisherId)
  });

  const stats = useQueries({
    queries: (sources.data?.items ?? []).map((source) => ({
      queryKey: ["source-stats", token, source.id],
      queryFn: () => {
        if (!token) {
          throw new Error("Missing auth token");
        }
        return api.sourceStats(token, source.id);
      },
      enabled: Boolean(token)
    }))
  });

  const snippet = useQuery({
    queryKey: ["source-snippet", token, snippetSourceId],
    queryFn: () => {
      if (!token || !snippetSourceId) {
        throw new Error("Missing source snippet input");
      }
      return api.sourceSnippet(token, snippetSourceId);
    },
    enabled: Boolean(token && snippetSourceId)
  });

  const createSource = useMutation({
    mutationFn: (values: SourceForm) => {
      if (!token || !publisherId) {
        throw new Error("Missing source page input");
      }
      return api.createSource(token, {
        publisher_id: publisherId,
        name: values.name,
        domain: values.domain
      });
    },
    onSuccess: async () => {
      form.reset();
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["sources", token, publisherId] });
    }
  });

  const statsBySource = useMemo(
    () =>
      new Map(
        (sources.data?.items ?? []).map((source, index) => [
          source.id,
          stats[index]?.data
        ])
      ),
    [sources.data?.items, stats]
  );
  const selectedSource = (sources.data?.items ?? []).find((source) => source.id === detailsSourceId);
  const selectedStats = detailsSourceId ? statsBySource.get(detailsSourceId) : undefined;
  const publisher = (publishers.data?.items ?? []).find((item) => item.id === publisherId);

  const columns = useMemo<ColumnDef<Source>[]>(
    () => [
      { header: "Name", accessorKey: "name" },
      { header: "Domain", accessorKey: "domain" },
      {
        header: "Subscribers",
        cell: ({ row }) => statsBySource.get(row.original.id)?.subscribers ?? 0
      },
      {
        header: "Today",
        cell: ({ row }) => statsBySource.get(row.original.id)?.subscribers_today ?? 0
      },
      {
        header: "Health",
        cell: ({ row }) => {
          const health = statsBySource.get(row.original.id)?.health;
          return <HealthBadge status={health?.status ?? "attention"} />;
        }
      },
      {
        header: "Demo",
        cell: ({ row }) => (
          <Link
            className="text-sm font-medium text-zinc-950 underline-offset-4 hover:underline"
            to={`/demo/${row.original.id}`}
          >
            Open
          </Link>
        )
      },
      {
        header: "Actions",
        id: "snippet",
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <button
              className="inline-flex h-8 items-center rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
              type="button"
              onClick={() => {
                setDetailsSourceId(row.original.id);
              }}
            >
              Details
            </button>
            <button
              className="inline-flex h-8 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
              type="button"
              onClick={() => {
                setSnippetSourceId(row.original.id);
              }}
            >
              <Code2 className="h-4 w-4" />
              Snippet
            </button>
          </div>
        )
      }
    ],
    [statsBySource]
  );

  const table = useReactTable({
    data: sources.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  function onSubmit(values: SourceForm) {
    createSource.mutate(values);
  }

  if (!publisherId) {
    return <Navigate to="/publishers" replace />;
  }

  return (
    <section className="space-y-4">
      <div className="flex items-end justify-between gap-4">
        <div>
          <Link
            className="mb-3 inline-flex h-8 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
            to="/publishers"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Link>
          <h1 className="text-xl font-semibold text-zinc-950">Sources</h1>
          <div className="mt-2 inline-flex items-center rounded-md border border-zinc-200 bg-white px-3 py-1.5 text-sm font-semibold text-zinc-950">
            {publisher?.name ?? "Publisher"}
          </div>
          <p className="mt-2 text-sm text-zinc-600">Sites and landings that collect subscribers.</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="text-sm text-zinc-500">{sources.data?.total ?? 0} total</div>
          <button
            className="inline-flex h-9 items-center gap-2 rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800"
            type="button"
            onClick={() => {
              setCreateOpen(true);
            }}
          >
            <Plus className="h-4 w-4" />
            Create
          </button>
        </div>
      </div>

      {sources.isError ? <ErrorMessage message={sources.error.message} /> : null}

      <DataTable table={table} isLoading={sources.isLoading} emptyText="No sources yet" />

      {selectedSource ? (
        <SourceDetails source={selectedSource} stats={selectedStats} />
      ) : null}

      {snippetSourceId ? (
        <section className="space-y-3 rounded-md border border-zinc-200 bg-white p-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-sm font-semibold text-zinc-950">Install snippet</h2>
            <button
              className="inline-flex h-8 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
              disabled={!snippet.data?.snippet}
              type="button"
              onClick={() => {
                if (snippet.data?.snippet) {
                  void navigator.clipboard.writeText(snippet.data.snippet);
                }
              }}
            >
              <Copy className="h-4 w-4" />
              Copy
            </button>
          </div>
          <pre className="max-h-72 overflow-auto rounded-md bg-zinc-950 p-4 text-xs text-zinc-50">
            {snippet.isLoading ? "Loading snippet" : snippet.data?.snippet}
          </pre>
        </section>
      ) : null}

      {createOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="w-full max-w-xl rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Create source</h2>
              <button
                className="inline-flex h-8 w-8 items-center justify-center rounded-md text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-950"
                type="button"
                onClick={() => {
                  form.reset();
                  setCreateOpen(false);
                }}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <form
              className="space-y-4 p-4"
              onSubmit={(event) => {
                void form.handleSubmit(onSubmit)(event);
              }}
            >
              <label className="block text-sm font-medium text-zinc-800">
                Name
                <input
                  className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
                  placeholder="News source"
                  {...form.register("name")}
                />
              </label>
              <label className="block text-sm font-medium text-zinc-800">
                Domain
                <input
                  className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
                  placeholder="example.com"
                  {...form.register("domain")}
                />
              </label>
              {Object.values(form.formState.errors).map((error) => (
                <ErrorMessage key={error.message} message={error.message ?? "Invalid source input"} />
              ))}
              {createSource.isError ? <ErrorMessage message={createSource.error.message} /> : null}
              <div className="flex justify-end gap-2">
                <button
                  className="inline-flex h-10 items-center justify-center rounded-md border border-zinc-300 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50"
                  type="button"
                  onClick={() => {
                    form.reset();
                    setCreateOpen(false);
                  }}
                >
                  Cancel
                </button>
                <button
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
                  disabled={createSource.isPending}
                  type="submit"
                >
                  {createSource.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </section>
  );
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800">
      {message}
    </div>
  );
}

function SourceDetails({ source, stats }: { source: Source; stats?: SourceStats }) {
  const chartData = [
    { label: "Total", value: stats?.subscribers ?? 0 },
    { label: "Today", value: stats?.subscribers_today ?? 0 },
    { label: "Events", value: stats?.events_today ?? 0 }
  ];
  const breakdown = stats?.event_breakdown ?? {};
  const eventRows = [
    ["Subscribed", breakdown.subscribed ?? 0],
    ["Push received", breakdown.push_received ?? 0],
    ["Shown", breakdown.notification_shown ?? 0],
    ["Click", breakdown.notification_click ?? 0],
    ["Close", breakdown.notification_close ?? 0],
    ["Payload failed", breakdown.payload_failed ?? 0]
  ] as const;
  const healthIssues = stats?.health.issues ?? [];

  return (
    <section className="grid gap-4 rounded-md border border-zinc-200 bg-white p-4 lg:grid-cols-[1fr_320px]">
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-semibold text-zinc-950">{source.name}</h2>
          <HealthBadge status={stats?.health.status ?? "attention"} />
        </div>
        <div className="grid gap-2 text-sm text-zinc-600 sm:grid-cols-2">
          <div>Domain: {source.domain}</div>
          <div>Status: {source.status}</div>
          <div className="break-all">VAPID key ID: {source.vapid_key_id || "fallback"}</div>
          <div className="break-all">Source ID: {source.id}</div>
          <div className="break-all">Publisher ID: {source.publisher_id}</div>
          <div>Last event: {stats?.last_event_at || "none"}</div>
        </div>
        <div className="pt-2">
          <h3 className="text-xs font-semibold uppercase text-zinc-500">Diagnostics</h3>
          <div className="mt-2 flex flex-wrap gap-2">
            {healthIssues.length === 0 ? (
              <span className="rounded-md bg-emerald-50 px-2 py-1 text-xs text-emerald-700">No issues</span>
            ) : (
              healthIssues.map((issue) => (
                <span className="rounded-md bg-amber-50 px-2 py-1 text-xs text-amber-800" key={issue}>
                  {issue.replaceAll("_", " ")}
                </span>
              ))
            )}
          </div>
        </div>
        <div className="grid gap-2 pt-2 text-sm sm:grid-cols-3">
          {eventRows.map(([label, value]) => (
            <div className="rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2" key={label}>
              <div className="text-xs text-zinc-500">{label}</div>
              <div className="mt-1 font-semibold text-zinc-950">{value}</div>
            </div>
          ))}
        </div>
      </div>
      <div className="h-40">
        <ResponsiveContainer height="100%" width="100%">
          <BarChart data={chartData}>
            <XAxis dataKey="label" tickLine={false} />
            <YAxis allowDecimals={false} width={32} />
            <Bar dataKey="value" fill="#18181b" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </section>
  );
}

function HealthBadge({ status }: { status: "ok" | "attention" }) {
  if (status === "ok") {
    return (
      <span className="inline-flex h-7 items-center gap-1 rounded-md bg-emerald-50 px-2 text-xs font-medium text-emerald-700">
        <CheckCircle2 className="h-3.5 w-3.5" />
        OK
      </span>
    );
  }
  return (
    <span className="inline-flex h-7 items-center gap-1 rounded-md bg-amber-50 px-2 text-xs font-medium text-amber-800">
      <AlertTriangle className="h-3.5 w-3.5" />
      Attention
    </span>
  );
}

function DataTable({
  table,
  isLoading,
  emptyText
}: {
  table: ReturnType<typeof useReactTable<Source>>;
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
        <div className="p-6 text-sm text-zinc-600">{isLoading ? "Loading sources" : emptyText}</div>
      ) : null}
    </div>
  );
}
