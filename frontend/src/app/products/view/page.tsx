"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { api, errorMessage } from "@/lib/api";
import type { Product } from "@/lib/types";
import { withAsyncData, type AsyncDataProps } from "@/hoc/withAsyncData";
import { withErrorBoundary } from "@/hoc/withErrorBoundary";
import { ProductForm } from "@/components/ProductForm";
import { ErrorMessage } from "@/components/ErrorMessage";
import { Spinner } from "@/components/Spinner";

// Lazy loading: the details grid is split into its own chunk and fetched on demand.
const DetailsTable = dynamic(() => import("@/components/DetailsTable"), {
  ssr: false,
  loading: () => <Spinner label="Cargando detalles…" />,
});

function ProductDetailView({ data: product, reload }: { id: string } & AsyncDataProps<Product>) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function remove() {
    if (!confirm(`¿Eliminar "${product.name}" y todos sus detalles?`)) return;
    try {
      await api.deleteProduct(product.id);
      router.push("/");
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  return (
    <div className="space-y-8">
      <Link href="/" className="text-sm text-indigo-600 hover:underline">
        ← Volver al catálogo
      </Link>
      {error && <ErrorMessage message={error} />}
      <section className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
        {editing ? (
          <ProductForm
            initial={product}
            submitLabel="Guardar cambios"
            onSubmit={async (input) => {
              await api.updateProduct(product.id, input);
              setEditing(false);
              reload();
            }}
            onCancel={() => setEditing(false)}
          />
        ) : (
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="text-2xl font-bold">{product.name}</h1>
              {product.description && <p className="mt-2 text-slate-600">{product.description}</p>}
              <p className="mt-3 text-xs text-slate-500">
                Actualizado {new Date(product.updatedAt).toLocaleString("es")}
              </p>
            </div>
            <div className="flex gap-2">
              <button type="button" onClick={() => setEditing(true)} className="btn-secondary">
                Editar
              </button>
              <button type="button" onClick={remove} className="btn-danger">
                Eliminar
              </button>
            </div>
          </div>
        )}
      </section>
      <DetailsTable productId={product.id} details={product.details} onChange={reload} />
    </div>
  );
}

const ProductView = withErrorBoundary(
  withAsyncData<{ id: string }, Product>(({ id }) => api.getProduct(id))(ProductDetailView),
);

function ProductFromQuery() {
  const id = useSearchParams().get("id");
  if (!id) return <ErrorMessage message="Falta el identificador del producto." />;
  return <ProductView id={id} />;
}

export default function ProductPage() {
  // useSearchParams requires a Suspense boundary in a statically exported app.
  return (
    <Suspense fallback={<Spinner />}>
      <ProductFromQuery />
    </Suspense>
  );
}
