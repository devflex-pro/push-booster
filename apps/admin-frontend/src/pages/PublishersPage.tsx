import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { Globe2, Loader2, Plus, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { z } from "zod";

import { Publisher, api } from "../api/client";
import { useAuthStore } from "../store/auth";

const publisherSchema = z.object({
  name: z.string().trim().min(2)
});

type PublisherForm = z.infer<typeof publisherSchema>;

export function PublishersPage() {
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const form = useForm<PublisherForm>({
    resolver: zodResolver(publisherSchema),
    defaultValues: { name: "" }
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

  const createPublisher = useMutation({
    mutationFn: (values: PublisherForm) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.createPublisher(token, values.name);
    },
    onSuccess: async () => {
      form.reset();
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["publishers", token] });
    }
  });

  const columns = useMemo<ColumnDef<Publisher>[]>(
    () => [
      { header: "Name", accessorKey: "name" },
      { header: "Status", accessorKey: "status" },
      {
        header: "Created",
        cell: ({ row }) => new Date(row.original.created_at).toLocaleString()
      },
      {
        header: "Actions",
        id: "actions",
        cell: ({ row }) => (
          <Link
            className="inline-flex h-8 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
            to={`/publishers/${row.original.id}/sources`}
          >
            <Globe2 className="h-4 w-4" />
            Sources
          </Link>
        )
      }
    ],
    []
  );

  const table = useReactTable({
    data: publishers.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  function onSubmit(values: PublisherForm) {
    createPublisher.mutate(values);
  }

  return (
    <section className="space-y-4">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-zinc-950">Publishers</h1>
          <p className="mt-1 text-sm text-zinc-600">Inventory containers for sources.</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="text-sm text-zinc-500">{publishers.data?.total ?? 0} total</div>
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

      {publishers.isError ? <ErrorMessage message={publishers.error.message} /> : null}

      <DataTable table={table} isLoading={publishers.isLoading} emptyText="No publishers yet" />

      {createOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="w-full max-w-md rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Create publisher</h2>
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
                  placeholder="Media Buyer A"
                  {...form.register("name")}
                />
              </label>
              {form.formState.errors.name ? (
                <div className="text-sm text-red-700">{form.formState.errors.name.message}</div>
              ) : null}
              {createPublisher.isError ? <ErrorMessage message={createPublisher.error.message} /> : null}
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
                  disabled={createPublisher.isPending}
                  type="submit"
                >
                  {createPublisher.isPending ? (
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

function DataTable({
  table,
  isLoading,
  emptyText
}: {
  table: ReturnType<typeof useReactTable<Publisher>>;
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
        <div className="p-6 text-sm text-zinc-600">{isLoading ? "Loading publishers" : emptyText}</div>
      ) : null}
    </div>
  );
}
