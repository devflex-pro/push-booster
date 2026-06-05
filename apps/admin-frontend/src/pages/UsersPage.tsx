import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable
} from "@tanstack/react-table";
import { Check, Loader2 } from "lucide-react";
import { useMemo } from "react";

import { AuthUser, api } from "../api/client";
import { cn } from "../lib/utils";
import { useAuthStore } from "../store/auth";

export function UsersPage() {
  const token = useAuthStore((state) => state.token);
  const queryClient = useQueryClient();

  const users = useQuery({
    queryKey: ["users", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.users(token);
    },
    enabled: Boolean(token)
  });

  const approve = useMutation({
    mutationFn: (id: string) => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.approveUser(token, id);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["users", token] });
    }
  });

  const columns = useMemo<ColumnDef<AuthUser>[]>(
    () => [
      {
        header: "Email",
        accessorKey: "email"
      },
      {
        header: "Role",
        accessorKey: "role"
      },
      {
        header: "Status",
        cell: ({ row }) => <StatusBadge user={row.original} />
      },
      {
        header: "Email verified",
        cell: ({ row }) => (row.original.email_verified ? "Yes" : "No")
      },
      {
        header: "",
        id: "actions",
        cell: ({ row }) =>
          row.original.approved ? null : (
            <button
              className="inline-flex h-8 items-center gap-2 rounded-md bg-zinc-950 px-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60"
              disabled={approve.isPending}
              type="button"
              onClick={() => {
                approve.mutate(row.original.id);
              }}
            >
              {approve.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Check className="h-4 w-4" />
              )}
              Approve
            </button>
          )
      }
    ],
    [approve]
  );

  const table = useReactTable({
    data: users.data?.items ?? [],
    columns,
    getCoreRowModel: getCoreRowModel()
  });

  return (
    <section className="space-y-4">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-zinc-950">Users</h1>
          <p className="mt-1 text-sm text-zinc-600">Email OTP accounts and approval state.</p>
        </div>
        <div className="text-sm text-zinc-500">{users.data?.total ?? 0} total</div>
      </div>

      {users.isLoading ? (
        <div className="rounded-md border border-zinc-200 bg-white p-6 text-sm text-zinc-600">
          Loading users
        </div>
      ) : null}

      {users.isError ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          {users.error.message}
        </div>
      ) : null}

      {approve.isError ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          {approve.error.message}
        </div>
      ) : null}

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
        {table.getRowModel().rows.length === 0 && !users.isLoading ? (
          <div className="p-6 text-sm text-zinc-600">No users yet</div>
        ) : null}
      </div>
    </section>
  );
}

function StatusBadge({ user }: { user: AuthUser }) {
  return (
    <span
      className={cn(
        "inline-flex rounded-md px-2 py-1 text-xs font-medium",
        user.approved
          ? "bg-emerald-50 text-emerald-800"
          : "bg-amber-50 text-amber-800"
      )}
    >
      {user.status}
    </span>
  );
}
