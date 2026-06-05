import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { ArrowLeft, Inbox, Loader2, Plus, X } from "lucide-react";
import { useMemo, useState } from "react";
import { UseFormRegisterReturn, useForm } from "react-hook-form";
import { Link, useParams } from "react-router";
import { z } from "zod";

import {
  PostbackConfig,
  PostbackConfigStatus,
  PostbackEvent,
  api
} from "../api/client";
import { useAuthStore } from "../store/auth";

const configSchema = z.object({
  name: z.string().trim().min(2),
  source_id: z.string().trim(),
  token: z.string().trim(),
  click_id_param: z.string().trim().min(1),
  delivery_id_param: z.string().trim().min(1),
  subscription_id_param: z.string().trim().min(1),
  external_id_param: z.string().trim().min(1),
  payout_param: z.string().trim().min(1),
  currency_param: z.string().trim().min(1),
  status_param: z.string().trim().min(1),
  default_currency: z.string().trim().min(3)
});

type ConfigForm = z.infer<typeof configSchema>;

const statuses: PostbackConfigStatus[] = ["active", "paused", "archived"];

export function PostbacksPage() {
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const form = useForm<ConfigForm>({
    resolver: zodResolver(configSchema),
    defaultValues: {
      name: "",
      source_id: "",
      token: "",
      click_id_param: "click_id",
      delivery_id_param: "delivery_id",
      subscription_id_param: "subscription_id",
      external_id_param: "external_id",
      payout_param: "payout",
      currency_param: "currency",
      status_param: "status",
      default_currency: "USD"
    }
  });

  const configs = useQuery({
    queryKey: ["postback-configs", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.postbackConfigs(token);
    },
    enabled: Boolean(token)
  });

  const createConfig = useMutation({
    mutationFn: (values: ConfigForm) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.createPostbackConfig(token, values);
    },
    onSuccess: async () => {
      form.reset();
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["postback-configs", token] });
    }
  });

  const updateStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: PostbackConfigStatus }) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.updatePostbackConfigStatus(token, id, status);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["postback-configs", token] });
    }
  });

  const configColumns = useMemo<ColumnDef<PostbackConfig>[]>(
    () => [
      { header: "Name", accessorKey: "name" },
      {
        header: "Endpoint",
        cell: ({ row }) => (
          <code className="block max-w-sm truncate text-xs text-zinc-600">
            {`/v1/postbacks/${row.original.id}?token=${row.original.token ?? ""}`}
          </code>
        )
      },
      {
        header: "Mapping",
        cell: ({ row }) => (
          <div className="text-xs text-zinc-600">
            {row.original.click_id_param} / {row.original.external_id_param} /{" "}
            {row.original.payout_param}
          </div>
        )
      },
      {
        header: "Status",
        cell: ({ row }) => (
          <select
            className="h-8 rounded-md border border-zinc-300 bg-white px-2 text-sm outline-none transition focus:border-zinc-950"
            disabled={updateStatus.isPending}
            value={row.original.status}
            onChange={(event) => {
              updateStatus.mutate({
                id: row.original.id,
                status: event.target.value as PostbackConfigStatus
              });
            }}
          >
            {statuses.map((status) => (
              <option key={status} value={status}>
                {status}
              </option>
            ))}
          </select>
        )
      },
      {
        header: "Actions",
        cell: ({ row }) => (
          <Link
            className="inline-flex h-8 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
            to={`/postbacks/${row.original.id}`}
          >
            <Inbox className="h-4 w-4" />
            Recent
          </Link>
        )
      }
    ],
    [updateStatus]
  );

  const configTable = useReactTable({
    data: configs.data?.items ?? [],
    columns: configColumns,
    getCoreRowModel: getCoreRowModel()
  });

  function onCreate(values: ConfigForm) {
    createConfig.mutate(values);
  }

  return (
    <section className="space-y-4">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-zinc-950">Postbacks</h1>
          <p className="mt-1 text-sm text-zinc-600">
            Incoming affiliate conversion endpoints.
          </p>
        </div>
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

      {configs.isError ? <ErrorMessage message={configs.error.message} /> : null}
      {updateStatus.isError ? <ErrorMessage message={updateStatus.error.message} /> : null}

      <DataTable table={configTable} empty="No postback configs yet." />

      {createOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="max-h-[90vh] w-full max-w-3xl overflow-auto rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Create postback endpoint</h2>
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
              className="grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-4"
              onSubmit={(event) => {
                void form.handleSubmit(onCreate)(event);
              }}
            >
              <Field label="Name" registration={form.register("name")} />
              <Field label="Source ID" registration={form.register("source_id")} />
              <Field label="Token" registration={form.register("token")} />
              <Field label="Default currency" registration={form.register("default_currency")} />
              <Field label="Click param" registration={form.register("click_id_param")} />
              <Field label="External ID param" registration={form.register("external_id_param")} />
              <Field label="Payout param" registration={form.register("payout_param")} />
              <Field label="Status param" registration={form.register("status_param")} />
              <div className="space-y-3 sm:col-span-2 lg:col-span-4">
                {form.formState.errors.name ? (
                  <ErrorMessage message={form.formState.errors.name.message ?? "Invalid name"} />
                ) : null}
                {createConfig.isError ? <ErrorMessage message={createConfig.error.message} /> : null}
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
                    disabled={createConfig.isPending}
                    type="submit"
                  >
                    {createConfig.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Plus className="h-4 w-4" />
                    )}
                    Create
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

export function PostbackEventsPage() {
  const { postbackConfigId } = useParams<{ postbackConfigId: string }>();
  const token = useAuthStore((state) => state.token);

  const configs = useQuery({
    queryKey: ["postback-configs", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.postbackConfigs(token);
    },
    enabled: Boolean(token)
  });

  const events = useQuery({
    queryKey: ["postbacks", token, postbackConfigId],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.postbacks(token, 100, postbackConfigId);
    },
    enabled: Boolean(token && postbackConfigId)
  });

  const config = (configs.data?.items ?? []).find((item) => item.id === postbackConfigId);
  const eventColumns = useMemo<ColumnDef<PostbackEvent>[]>(postbackEventColumns, []);
  const eventTable = useReactTable({
    data: events.data?.items ?? [],
    columns: eventColumns,
    getCoreRowModel: getCoreRowModel()
  });

  return (
    <section className="space-y-4">
      <div>
        <Link
          className="mb-3 inline-flex h-8 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
          to="/postbacks"
        >
          <ArrowLeft className="h-4 w-4" />
          Back
        </Link>
        <div className="flex items-center gap-2">
          <Inbox className="h-4 w-4 text-zinc-500" />
          <h1 className="text-xl font-semibold text-zinc-950">Recent postbacks</h1>
        </div>
        <p className="mt-1 text-sm text-zinc-600">
          {config?.name ?? "Postback endpoint"} · {events.data?.total ?? 0} recent
        </p>
      </div>

      {configs.isError ? <ErrorMessage message={configs.error.message} /> : null}
      {events.isError ? <ErrorMessage message={events.error.message} /> : null}

      <DataTable table={eventTable} empty="No postbacks received yet." />
    </section>
  );
}

