"use client";

import { Fragment, useState, type FormEvent } from "react";
import { api, errorMessage } from "@/lib/api";
import type { DetailInput, ProductDetail } from "@/lib/types";
import { ErrorMessage } from "./ErrorMessage";

interface Props {
  productId: string;
  details: ProductDetail[];
  onChange: () => void;
}

const empty: DetailInput = { sku: "", price: 0, stock: 0 };
const money = new Intl.NumberFormat("es-EC", { style: "currency", currency: "USD" });

/** Detail (variants) grid of a product. Loaded lazily from the product view. */
export default function DetailsTable({ productId, details, onChange }: Props) {
  const [editing, setEditing] = useState<string | null>(null); // detail id, "new" or null
  const [form, setForm] = useState<DetailInput>(empty);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  function start(id: string, initial: DetailInput) {
    setEditing(id);
    setForm(initial);
    setError(null);
  }

  async function run(action: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      setEditing(null);
      onChange();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    void run(() =>
      editing === "new"
        ? api.addDetail(productId, form)
        : api.updateDetail(productId, editing as string, form),
    );
  }

  const formRow = (
    <tr className="bg-indigo-50/50">
      <td className="px-3 py-2">
        <input
          form="detail-form"
          aria-label="SKU"
          value={form.sku}
          onChange={(e) => setForm({ ...form, sku: e.target.value })}
          required
          maxLength={64}
          className="input"
        />
      </td>
      <td className="px-3 py-2">
        <input
          form="detail-form"
          aria-label="Precio"
          type="number"
          min={0}
          step="0.01"
          value={form.price}
          onChange={(e) => setForm({ ...form, price: e.target.valueAsNumber || 0 })}
          className="input"
        />
      </td>
      <td className="px-3 py-2">
        <input
          form="detail-form"
          aria-label="Stock"
          type="number"
          min={0}
          step={1}
          value={form.stock}
          onChange={(e) => setForm({ ...form, stock: e.target.valueAsNumber || 0 })}
          className="input"
        />
      </td>
      <td className="space-x-2 whitespace-nowrap px-3 py-2 text-right">
        <button form="detail-form" type="submit" disabled={busy} className="btn-primary">
          Guardar
        </button>
        <button type="button" onClick={() => setEditing(null)} className="btn-secondary">
          Cancelar
        </button>
      </td>
    </tr>
  );

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Detalles ({details.length})</h2>
        {editing === null && (
          <button type="button" onClick={() => start("new", empty)} className="btn-primary">
            Añadir detalle
          </button>
        )}
      </div>
      {error && <ErrorMessage message={error} />}
      <form id="detail-form" onSubmit={submit} />
      <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50 text-left text-xs uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-3 py-2">SKU</th>
              <th className="px-3 py-2">Precio</th>
              <th className="px-3 py-2">Stock</th>
              <th className="px-3 py-2" />
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {details.map((d) =>
              editing === d.id ? (
                <Fragment key={d.id}>{formRow}</Fragment>
              ) : (
                <tr key={d.id}>
                  <td className="px-3 py-2 font-mono">{d.sku}</td>
                  <td className="px-3 py-2">{money.format(d.price)}</td>
                  <td className="px-3 py-2">{d.stock}</td>
                  <td className="space-x-3 whitespace-nowrap px-3 py-2 text-right">
                    <button
                      type="button"
                      disabled={editing !== null}
                      onClick={() => start(d.id, { sku: d.sku, price: d.price, stock: d.stock })}
                      className="text-indigo-600 hover:underline disabled:opacity-40"
                    >
                      Editar
                    </button>
                    <button
                      type="button"
                      disabled={busy || editing !== null}
                      onClick={() => {
                        if (confirm(`¿Eliminar el detalle ${d.sku}?`)) void run(() => api.deleteDetail(productId, d.id));
                      }}
                      className="text-red-600 hover:underline disabled:opacity-40"
                    >
                      Eliminar
                    </button>
                  </td>
                </tr>
              ),
            )}
            {editing === "new" && formRow}
            {details.length === 0 && editing !== "new" && (
              <tr>
                <td colSpan={4} className="px-3 py-6 text-center text-slate-500">
                  Este producto aún no tiene detalles.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
