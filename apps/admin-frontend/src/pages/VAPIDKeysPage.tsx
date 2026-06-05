import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { Copy, KeyRound, Loader2, Link2, Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { VAPIDKey, VAPIDKeyStatus, api } from "../api/client";
import { useAuthStore } from "../store/auth";

const keySchema = z.object({
  name: z.string().trim().min(2)
});

const attachSchema = z.object({
  source_id: z.string().trim().min(1),
  vapid_key_id: z.string().trim().min(1)
});

type KeyForm = z.infer<typeof keySchema>;
type AttachForm = z.infer<typeof attachSchema>;

const statuses: VAPIDKeyStatus[] = ["active", "deprecated", "revoked"];

export function VAPIDKeysPage() {
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();
  const [createdKey, setCreatedKey] = useState<VAPIDKey | null>(null);
  const keyForm = useForm<KeyForm>({
    resolver: zodResolver(keySchema),
    defaultValues: { name: "" }
  });
  const attachForm = useForm<AttachForm>({
    resolver: zodResolver(attachSchema),
    defaultValues: { source_id: "", vapid_key_id: "" }
  });

  const keys = useQuery({
    queryKey: ["vapid-keys", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.vapidKeys(token);
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

  const createKey = useMutation({
    mutationFn: (values: KeyForm) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.createVAPIDKey(token, values.name);
    },
    onSuccess: async (key) => {
      setCreatedKey(key);
      keyForm.reset();
      await queryClient.invalidateQueries({ queryKey: ["vapid-keys", token] });
    }
  });

  const updateStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: VAPIDKeyStatus }) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.updateVAPIDKeyStatus(token, id, status);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["vapid-keys", token] });
    }
  });

  const attachKey = useMutation({
    mutationFn: (values: AttachForm) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.attachVAPIDKeyToSource(token, values.source_id, values.vapid_key_id);
    },
    onSuccess: async () => {
      attachForm.reset();
      await queryClient.invalidateQueries({ queryKey: ["sources", token] });
    }
  });

  const columns = useMemo<ColumnDef<VAPIDKey>[]>(
    () => [
      { header: "Name", accessorKey: "name" },
      {
        header: "Public key",
        cell: ({ row }) => (
          <code className="block max-w-md truncate text-xs text-zinc-600">
            {row.original.public_key}
          </code>
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
                status: event.target.value as VAPIDKeyStatus
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
        header: "Created",
        cell: ({ row }) => new Date(row.original.created_at).toLocaleString()
      }
    ],
    [updateStatus]
  );

  const table = useReactTable({
    data: keys.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  function onCreate(values: KeyForm) {
    createKey.mutate(values);
  }

  function onAttach(values: AttachForm) {
    attachKey.mutate(values);
  }

  return (
    <section className="space-y-4">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-zinc-950">VAPID keys</h1>
          <p className="mt-1 text-sm text-zinc-600">Web Push keys bound to subscription sources.</p>
        </div>
        <div className="text-sm text-zinc-500">{keys.data?.total ?? 0} total</div>
      </div>

      <form
        className="flex flex-col gap-3 rounded-md border border-zinc-200 bg-white p-4 sm:flex-row"
        onSubmit={(event) => {
          void keyForm.handleSubmit(onCreate)(event);
        }}
      >
        <label className="flex-1 text-sm font-medium text-zinc-800">
          Key name
          <input
            className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
            placeholder="Main source key"
            {...keyForm.register("name")}
          />
        </label>
        <button
          className="inline-flex h-10 items-center justify-center gap-2 self-end rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
          disabled={createKey.isPending}
          type="submit"
        >
          {createKey.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Plus className="h-4 w-4" />
          )}
          Generate
        </button>
      </form>

      {keyForm.formState.errors.name ? (
        <ErrorMessage message={keyForm.formState.errors.name.message ?? "Invalid key name"} />
      ) : null}
      {keys.isError ? <ErrorMessage message={keys.error.message} /> : null}
      {createKey.isError ? <ErrorMessage message={createKey.error.message} /> : null}
      {updateStatus.isError ? <ErrorMessage message={updateStatus.error.message} /> : null}
      {attachKey.isError ? <ErrorMessage message={attachKey.error.message} /> : null}

      {createdKey?.private_key ? (
        <section className="space-y-3 rounded-md border border-amber-200 bg-amber-50 p-4">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2 text-sm font-semibold text-amber-950">
              <KeyRound className="h-4 w-4" />
              Private key is shown once
            </div>
            <button
              className="inline-flex h-8 items-center gap-2 rounded-md border border-amber-300 bg-white px-3 text-sm text-amber-950 transition hover:bg-amber-100"
              type="button"
              onClick={() => {
                void navigator.clipboard.writeText(createdKey.private_key ?? "");
              }}
            >
              <Copy className="h-4 w-4" />
              Copy
            </button>
          </div>
          <code className="block break-all rounded-md bg-white p-3 text-xs text-amber-950">
            {createdKey.private_key}
          </code>
        </section>
      ) : null}

      <DataTable table={table} isLoading={keys.isLoading} />

      <form
        className="grid gap-3 rounded-md border border-zinc-200 bg-white p-4 lg:grid-cols-[1fr_1fr_auto]"
        onSubmit={(event) => {
          void attachForm.handleSubmit(onAttach)(event);
        }}
      >
        <label className="text-sm font-medium text-zinc-800">
          Source
          <select
            className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
            {...attachForm.register("source_id")}
          >
            <option value="">Select source</option>
            {(sources.data?.items ?? []).map((source) => (
              <option key={source.id} value={source.id}>
                {source.name} / {source.domain}
              </option>
            ))}
          </select>
        </label>
        <label className="text-sm font-medium text-zinc-800">
          Active VAPID key
          <select
            className="mt-2 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
            {...attachForm.register("vapid_key_id")}
          >
            <option value="">Select key</option>
            {(keys.data?.items ?? [])
              .filter((key) => key.status === "active")
              .map((key) => (
                <option key={key.id} value={key.id}>
                  {key.name}
                </option>
              ))}
          </select>
        </label>
        <button
          className="inline-flex h-10 items-center justify-center gap-2 self-end rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
          disabled={attachKey.isPending}
          type="submit"
        >
          {attachKey.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Link2 className="h-4 w-4" />
          )}
          Bind
        </button>
      </form>
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

function DataTable({
  table,
  isLoading
}: {
  table: ReturnType<typeof useReactTable<VAPIDKey>>;
  isLoading: boolean;
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
        <div className="p-6 text-sm text-zinc-600">
          {isLoading ? "Loading VAPID keys" : "No VAPID keys yet"}
        </div>
      ) : null}
    </div>
  );
}
