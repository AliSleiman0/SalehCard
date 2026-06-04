import { lazy } from 'react'
import { Routes, Route } from 'react-router-dom'
import { AdminLayout } from '@/components/layout/AdminLayout'
import { RequireAdmin } from './RequireAdmin'

const LoginPage = lazy(() => import('@/features/auth/pages/LoginPage'))
const DashboardPage = lazy(() => import('@/features/dashboard/pages/DashboardPage'))
const ProductListPage = lazy(() => import('@/features/products/pages/ProductListPage'))
const ProductEditPage = lazy(() => import('@/features/products/pages/ProductEditPage'))
const InventoryPage = lazy(() => import('@/features/inventory/pages/InventoryPage'))
const OrderListPage = lazy(() => import('@/features/orders/pages/OrderListPage'))
const OrderDetailPage = lazy(() => import('@/features/orders/pages/OrderDetailPage'))
const UserListPage = lazy(() => import('@/features/users/pages/UserListPage'))
const UserDetailPage = lazy(() => import('@/features/users/pages/UserDetailPage'))
const ResellerListPage = lazy(() => import('@/features/resellers/pages/ResellerListPage'))
const ResellerDetailPage = lazy(() => import('@/features/resellers/pages/ResellerDetailPage'))
const FinancePage = lazy(() => import('@/features/finance/pages/FinancePage'))
const PromoListPage = lazy(() => import('@/features/promos/pages/PromoListPage'))
const PromoEditPage = lazy(() => import('@/features/promos/pages/PromoEditPage'))
const ReviewsPage = lazy(() => import('@/features/reviews/pages/ReviewsPage'))
const SettingsPage = lazy(() => import('@/features/settings/pages/SettingsPage'))
const NotFoundPage = lazy(() => import('@/features/misc/NotFoundPage'))

export default function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <RequireAdmin>
            <AdminLayout />
          </RequireAdmin>
        }
      >
        <Route path="/" element={<DashboardPage />} />
        <Route path="/products" element={<ProductListPage />} />
        <Route path="/products/new" element={<ProductEditPage />} />
        <Route path="/products/:id/edit" element={<ProductEditPage />} />
        <Route path="/inventory" element={<InventoryPage />} />
        <Route path="/orders" element={<OrderListPage />} />
        <Route path="/orders/:id" element={<OrderDetailPage />} />
        <Route path="/users" element={<UserListPage />} />
        <Route path="/users/:id" element={<UserDetailPage />} />
        <Route path="/resellers" element={<ResellerListPage />} />
        <Route path="/resellers/:id" element={<ResellerDetailPage />} />
        <Route path="/finance" element={<FinancePage />} />
        <Route path="/promos" element={<PromoListPage />} />
        <Route path="/promos/new" element={<PromoEditPage />} />
        <Route path="/promos/:id/edit" element={<PromoEditPage />} />
        <Route path="/reviews" element={<ReviewsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
