"use client";

import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { ProductForm } from "@/components/ProductForm";

export default function NewProductPage() {
  const router = useRouter();
  return (
    <div className="max-w-xl space-y-6">
      <h1 className="text-2xl font-bold">Nuevo producto</h1>
      <div className="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
        <ProductForm
          submitLabel="Crear"
          onSubmit={async (input) => {
            const p = await api.createProduct(input);
            router.push(`/products/view/?id=${encodeURIComponent(p.id)}`);
          }}
          onCancel={() => router.push("/")}
        />
      </div>
    </div>
  );
}
