export interface ProductDetail {
  id: string;
  productId: string;
  sku: string;
  price: number;
  stock: number;
}

export interface Product {
  id: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  details: ProductDetail[];
}

export type ProductInput = Pick<Product, "name" | "description">;
export type DetailInput = Pick<ProductDetail, "sku" | "price" | "stock">;
