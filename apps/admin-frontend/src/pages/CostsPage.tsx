import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ColumnDef,
  Table,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { Loader2, Plus, Upload, X } from "lucide-react";
import { useMemo, useState } from "react";
import { type UseFormRegisterReturn, useForm } from "react-hook-form";
import { z } from "zod";

import { CostEntry, api } from "../api/client";
import { useAuthStore } from "../store/auth";

const costSchema = z.object({
  date: z.string().trim().min(10),
  publisher_id: z.string().trim(),
  source_id: z.string().trim(),
  amount: z.number().positive(),
  currency: z.string().trim().min(3),
  note: z.string().trim()
});

type CostForm = z.infer<typeof costSchema>;

const emptyCostEntries: CostEntry[] = [];

export function CostsPage() {
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [page, setPage] = useState(0);
  const [pageSize, setPageSize] = useState(25);
  const form = useForm<CostForm>({
    resolver: zodResolver(costSchema),
    defaultValues: defaultCostForm()
  });

  const costs = useQuery({
    queryKey: ["costs", token, page, pageSize],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.costs(token, pageSize, page * pageSize);
    },
    enabled: Boolean(token)
  });

  const createCost = useMutation({
    mutationFn: (values: CostForm) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.createCost(token, values);
    },
    onSuccess: async () => {
      form.reset(defaultCostForm());
      setCreateOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["dashboard-report", token] });
      await queryClient.invalidateQueries({ queryKey: ["performance-report", token] });
      await queryClient.invalidateQueries({ queryKey: ["costs", token] });
      setPage(0);
    }
  });

  const importCosts = useMutation({
    mutationFn: (file: File) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.importCosts(token, file);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["dashboard-report", token] });
      await queryClient.invalidateQueries({ queryKey: ["performance-report", token] });
      await queryClient.invalidateQueries({ queryKey: ["costs", token] });
      setPage(0);
    }
  });

  const costColumns = useMemo<ColumnDef<CostEntry>[]>(
    () => [
      { header: "Date", accessorKey: "date" },
      { header: "Amount", cell: ({ row }) => money(row.original.amount) },
      { header: "Currency", accessorKey: "currency" },
      {
        header: "Scope",
        cell: ({ row }) => (
          <span
            className="block max-w-md truncate text-xs text-zinc-600"
            title={costScopeIDs(row.original)}
          >
            {costScope(row.original)}
          </span>
        )
      },
      { header: "Note", accessorKey: "note" }
    ],
    []
  );

  const costTable = useReactTable({
    data: costs.data?.items ?? emptyCostEntries,
    columns: costColumns,
    getCoreRowModel: getCoreRowModel()
  });

  function onCreate(values: CostForm) {
    createCost.mutate(values);
  }

  const total = costs.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const canPrevious = page > 0;
  const canNext = page + 1 < pageCount;

  return (
    <section className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-zinc-950">Costs</h1>
          <p className="mt-1 text-sm text-zinc-600">
            Manual spend entries and CSV cost imports.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <label className="inline-flex h-9 cursor-pointer items-center justify-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50">
            {importCosts.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Upload className="h-4 w-4" />
            )}
            Import CSV
            <input
              accept=".csv,text/csv"
              className="sr-only"
              disabled={importCosts.isPending}
              type="file"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) {
                  importCosts.mutate(file);
                }
                event.target.value = "";
              }}
            />
          </label>
          <button
            className="inline-flex h-9 items-center gap-2 rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800"
            type="button"
            onClick={() => {
              setCreateOpen(true);
            }}
          >
            <Plus className="h-4 w-4" />
            Add cost
          </button>
        </div>
      </div>

      <div className="rounded-md border border-zinc-200 bg-white p-4">
        <h2 className="text-sm font-semibold text-zinc-950">CSV Cost Import</h2>
        <p className="mt-1 text-xs text-zinc-500">
          Columns: date, publisher_id, source_id, amount, currency, note.
        </p>
        {importCosts.data ? (
          <div className="mt-3 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700">
            Imported {importCosts.data.inserted} cost rows.
          </div>
        ) : null}
      </div>

      {costs.isError ? <ErrorMessage message={costs.error.message} /> : null}
      {createCost.isError && !createOpen ? <ErrorMessage message={createCost.error.message} /> : null}
      {importCosts.isError ? <ErrorMessage message={importCosts.error.message} /> : null}

      <DataTable
        page={page}
        pageCount={pageCount}
        pageSize={pageSize}
        table={costTable}
        title="Cost Entries"
        total={total}
        onNext={() => {
          setPage((current) => (canNext ? current + 1 : current));
        }}
        onPageSizeChange={(value) => {
          setPageSize(value);
          setPage(0);
        }}
        onPageChange={setPage}
        onPrevious={() => {
          setPage((current) => (canPrevious ? current - 1 : current));
        }}
      />

      {createOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4">
          <div className="max-h-[90vh] w-full max-w-xl overflow-auto rounded-md border border-zinc-200 bg-white shadow-xl">
            <div className="flex items-center justify-between border-b border-zinc-200 px-4 py-3">
              <h2 className="text-sm font-semibold text-zinc-950">Add cost</h2>
              <button
                className="inline-flex h-8 w-8 items-center justify-center rounded-md text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-950"
                type="button"
                onClick={() => {
                  form.reset(defaultCostForm());
                  setCreateOpen(false);
                }}
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <form
              className="space-y-3 p-4"
              onSubmit={(event) => {
                void form.handleSubmit(onCreate)(event);
              }}
            >
              <Field label="Date" registration={form.register("date")} type="date" />
              <Field
                label="Amount"
                registration={form.register("amount", { valueAsNumber: true })}
                type="number"
              />
              <Field label="Currency" registration={form.register("currency")} />
              <Field label="Publisher ID" registration={form.register("publisher_id")} />
              <Field label="Source ID" registration={form.register("source_id")} />
              <Field label="Note" registration={form.register("note")} />
              {form.formState.errors.amount ? (
                <ErrorMessage message={form.formState.errors.amount.message ?? "Invalid amount"} />
              ) : null}
              {createCost.isError ? <ErrorMessage message={createCost.error.message} /> : null}
              <div className="flex justify-end gap-2">
                <button
                  className="inline-flex h-10 items-center justify-center rounded-md border border-zinc-300 bg-white px-4 text-sm text-zinc-700 transition hover:bg-zinc-50"
                  type="button"
                  onClick={() => {
                    form.reset(defaultCostForm());
                    setCreateOpen(false);
                  }}
                >
                  Cancel
                </button>
                <button
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:opacity-60"
                  disabled={createCost.isPending}
                  type="submit"
                >
                  {createCost.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                  Add cost
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </section>
  );
}

