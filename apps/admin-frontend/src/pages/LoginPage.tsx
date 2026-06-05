import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { ArrowRight, Loader2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Navigate, useNavigate } from "react-router";
import { z } from "zod";

import { ApiRequestError, api } from "../api/client";
import { useAuthStore } from "../store/auth";

const loginSchema = z.object({
  email: z.email(),
  otp: z.string().trim().optional()
});

type LoginForm = z.infer<typeof loginSchema>;

export function LoginPage() {
  const token = useAuthStore((state) => state.token);
  const setToken = useAuthStore((state) => state.setToken);
  const navigate = useNavigate();
  const [requestedEmail, setRequestedEmail] = useState("");
  const [devOTP, setDevOTP] = useState("");
  const [pendingApproval, setPendingApproval] = useState(false);

  const form = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "admin@example.com",
      otp: ""
    }
  });

  const requestOTP = useMutation({
    mutationFn: (email: string) => api.requestOTP(email),
    onSuccess: (challenge) => {
      setRequestedEmail(challenge.email);
      setDevOTP(challenge.otp ?? "");
      setPendingApproval(false);
      form.setValue("email", challenge.email);
      if (challenge.otp) {
        form.setValue("otp", challenge.otp);
      }
    }
  });

  const verifyOTP = useMutation({
    mutationFn: (values: { email: string; otp: string }) =>
      api.verifyOTP(values.email, values.otp),
    onSuccess: (session) => {
      setToken(session.token);
      void navigate("/publishers", { replace: true });
    },
    onError: (error: Error) => {
      if (error instanceof ApiRequestError && error.response?.status === "pending_approval") {
        setPendingApproval(true);
      }
    }
  });

  if (token) {
    return <Navigate to="/publishers" replace />;
  }

  const isVerifying = Boolean(requestedEmail);
  const isBusy = requestOTP.isPending || verifyOTP.isPending;
  const error = requestOTP.error ?? verifyOTP.error;

  function onSubmit(values: LoginForm) {
    if (isVerifying) {
      verifyOTP.mutate({
        email: values.email,
        otp: values.otp ?? ""
      });
      return;
    }
    requestOTP.mutate(values.email);
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-100 px-4 py-10">
      <section className="w-full max-w-md rounded-md border border-zinc-200 bg-white p-6 shadow-sm">
        <div className="mb-6">
          <div className="text-xs font-semibold uppercase tracking-wide text-zinc-500">
            Push Booster Admin
          </div>
          <h1 className="mt-2 text-2xl font-semibold text-zinc-950">Email OTP login</h1>
        </div>

        <form
          className="space-y-4"
          onSubmit={(event) => {
            void form.handleSubmit(onSubmit)(event);
          }}
        >
          <label className="block text-sm font-medium text-zinc-800">
            Email
            <input
              className="mt-2 w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm outline-none ring-0 transition focus:border-zinc-950"
              disabled={isBusy || isVerifying}
              {...form.register("email")}
            />
          </label>
          {form.formState.errors.email ? (
            <div className="text-sm text-red-700">{form.formState.errors.email.message}</div>
          ) : null}

          {isVerifying ? (
            <label className="block text-sm font-medium text-zinc-800">
              OTP
              <input
                className="mt-2 w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm outline-none ring-0 transition focus:border-zinc-950"
                disabled={isBusy}
                inputMode="numeric"
                {...form.register("otp")}
              />
            </label>
          ) : null}

          {devOTP ? (
            <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900">
              Local OTP: <span className="font-mono font-semibold">{devOTP}</span>
            </div>
          ) : null}

          {pendingApproval ? (
            <div className="rounded-md border border-sky-200 bg-sky-50 px-3 py-2 text-sm text-sky-900">
              User is pending admin approval.
            </div>
          ) : null}

          {error ? (
            <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
              {error.message}
            </div>
          ) : null}

          <button
            className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-zinc-950 px-4 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60"
            disabled={isBusy}
            type="submit"
          >
            {isBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : <ArrowRight className="h-4 w-4" />}
            {isVerifying ? "Verify OTP" : "Send OTP"}
          </button>

          {isVerifying ? (
            <button
              className="h-9 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
              disabled={isBusy}
              type="button"
              onClick={() => {
                setRequestedEmail("");
                setDevOTP("");
                setPendingApproval(false);
                form.setValue("otp", "");
              }}
            >
              Change email
            </button>
          ) : null}
        </form>
      </section>
    </main>
  );
}
