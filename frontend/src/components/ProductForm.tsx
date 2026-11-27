"use client";

import { useState, type FormEvent } from "react";
import { errorMessage } from "@/lib/api";
import type { ProductInput } from "@/lib/types";
import { ErrorMessage } from "./ErrorMessage";

interface Props {
  initial?: ProductInput;
  submitLabel: string;
  onSubmit: (input: ProductInput) => Promise<void>;
  onCancel?: () => void;
}

export function ProductForm({ initial, submitLabel, onSubmit, onCancel }: Props) {
  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      await onSubmit({ name, description });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && <ErrorMessage message={error} />}
      <label className="block">
        <span className="text-sm font-medium text-slate-700">Nombre</span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          maxLength={100}
          className="input mt-1"
        />
      </label>
      <label className="block">
        <span className="text-sm font-medium text-slate-700">Descripción</span>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          className="input mt-1"
        />
      </label>
      <div className="flex gap-2">
        <button type="submit" disabled={saving} className="btn-primary">
          {saving ? "Guardando…" : submitLabel}
        </button>
        {onCancel && (
          <button type="button" onClick={onCancel} className="btn-secondary">
            Cancelar
          </button>
        )}
      </div>
    </form>
  );
}
