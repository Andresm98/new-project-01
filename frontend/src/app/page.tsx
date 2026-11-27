"use client";

import Link from "next/link";
import { api } from "@/lib/api";
import type { Product } from "@/lib/types";
import { withAsyncData, type AsyncDataProps } from "@/hoc/withAsyncData";
import { withErrorBoundary } from "@/hoc/withErrorBoundary";

function ProductList({ data }: AsyncDataProps<Product[]>) {
  if (data.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-slate-300 bg-white p-10 text-center text-slate-500">
        No hay productos todavía.{" "}
        <Link href="/products/new/" className="text-indigo-600 hover:underline">
          Crea el primero
        </Link>
        .
      </div>
    );
  }
  return (
    <ul className="grid gap-4 sm:grid-cols-2">
      {data.map((p) => (
        <li key={p.id}>
          <Link
            href={`/products/view/?id=${encodeURIComponent(p.id)}`}
            className="block rounded-lg border border-slate-200 bg-white p-4 shadow-sm transition hover:border-indigo-300 hover:shadow"
          >
            <h2 className="font-semibold">{p.name}</h2>
            {p.description && <p className="mt-1 line-clamp-2 text-sm text-slate-600">{p.description}</p>}
            <p className="mt-3 text-xs text-slate-500">{p.details.length} detalle(s)</p>
          </Link>
        </li>
      ))}
    </ul>
  );
}

const ProductListView = withErrorBoundary(withAsyncData<object, Product[]>(() => api.listProducts())(ProductList));

export default function HomePage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Catálogo</h1>
      <ProductListView />
    </div>
  );
}
