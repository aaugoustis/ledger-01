"use client";

import { OccurrenceScope } from "@ledger-01/api";
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

/**
 * Asks which occurrences of a recurring transaction a change applies to: only the chosen ones, or
 * those and every later one that isn't due yet. Past occurrences never change.
 */
export function ScopeDialog({
  open,
  onOpenChange,
  title,
  description,
  only = "Only this one",
  future = "This and future ones",
  offerOnly = true,
  destructive = false,
  disabled = false,
  onChoose,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  only?: string;
  future?: string;
  /** False when the change only makes sense for the series (a new repeat schedule). */
  offerOnly?: boolean;
  destructive?: boolean;
  disabled?: boolean;
  onChoose: (scope: OccurrenceScope) => void;
}) {
  const variant = destructive ? "destructive" : "default";
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          {offerOnly && (
            <AlertDialogAction
              variant="outline"
              disabled={disabled}
              onClick={() => onChoose(OccurrenceScope.ONLY_THIS)}
            >
              {only}
            </AlertDialogAction>
          )}
          <AlertDialogAction
            variant={variant}
            disabled={disabled}
            onClick={() => onChoose(OccurrenceScope.THIS_AND_FUTURE)}
          >
            {future}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
