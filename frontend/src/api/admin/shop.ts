import { apiClient } from '../client'

export interface AdminShopProduct {
  id: number
  name: string
  description?: string
  price: number
  currency: string
  redeem_type: string
  redeem_value: number
  group_id?: number
  validity_days: number
  stock_count: number
  is_active: boolean
  sort_order: number
  creem_product_id: string
  created_at: string
}

export interface ShopProductStock {
  id: number
  product_id: number
  redeem_code_id: number
  status: string
  order_id?: number
  created_at: string
}

export interface CreateProductRequest {
  name: string
  description?: string
  price: number
  currency?: string
  redeem_type: string
  redeem_value?: number
  group_id?: number
  validity_days?: number
  is_active?: boolean
  sort_order?: number
  creem_product_id?: string
}

export const adminShopAPI = {
  listProducts: () => apiClient.get<AdminShopProduct[]>('/admin/shop/products').then(r => r.data),
  createProduct: (req: CreateProductRequest) =>
    apiClient.post<AdminShopProduct>('/admin/shop/products', req).then(r => r.data),
  updateProduct: (id: number, req: Partial<CreateProductRequest>) =>
    apiClient.put<AdminShopProduct>(`/admin/shop/products/${id}`, req).then(r => r.data),
  deleteProduct: (id: number) => apiClient.delete(`/admin/shop/products/${id}`),
  getStockList: (productId: number) =>
    apiClient.get<ShopProductStock[]>(`/admin/shop/products/${productId}/stocks`).then(r => r.data),
  addStock: (productId: number, count: number) =>
    apiClient.post<{ added: number }>(`/admin/shop/products/${productId}/stocks`, { count }).then(r => r.data),
  deleteStock: (stockId: number) => apiClient.delete(`/admin/shop/stocks/${stockId}`),
}

export default adminShopAPI
