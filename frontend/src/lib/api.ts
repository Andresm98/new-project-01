import type { DetailInput, Product, ProductDetail, ProductInput } from "./types";

// Relative by default: nginx proxies /api to the backend, so calls are same-origin.
const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "/api/v1";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${BASE_URL}${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...init?.headers },
    });
  } catch {
    throw new ApiError(0, "network_error", "No se pudo conectar con el servidor.");
  }

  if (res.status === 204) return undefined as T;

  const body: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const err = (body as { error?: { code?: string; message?: string } } | null)?.error;
    throw new ApiError(res.status, err?.code ?? "unknown_error", err?.message ?? `Error HTTP ${res.status}`);
  }
  return body as T;
}

const json = (data: unknown) => JSON.stringify(data);

export const api = {
  listProducts: () => request<Product[]>("/products"),
  getProduct: (id: string) => request<Product>(`/products/${encodeURIComponent(id)}`),
  createProduct: (input: ProductInput) =>
    request<Product>("/products", { method: "POST", body: json(input) }),
  updateProduct: (id: string, input: ProductInput) =>
    request<Product>(`/products/${encodeURIComponent(id)}`, { method: "PUT", body: json(input) }),
  deleteProduct: (id: string) =>
    request<void>(`/products/${encodeURIComponent(id)}`, { method: "DELETE" }),
  addDetail: (productId: string, input: DetailInput) =>
    request<ProductDetail>(`/products/${encodeURIComponent(productId)}/details`, {
      method: "POST",
      body: json(input),
    }),
  updateDetail: (productId: string, detailId: string, input: DetailInput) =>
    request<ProductDetail>(
      `/products/${encodeURIComponent(productId)}/details/${encodeURIComponent(detailId)}`,
      { method: "PUT", body: json(input) },
    ),
  deleteDetail: (productId: string, detailId: string) =>
    request<void>(
      `/products/${encodeURIComponent(productId)}/details/${encodeURIComponent(detailId)}`,
      { method: "DELETE" },
    ),
};

/** Human-readable message for any error thrown by the API layer. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case "not_found":
        return "El recurso no existe.";
      case "duplicate_sku":
        return "Ese SKU ya está en uso.";
      case "validation_error":
        return `Datos inválidos: ${err.message}`;
      default:
        return err.message;
    }
  }
  return err instanceof Error ? err.message : "Error inesperado.";
}
