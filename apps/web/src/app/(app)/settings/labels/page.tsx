"use client";

import { useMutation } from "@connectrpc/connect-query";
import { type Label, LabelService } from "@ledger-01/api";
import { Button } from "@ledger-01/ui/components/button";
import { ColorDot } from "@ledger-01/ui/components/category-icon";
import { Field } from "@ledger-01/ui/components/field";
import { Input } from "@ledger-01/ui/components/input";
import { Panel } from "@ledger-01/ui/components/panel";
import { Popover, PopoverContent, PopoverTrigger } from "@ledger-01/ui/components/popover";
import { categoryColors } from "@ledger-01/ui/lib/category-icons";
import { Check, Pencil, Plus, Trash2, X } from "lucide-react";
import { useState } from "react";
import { ColorPicker } from "@/components/appearance-pickers";
import { useLabels } from "@/lib/api/hooks";

export default function LabelsPage() {
  const { labels } = useLabels();
  const create = useMutation(LabelService.method.createLabel);
  const [name, setName] = useState("");
  const [color, setColor] = useState<string>(categoryColors[0]);

  return (
    <div className="flex flex-col gap-6">
      <Panel serial="LBL-01 · NEW">
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={async (e) => {
            e.preventDefault();
            await create.mutateAsync({ name: name.trim(), color });
            setName("");
          }}
        >
          <Field label="New label" htmlFor="label-name" className="min-w-48 flex-1">
            <Input
              id="label-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={30}
              placeholder="e.g. Coffee"
            />
          </Field>
          <ColorButton value={color} onChange={setColor} />
          <Button type="submit" disabled={!name.trim() || create.isPending}>
            <Plus strokeWidth={1.25} /> Add label
          </Button>
        </form>
      </Panel>

      <Panel serial="LBL-02 · ALL">
        {labels.length === 0 ? (
          <p className="text-body-sm text-ink-soft">
            No labels yet. Labels tag transactions across categories, like “Trip to Rio” or
            “Reimbursable”.
          </p>
        ) : (
          <ul className="flex flex-col">
            {labels.map((l) => (
              <LabelRow key={l.id} label={l} />
            ))}
          </ul>
        )}
      </Panel>
    </div>
  );
}

function ColorButton({ value, onChange }: { value: string; onChange: (c: string) => void }) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button type="button" variant="secondary" size="icon" aria-label="Label color">
          <ColorDot color={value} className="size-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72">
        <ColorPicker value={value} onChange={onChange} />
      </PopoverContent>
    </Popover>
  );
}

function LabelRow({ label }: { label: Label }) {
  const update = useMutation(LabelService.method.updateLabel);
  const del = useMutation(LabelService.method.deleteLabel);
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(label.name);

  return (
    <li className="flex items-center gap-3 border-b border-seam/60 py-2 last:border-b-0">
      <ColorButton
        value={label.color}
        onChange={(color) => update.mutate({ id: label.id, color })}
      />
      {editing ? (
        <form
          className="flex flex-1 items-center gap-2"
          onSubmit={async (e) => {
            e.preventDefault();
            await update.mutateAsync({ id: label.id, name: name.trim() });
            setEditing(false);
          }}
        >
          <Input
            aria-label="Label name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={30}
            autoFocus
            className="h-9"
          />
          <Button type="submit" size="icon-sm" variant="ghost" aria-label="Save">
            <Check strokeWidth={1.25} />
          </Button>
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            aria-label="Cancel"
            onClick={() => setEditing(false)}
          >
            <X strokeWidth={1.25} />
          </Button>
        </form>
      ) : (
        <>
          <span className="flex-1 text-body-md text-ink">{label.name}</span>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={`Rename ${label.name}`}
            onClick={() => setEditing(true)}
          >
            <Pencil strokeWidth={1.25} />
          </Button>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={`Delete ${label.name}`}
            onClick={() => del.mutate({ id: label.id })}
          >
            <Trash2 strokeWidth={1.25} />
          </Button>
        </>
      )}
    </li>
  );
}