function DataTable<TRow>(props: {
  page: number;
  pageCount: number;
  pageSize: number;
  title: string;
  table: Table<TRow>;
  total: number;
  onNext: () => void;
  onPageChange: (page: number) => void;
  onPageSizeChange: (value: number) => void;
  onPrevious: () => void;
}) {
  const pages = visiblePages(props.page, props.pageCount);

  return (
    <div className="overflow-hidden rounded-md border border-zinc-200 bg-white">
      <div className="border-b border-zinc-200 px-4 py-3 text-sm font-semibold text-zinc-950">
        {props.title}
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-zinc-200 text-sm">
          <thead className="bg-zinc-50">
            {props.table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className="px-4 py-3 text-left text-xs font-medium uppercase text-zinc-500"
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {props.table.getRowModel().rows.map((row) => (
              <tr key={row.id}>
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="whitespace-nowrap px-4 py-3 text-zinc-700">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-zinc-200 px-4 py-3">
        <div className="text-xs text-zinc-500">
          Page {props.page + 1} of {props.pageCount} · {props.total} total
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <select
            className="h-8 rounded-md border border-zinc-300 bg-white px-2 text-sm text-zinc-700 outline-none transition focus:border-zinc-950"
            value={props.pageSize}
            onChange={(event) => {
              props.onPageSizeChange(Number(event.target.value));
            }}
          >
            {[10, 25, 50, 100].map((value) => (
              <option key={value} value={value}>
                {value} / page
              </option>
            ))}
          </select>
          <button
            className="inline-flex h-8 items-center rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-50"
            disabled={props.page === 0}
            type="button"
            onClick={() => {
              props.onPageChange(0);
            }}
          >
            First
          </button>
          <button
            className="inline-flex h-8 items-center rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-50"
            disabled={props.page === 0}
            type="button"
            onClick={props.onPrevious}
          >
            Previous
          </button>
          <div className="flex items-center gap-1">
            {pages.map((page) => (
              <button
                className={
                  page === props.page
                    ? "inline-flex h-8 min-w-8 items-center justify-center rounded-md bg-zinc-950 px-2 text-sm text-white"
                    : "inline-flex h-8 min-w-8 items-center justify-center rounded-md border border-zinc-300 bg-white px-2 text-sm text-zinc-700 transition hover:bg-zinc-50"
                }
                key={page}
                type="button"
                onClick={() => {
                  props.onPageChange(page);
                }}
              >
                {page + 1}
              </button>
            ))}
          </div>
          <button
            className="inline-flex h-8 items-center rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-50"
            disabled={props.page + 1 >= props.pageCount}
            type="button"
            onClick={props.onNext}
          >
            Next
          </button>
          <button
            className="inline-flex h-8 items-center rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50 disabled:opacity-50"
            disabled={props.page + 1 >= props.pageCount}
            type="button"
            onClick={() => {
              props.onPageChange(props.pageCount - 1);
            }}
          >
            Last
          </button>
        </div>
      </div>
    </div>
  );
}

function visiblePages(current: number, total: number): number[] {
  const start = Math.max(0, Math.min(current - 2, total - 5));
  const end = Math.min(total, start + 5);
  return Array.from({ length: end - start }, (_, index) => start + index);
}

function Field(props: {
  label: string;
  registration: UseFormRegisterReturn;
  type?: string;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-zinc-600">{props.label}</span>
      <input
        className="h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm outline-none transition focus:border-zinc-950"
        step={props.type === "number" ? "0.01" : undefined}
        type={props.type ?? "text"}
        {...props.registration}
      />
    </label>
  );
}

function ErrorMessage(props: { message: string }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
      {props.message}
    </div>
  );
}

function defaultCostForm(): CostForm {
  return {
    date: new Date().toISOString().slice(0, 10),
    publisher_id: "",
    source_id: "",
    amount: 0,
    currency: "USD",
    note: ""
  };
}

function money(value: number) {
  return new Intl.NumberFormat("en-US", {
    currency: "USD",
    maximumFractionDigits: 2,
    style: "currency"
  }).format(value);
}

function costScope(entry: CostEntry) {
  const scope: Array<[string, string | undefined]> = [
    ["Publisher", entry.publisher_name || entry.publisher_id],
    ["Source", entry.source_name || entry.source_id],
    ["Campaign", entry.campaign_name || entry.campaign_id],
    ["Creative", entry.creative_name || entry.creative_id]
  ];

  return scope
    .filter((item): item is [string, string] => Boolean(item[1]))
    .map(([label, value]) => `${label}: ${value}`)
    .join(" / ");
}

function costScopeIDs(entry: CostEntry) {
  const scope: Array<[string, string | undefined]> = [
    ["pub", entry.publisher_id],
    ["src", entry.source_id],
    ["cmp", entry.campaign_id],
    ["crt", entry.creative_id]
  ];

  return scope
    .filter((item): item is [string, string] => Boolean(item[1]))
    .map(([label, value]) => `${label}:${value}`)
    .join(" ");
}
