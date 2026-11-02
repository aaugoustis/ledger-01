"use client";

import { useMutation, useQuery } from "@connectrpc/connect-query";
import { UserService } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { Field } from "@ledger-01/ui/components/field";
import { Input } from "@ledger-01/ui/components/input";
import { Panel } from "@ledger-01/ui/components/panel";
import Link from "next/link";
import { useEffect, useState } from "react";
import { toast } from "sonner";

export default function AccountPage() {
  const { data } = useQuery(UserService.method.getMe, {});
  const update = useMutation(UserService.method.updateProfile);
  const [name, setName] = useState("");

  useEffect(() => {
    if (data?.user) {
      setName(data.user.name);
    }
  }, [data?.user]);

  return (
    <div className="flex flex-col gap-6">
      <Panel serial="ACC-01 · PROFILE" className="max-w-2xl">
        <form
          className="flex flex-col gap-5"
          onSubmit={async (e) => {
            e.preventDefault();
            await update.mutateAsync({ name: name.trim() });
            toast.success("Saved");
          }}
        >
          <Field label="Email" help="From your Google account.">
            <Input value={data?.user?.email ?? ""} disabled readOnly />
          </Field>
          <Field label="Name" htmlFor="acc-name">
            <Input
              id="acc-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={80}
              required
            />
          </Field>
          <Button type="submit" className="self-start" disabled={update.isPending || !name.trim()}>
            Save
          </Button>
        </form>
      </Panel>
      <Panel serial="ACC-02 · HOUSEHOLD" recess className="max-w-2xl">
        <p className="text-body-sm text-ink">
          Household <strong>{data?.household?.name}</strong>. Members, invites and the base currency
          are in{" "}
          <Link href="/settings/household" className="text-signal-ink underline">
            Household
          </Link>
          .
        </p>
      </Panel>
    </div>
  );
}
