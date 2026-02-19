import { apiClient } from './client'

export interface ShopProduct {
  id: number
  name: string
  description?: string
  price: number
  currency: string
  redeem_type: string
  stock_count: number
  is_active: boolean
  sort_order: number
}

export interface ShopOrder {
  id: number
  order_no: string
  product_name: string
  amount: number
  currency: string
  payment_method?: string
  status: string
  paid_at?: string
  expires_at?: string
  created_at: string
}

export interface CreateOrderResponse {
  order: ShopOrder
  pay_url: string
}

export const shopAPI = {
  getProducts: () => apiClient.get<ShopProduct[]>('/shop/products').then(r => r.data),
  createOrder: (product_id: number, payment_method: string) =>
    apiClient.post<CreateOrderResponse>('/shop/orders', { product_id, payment_method }).then(r => r.data),
  queryOrder: (orderNo: string) =>
    apiClient.get<ShopOrder>(`/shop/orders/${orderNo}`).then(r => r.data),
}

export default shopAPI
