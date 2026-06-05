import { useQuery } from "@tanstack/react-query";
import {
  ColumnDef,
  Table,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { ArrowLeft } from "lucide-react";
import { useMemo, useState } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";

import { PerformanceRow, api } from "../api/client";
import { useAuthStore } from "../store/auth";

const emptyPerformanceRows: PerformanceRow[] = [];

type Drilldown =
  | { groupBy: "date" }
  | { groupBy: "publisher" | "campaign"; date: string }
  | { groupBy: "source"; date: string; publisherId: string; publisherName: string }
  | { groupBy: "creative"; date: string; campaignId: string; campaignName: string };

export function ReportsPage() {
  const token = useAuthStore((state) => state.token);
  const [drilldown, setDrilldown] = useState<Drilldown>({ groupBy: "date" });
  const [drilldownHistory, setDrilldownHistory] = useState<Drilldown[]>([]);
  const [dateFrom, setDateFrom] = useState(defaultDateFrom());
  const [dateTo, setDateTo] = useState(defaultDateTo());
  const groupBy = drilldown.groupBy;
  const reportDateFrom = drilldown.groupBy === "date" ? dateFrom : drilldown.date;
  const reportDateTo = drilldown.groupBy === "date" ? dateTo : drilldown.date;

  const performance = useQuery({
    queryKey: ["performance-report", token, drilldown, reportDateFrom, reportDateTo],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.performanceReport(token, drilldown.groupBy, {
        date_from: reportDateFrom,
        date_to: reportDateTo,
        publisher_id: drilldown.groupBy === "source" ? drilldown.publisherId : undefined,
        campaign_id: drilldown.groupBy === "creative" ? drilldown.campaignId : undefined,
        sort_by: drilldown.groupBy === "date" ? "date" : undefined
      });
    },
    enabled: Boolean(token)
  });

  const performanceColumns = useMemo<ColumnDef<PerformanceRow>[]>(
    () => [
      {
        header: dimensionHeader(drilldown),
        cell: ({ row }) => <DimensionCell row={row.original} />
      },
      { header: "Sent", accessorKey: "sent" },
      { header: "Impr.", accessorKey: "shown" },
      { header: "Clicks", accessorKey: "clicks" },
      { header: "CTR", cell: ({ row }) => percent(row.original.ctr) },
      { header: "Conv", accessorKey: "conversions" },
      { header: "CR", cell: ({ row }) => percent(row.original.cr) },
      { header: "Spend", cell: ({ row }) => money(row.original.spend) },
      { header: "Revenue", cell: ({ row }) => money(row.original.revenue) },
      {
        header: "Profit",
        cell: ({ row }) => (
          <span className={row.original.profit >= 0 ? "text-emerald-700" : "text-red-700"}>
            {money(row.original.profit)}
          </span>
        )
      },
      { header: "ROI", cell: ({ row }) => percent(row.original.roi) },
      {
        header: "",
        id: "actions",
        cell: ({ row }) => (
          <ReportRowActions
            drilldown={drilldown}
            row={row.original}
            onDrilldown={(nextDrilldown) => {
              setDrilldownHistory((history) => [...history, drilldown]);
              setDrilldown(nextDrilldown);
            }}
          />
        )
      }
    ],
    [drilldown]
  );

  const performanceRows = useMemo(() => {
    const rows = performance.data?.items ?? emptyPerformanceRows;
    if (groupBy !== "date") {
      return rows;
    }
    return [...rows].sort((left, right) => left.key.localeCompare(right.key));
  }, [groupBy, performance.data?.items]);
  const chartRows = useMemo(
    () =>
      performanceRows.map((row) => ({
        ...row,
        dimensionLabel: row.name || row.key
      })),
    [performanceRows]
  );
  const summary = useMemo(() => performanceSummary(performanceRows), [performanceRows]);

  const performanceTable = useReactTable({
    data: performanceRows,
    columns: performanceColumns,
    getCoreRowModel: getCoreRowModel()
  });

  return (
    <section className="space-y-4">
      <div className="space-y-3">
        <div>
          <h1 className="text-xl font-semibold text-zinc-950">Reports</h1>
          <p className="mt-1 text-sm text-zinc-600">
            Daily performance with drill-downs into publishers, sources, campaigns and creatives.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {drilldown.groupBy === "date" ? (
            <>
              <DateInput label="From" value={dateFrom} onChange={setDateFrom} />
              <DateInput label="To" value={dateTo} onChange={setDateTo} />
            </>
          ) : (
            <button
              className="inline-flex h-9 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 hover:bg-zinc-100"
              type="button"
              onClick={() => {
                const previous = drilldownHistory.at(-1) ?? { groupBy: "date" };
                setDrilldown(previous);
                setDrilldownHistory((history) => history.slice(0, -1));
              }}
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </button>
          )}
          {drilldown.groupBy === "date" ? null : (
            <div className="text-base font-medium text-zinc-800">
              {reportScopeLabel(drilldown)}
            </div>
          )}
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-3 lg:grid-cols-6">
        <MetricCard label="Spend" value={money(summary.spend)} />
        <MetricCard label="Revenue" value={money(summary.revenue)} />
        <MetricCard label="Profit" value={money(summary.profit)} />
        <MetricCard label="ROI" value={percent(summary.roi)} />
        <MetricCard label="Clicks" value={String(summary.clicks)} />
        <MetricCard label="Conversions" value={String(summary.conversions)} />
      </div>

      <div className="rounded-md border border-zinc-200 bg-white p-4">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-zinc-950">Arbitrage Cockpit</h2>
          <div className="text-xs text-zinc-500">
            {performance.data?.total ?? 0} rows
          </div>
        </div>
        <div className="h-72">
          <ResponsiveContainer height="100%" width="100%">
            <BarChart data={chartRows} margin={{ bottom: 4, left: 4, right: 8, top: 8 }}>
              <CartesianGrid stroke="#e4e4e7" vertical={false} />
              <XAxis
                dataKey="dimensionLabel"
                angle={-55}
                height={64}
                interval={0}
                minTickGap={6}
                tick={{ fontSize: 11, textAnchor: "end" }}
                tickFormatter={(value) => chartTickLabel(String(value))}
                tickLine={false}
                tickMargin={8}
              />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip
                formatter={(value) => money(Number(value))}
                labelFormatter={(label) => String(label)}
              />
              <Bar dataKey="spend" fill="#b08d57" radius={[3, 3, 0, 0]} />
              <Bar dataKey="revenue" fill="#2f6b4f" radius={[3, 3, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {performance.isError ? <ErrorMessage message={performance.error.message} /> : null}

      <DataTable title="Performance" table={performanceTable} />
    </section>
  );
}

function DateInput(props: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label className="flex items-center gap-2 text-xs text-zinc-500">
      {props.label}
      <input
        className="h-9 rounded-md border border-zinc-300 bg-white px-2 text-sm text-zinc-700 outline-none transition focus:border-zinc-950"
        type="date"
        value={props.value}
        onChange={(event) => {
          props.onChange(event.target.value);
        }}
      />
    </label>
  );
}

function DimensionCell(props: { row: PerformanceRow }) {
  const label = props.row.name || props.row.key;

  return (
    <div title={props.row.key}>
      <div className="max-w-sm truncate text-sm font-medium text-zinc-800">
        {label}
      </div>
    </div>
  );
}

function ReportRowActions(props: {
  drilldown: Drilldown;
  row: PerformanceRow;
  onDrilldown: (drilldown: Drilldown) => void;
}) {
  const label = props.row.name || props.row.key;
  if (props.drilldown.groupBy === "date") {
    return (
      <div className="flex justify-end gap-2">
        <DrilldownButton
          label="Publishers"
          onClick={() => {
            props.onDrilldown({ groupBy: "publisher", date: props.row.key });
          }}
        />
        <DrilldownButton
          label="Campaigns"
          onClick={() => {
            props.onDrilldown({ groupBy: "campaign", date: props.row.key });
          }}
        />
      </div>
    );
  }
  if (props.drilldown.groupBy === "publisher") {
    const date = props.drilldown.date;
    return (
      <div className="flex justify-end">
        <DrilldownButton
          label="Sources"
          onClick={() => {
            props.onDrilldown({
              groupBy: "source",
              date,
              publisherId: props.row.key,
              publisherName: label
            });
          }}
        />
      </div>
    );
  }
  if (props.drilldown.groupBy === "campaign") {
    const date = props.drilldown.date;
    return (
      <div className="flex justify-end">
        <DrilldownButton
          label="Creatives"
          onClick={() => {
            props.onDrilldown({
              groupBy: "creative",
              date,
              campaignId: props.row.key,
              campaignName: label
            });
          }}
        />
      </div>
    );
  }
  return null;
}

function DrilldownButton(props: { label: string; onClick: () => void }) {
  return (
    <button
      className="h-8 rounded-md border border-zinc-300 bg-white px-3 text-xs font-medium text-zinc-700 hover:bg-zinc-100"
      type="button"
      onClick={props.onClick}
    >
      {props.label}
    </button>
  );
}

function dimensionHeader(drilldown: Drilldown) {
  switch (drilldown.groupBy) {
    case "publisher":
      return "Publisher";
    case "source":
      return "Source";
    case "campaign":
      return "Campaign";
    case "creative":
      return "Creative";
    default:
      return "Date";
  }
}

function reportScopeLabel(drilldown: Drilldown) {
  switch (drilldown.groupBy) {
    case "publisher":
      return `${drilldown.date} / publishers`;
    case "source":
      return `${drilldown.date} / ${drilldown.publisherName} / sources`;
    case "campaign":
      return `${drilldown.date} / campaigns`;
    case "creative":
      return `${drilldown.date} / ${drilldown.campaignName} / creatives`;
    default:
      return "";
  }
}

function performanceSummary(rows: PerformanceRow[]) {
  const summary = rows.reduce(
    (total, row) => ({
      spend: total.spend + row.spend,
      revenue: total.revenue + row.revenue,
      conversions: total.conversions + row.conversions,
      clicks: total.clicks + row.clicks
    }),
    {
      spend: 0,
      revenue: 0,
      conversions: 0,
      clicks: 0
    }
  );
  const profit = summary.revenue - summary.spend;
  return {
    ...summary,
    profit,
    roi: summary.spend === 0 ? 0 : (profit / summary.spend) * 100
  };
}

function MetricCard(props: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-zinc-200 bg-white p-4">
      <div className="text-xs uppercase text-zinc-500">{props.label}</div>
      <div className="mt-2 text-xl font-semibold text-zinc-950">{props.value}</div>
    </div>
  );
}

function DataTable<TRow>(props: {
  title: string;
  table: Table<TRow>;
}) {
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
              <tr key={row.id} className="odd:bg-white even:bg-zinc-50/70 hover:bg-zinc-100/70">
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
    </div>
  );
}

function ErrorMessage(props: { message: string }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
      {props.message}
    </div>
  );
}

function money(value: number) {
  return new Intl.NumberFormat("en-US", {
    currency: "USD",
    maximumFractionDigits: 2,
    style: "currency"
  }).format(value);
}

function percent(value: number) {
  return `${value.toFixed(2)}%`;
}

function chartTickLabel(value: string) {
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return value.slice(5);
  }
  return value.length > 18 ? `${value.slice(0, 17)}...` : value;
}

function defaultDateTo() {
  return new Date().toISOString().slice(0, 10);
}

function defaultDateFrom() {
  const date = new Date();
  date.setUTCDate(1);
  return date.toISOString().slice(0, 10);
}
