"use client";

import { timestampDate } from "@bufbuild/protobuf/wkt";
import { createConnectQueryKey, useMutation, useQuery } from "@connectrpc/connect-query";
import { HouseholdService, type Member } from "@ledger-01/api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@ledger-01/ui/components/alert-dialog";
import { Badge } from "@ledger-01/ui/components/badge";
import { Button } from "@ledger-01/ui/components/button";
import { Field } from "@ledger-01/ui/components/field";
import { Input } from "@ledger-01/ui/components/input";
import { Panel } from "@ledger-01/ui/components/panel";
import { Skeleton } from "@ledger-01/ui/components/skeleton";
import { fault } from "@ledger-01/ui/lib/fault";
import { useQueryClient } from "@tanstack/react-query";
import { LogOut, UserPlus, UserX, X } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { toast } from "sonner";
import { CurrencySelect } from "@/components/currency-select";
import { MemberAvatar } from "@/components/member-avatar";
import { useHousehold } from "@/lib/api/hooks";

const dateFormat = new Intl.DateTimeFormat("en", { dateStyle: "medium" });
const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Household settings, members, invites and (when in more than one) the household switcher. */
export default function HouseholdPage() {
  const id = useId();
  const queryClient = useQueryClient();
  const { household, isOwner } = useHousehold();
  const members = useQuery(HouseholdService.method.listMembers, {});
  const households = useQuery(HouseholdService.method.listHouseholds, {});
  const update = useMutation(HouseholdService.method.updateHousehold);
  const invite = useMutation(HouseholdService.method.inviteMember);
  const cancelInvite = useMutation(HouseholdService.method.cancelInvite);
  const remove = useMutation(HouseholdService.method.removeMember);
  const leave = useMutation(HouseholdService.method.leaveHousehold);
  const switchTo = useMutation(HouseholdService.method.switchHousehold);

  const [name, setName] = useState("");
  const [currency, setCurrency] = useState("USD");
  const [email, setEmail] = useState("");
  const [emailError, setEmailError] = useState<string>();
  const [removing, setRemoving] = useState<Member>();
  const [leaving, setLeaving] = useState(false);

  useEffect(() => {
    if (household) {
      setName(household.name);
      setCurrency(household.baseCurrency);
    }
  }, [household]);

  const refreshMembers = () =>
    queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: HouseholdService, cardinality: undefined }),
    });
  // Everything on screen belongs to the active household, so a switch starts the app over.
  const restart = () => window.location.assign("/dashboard");

  const list = members.data?.members ?? [];
  const former = members.data?.formerMembers ?? [];
  const invites = members.data?.invites ?? [];
  const mine = households.data?.households ?? [];

  return (
    <div className="flex flex-col gap-6">
      <Panel serial="HH-01 · HOUSEHOLD" className="max-w-2xl">
        <form
          className="flex flex-col gap-5"
          onSubmit={async (e) => {
            e.preventDefault();
            await update.mutateAsync({ name: name.trim(), baseCurrency: currency });
            // Totals everywhere are in the base currency.
            await queryClient.invalidateQueries();
            toast.success("Saved");
          }}
        >
          <Field label="Name" htmlFor={`${id}-name`}>
            <Input
              id={`${id}-name`}
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={80}
              required
              disabled={!isOwner}
            />
          </Field>
          <Field
            label="Base currency"
            htmlFor={`${id}-currency`}
            help="Household-wide totals, reports and budgets use it, for everyone in the household."
          >
            {isOwner ? (
              <CurrencySelect id={`${id}-currency`} value={currency} onChange={setCurrency} />
            ) : (
              <Input id={`${id}-currency`} value={currency} disabled readOnly />
            )}
          </Field>
          {isOwner ? (
            <Button
              type="submit"
              className="self-start"
              disabled={update.isPending || !name.trim()}
            >
              Save
            </Button>
          ) : (
            <p className="text-body-sm text-ink-soft">Only the owner can change these.</p>
          )}
        </form>
      </Panel>

      <Panel serial="HH-02 · MEMBERS" className="max-w-2xl">
        {members.isPending ? (
          <Skeleton className="h-10" />
        ) : (
          <ul className="flex flex-col">
            {list.map((m) => (
              <li
                key={m.userId}
                className="flex items-center gap-3 border-b border-seam/60 py-3 last:border-b-0"
              >
                <MemberAvatar member={m} />
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-body-md text-ink">{m.name || m.email}</span>
                  <span className="truncate text-body-sm text-ink-soft">{m.email}</span>
                </div>
                <Badge variant={m.role === "owner" ? "tint" : "default"}>
                  {m.role === "owner" ? "Owner" : "Member"}
                </Badge>
                {isOwner && m.role !== "owner" && (
                  <Button
                    variant="secondary"
                    size="sm"
                    aria-label={`Remove ${m.name || m.email}`}
                    onClick={() => setRemoving(m)}
                  >
                    <UserX strokeWidth={1.25} /> Remove
                  </Button>
                )}
              </li>
            ))}
            {invites.map((inv) => (
              <li
                key={inv.email}
                className="flex items-center gap-3 border-b border-seam/60 py-3 last:border-b-0"
              >
                <MemberAvatar member={{ name: "", email: inv.email, avatarUrl: "" }} />
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-body-md text-ink">{inv.email}</span>
                  <span className="truncate text-body-sm text-ink-soft">
                    Invited
                    {inv.createdAt ? ` ${dateFormat.format(timestampDate(inv.createdAt))}` : ""}
                    {inv.allowed
                      ? " · joins on their next sign-in"
                      : " · not on this server's allowlist yet, so they can't sign in"}
                  </span>
                </div>
                <Badge variant={inv.allowed ? "outline" : "destructive"}>Invited</Badge>
                {isOwner && (
                  <Button
                    variant="secondary"
                    size="sm"
                    aria-label={`Cancel invite for ${inv.email}`}
                    disabled={cancelInvite.isPending}
                    onClick={async () => {
                      await cancelInvite.mutateAsync({ email: inv.email });
                      await refreshMembers();
                      toast.success("Invite cancelled");
                    }}
                  >
                    <X strokeWidth={1.25} /> Cancel
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
        {isOwner && (
          <form
            noValidate
            className="mt-5 flex flex-col gap-3 sm:flex-row sm:items-start"
            onSubmit={async (e) => {
              e.preventDefault();
              const value = email.trim();
              if (!EMAIL.test(value)) {
                setEmailError("Enter an email address.");
                fault(`${id}-email`);
                return;
              }
              setEmailError(undefined);
              const res = await invite.mutateAsync({ email: value });
              setEmail("");
              await refreshMembers();
              toast.success(
                res.invite?.allowed
                  ? `Invited ${value}. They join when they next sign in.`
                  : `Invited ${value}. Add them to ALLOWED_EMAILS so they can sign in.`,
              );
            }}
          >
            <Field
              label="Invite by email"
              htmlFor={`${id}-email`}
              error={emailError}
              help="They join this household when they sign in with Google. Nothing is emailed: let them know."
              className="flex-1"
            >
              <Input
                id={`${id}-email`}
                type="email"
                inputMode="email"
                autoComplete="off"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </Field>
            <Button type="submit" className="sm:mt-7" disabled={invite.isPending}>
              <UserPlus strokeWidth={1.25} /> Invite
            </Button>
          </form>
        )}
        {!isOwner && household && (
          <Button variant="secondary" className="mt-5" onClick={() => setLeaving(true)}>
            <LogOut strokeWidth={1.25} /> Leave household
          </Button>
        )}
      </Panel>

      {former.length > 0 && (
        <Panel serial="HH-03 · FORMER" recess className="max-w-2xl">
          <p className="mb-2 text-body-sm text-ink">
            These people left but entered transactions here. Their transactions stay.
          </p>
          <ul className="flex flex-col">
            {former.map((m) => (
              <li key={m.userId} className="flex items-center gap-3 py-2">
                <MemberAvatar member={m} />
                <span className="truncate text-body-sm text-ink">
                  {m.name || m.email} · {m.email}
                </span>
              </li>
            ))}
          </ul>
        </Panel>
      )}

      {mine.length > 1 && (
        <Panel serial="HH-04 · SWITCH" className="max-w-2xl">
          <p className="mb-2 text-body-sm text-ink">You belong to more than one household.</p>
          <ul className="flex flex-col">
            {mine.map((h) => (
              <li
                key={h.id}
                className="flex items-center gap-3 border-b border-seam/60 py-3 last:border-b-0"
              >
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-body-md text-ink">{h.name}</span>
                  <span className="truncate text-body-sm text-ink-soft">
                    {h.role === "owner" ? "Owner" : "Member"} · {h.memberCount}{" "}
                    {h.memberCount === 1 ? "person" : "people"}
                  </span>
                </div>
                {h.active ? (
                  <Badge variant="signal">Current</Badge>
                ) : (
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={switchTo.isPending}
                    onClick={async () => {
                      await switchTo.mutateAsync({ householdId: h.id });
                      restart();
                    }}
                  >
                    Switch
                  </Button>
                )}
              </li>
            ))}
          </ul>
        </Panel>
      )}

      <AlertDialog open={removing !== undefined} onOpenChange={(o) => !o && setRemoving(undefined)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {removing?.name || removing?.email}?</AlertDialogTitle>
            <AlertDialogDescription>
              They lose access to this household right away. The transactions they entered stay.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                if (!removing) return;
                await remove.mutateAsync({ userId: removing.userId });
                await refreshMembers();
                toast.success(`Removed ${removing.name || removing.email}`);
              }}
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={leaving} onOpenChange={setLeaving}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Leave {household?.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              You lose access to its wallets. The transactions you entered stay. To come back, the
              owner has to invite you again.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                await leave.mutateAsync({});
                restart();
              }}
            >
              Leave
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
