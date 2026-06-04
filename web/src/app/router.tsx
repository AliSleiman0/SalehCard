import { Suspense, lazy } from 'react'
import { Routes, Route } from 'react-router-dom'
import Layout from './Layout'

const HomePage = lazy(() => import('@/features/catalog/pages/HomePage'))
const CategoryPage = lazy(() => import('@/features/catalog/pages/CategoryPage'))
const ProductDetailPage = lazy(() => import('@/features/catalog/pages/ProductDetailPage'))
const CartPage = lazy(() => import('@/features/cart/pages/CartPage'))
const CheckoutPage = lazy(() => import('@/features/checkout/pages/CheckoutPage'))
const OrderSuccessPage = lazy(() => import('@/features/orders/pages/OrderSuccessPage'))
const LoginPage = lazy(() => import('@/features/auth/pages/LoginPage'))
const RegisterPage = lazy(() => import('@/features/auth/pages/RegisterPage'))
const DashboardPage = lazy(() => import('@/features/auth/pages/DashboardPage'))
const WalletPage = lazy(() => import('@/features/wallet/pages/WalletPage'))
const OrdersPage = lazy(() => import('@/features/orders/pages/OrdersPage'))
const OrderDetailPage = lazy(() => import('@/features/orders/pages/OrderDetailPage'))
const SavedIDsPage = lazy(() => import('@/features/auth/pages/SavedIDsPage'))
const ResellerDashboardPage = lazy(() => import('@/features/reseller/pages/ResellerDashboardPage'))

const fallback = (
  <div className="flex items-center justify-center h-screen">Loading...</div>
)

export default function AppRouter() {
  return (
    <Suspense fallback={fallback}>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/category/:slug" element={<CategoryPage />} />
          <Route path="/product/:id" element={<ProductDetailPage />} />
          <Route path="/cart" element={<CartPage />} />
          <Route path="/checkout" element={<CheckoutPage />} />
          <Route path="/order-success/:orderId" element={<OrderSuccessPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/wallet" element={<WalletPage />} />
          <Route path="/orders" element={<OrdersPage />} />
          <Route path="/orders/:id" element={<OrderDetailPage />} />
          <Route path="/saved-ids" element={<SavedIDsPage />} />
          <Route path="/reseller" element={<ResellerDashboardPage />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