function postbackEventColumns(): ColumnDef<PostbackEvent>[] {
  return [
    {
      header: "Received",
      cell: ({ row }) => new Date(row.original.received_at).toLocaleString()
    },
    { header: "Status", accessorKey: "status" },
    {
      header: "Revenue",
      cell: ({ row }) => `${row.original.payout.toFixed(2)} ${row.original.currency}`
    },
    {
      header: "Attribution",
      cell: ({ row }) => (
        <span
          className={
            row.original.attribution_status === "resolved"
              ? "text-emerald-700"
              : "text-amber-700"
          }
        >
          {row.original.attribution_status}
        </span>
      )
    },
    {
      header: "IDs",
      cell: ({ row }) => (
        <code className="block max-w-md truncate text-xs text-zinc-600">
          {row.original.delivery_id || row.original.subscription_id || row.original.dedupe_key}
        </code>
      )
    },
    {
      header: "Raw",
      cell: ({ row }) => (
        <code className="block max-w-xs truncate text-xs text-zinc-500">
          {row.original.raw_payload}
        </code>
      )
    }
  ];
}

function Field({
  label,
  registration
}: {
  label: string;
  registration: UseFormRegisterReturn;
}) {
  return (
    <label className="text-sm font-medium text-zinc-800">
      {label}
      <input
        className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
        {...registration}
      />
    </label>
  );
}

function DataTable<TData>({
  table,
  empty
}: {
  table: ReturnType<typeof useReactTable<TData>>;
  empty: string;
}) {
  return (
    <div className="overflow-hidden rounded-md border border-zinc-200 bg-white">
      <table className="w-full border-collapse text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase tracking-wide text-zinc-500">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id} className="px-3 py-2 font-medium">
                  {flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id} className="border-t border-zinc-100">
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id} className="px-3 py-3 align-middle text-zinc-700">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
          {table.getRowModel().rows.length === 0 ? (
            <tr>
              <td className="px-3 py-8 text-center text-sm text-zinc-500" colSpan={20}>
                {empty}
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
  );
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
      {message}
    </div>
  );
}
