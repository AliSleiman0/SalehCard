import { lazy } from 'react'
import { Routes, Route } from 'react-router-dom'
import { AdminLayout } from '@/components/layout/AdminLayout'
import { RequireAdmin } from './RequireAdmin'
import { RequireDomain } from './RequireDomain'

const LoginPage = lazy(() => import('@/features/auth/pages/LoginPage'))
const DashboardPage = lazy(() => import('@/features/dashboard/pages/DashboardPage'))
const ProductListPage = lazy(() => import('@/features/products/pages/ProductListPage'))
const ProductEditPage = lazy(() => import('@/features/products/pages/ProductEditPage'))
const CategoriesPage = lazy(() => import('@/features/categories/pages/CategoriesPage'))
const InventoryPage = lazy(() => import('@/features/inventory/pages/InventoryPage'))
const OrderListPage = lazy(() => import('@/features/orders/pages/OrderListPage'))
const OrderDetailPage = lazy(() => import('@/features/orders/pages/OrderDetailPage'))
const UserListPage = lazy(() => import('@/features/users/pages/UserListPage'))
const UserDetailPage = lazy(() => import('@/features/users/pages/UserDetailPage'))
const ResellerListPage = lazy(() => import('@/features/resellers/pages/ResellerListPage'))
const ResellerDetailPage = lazy(() => import('@/features/resellers/pages/ResellerDetailPage'))
const FinancePage = lazy(() => import('@/features/finance/pages/FinancePage'))
const TopupsPage = lazy(() => import('@/features/topups/pages/TopupsPage'))
const PaymentsPage = lazy(() => import('@/features/payments/pages/PaymentsPage'))
const BridgePage = lazy(() => import('@/features/bridge/pages/BridgePage'))
const SuppliersPage = lazy(() => import('@/features/suppliers/pages/SuppliersPage'))
const PromoListPage = lazy(() => import('@/features/promos/pages/PromoListPage'))
const PromoEditPage = lazy(() => import('@/features/promos/pages/PromoEditPage'))
const OffersListPage = lazy(() => import('@/features/offers/pages/OffersListPage'))
const OffersEditPage = lazy(() => import('@/features/offers/pages/OffersEditPage'))
const ExpensesListPage = lazy(() => import('@/features/expenses/pages/ExpensesListPage'))
const ExpenseEditPage = lazy(() => import('@/features/expenses/pages/ExpenseEditPage'))
const ReviewsPage = lazy(() => import('@/features/reviews/pages/ReviewsPage'))
const KycPage = lazy(() => import('@/features/kyc/pages/KycPage'))
const AuditLogPage = lazy(() => import('@/features/audit/pages/AuditLogPage'))
const SettingsPage = lazy(() => import('@/features/settings/pages/SettingsPage'))
const RolesPage = lazy(() => import('@/features/roles/pages/RolesPage'))
const NotFoundPage = lazy(() => import('@/features/misc/NotFoundPage'))

// RBAC route guard: renders el only when the admin's role can view `domain`
// (redirecting to their first permitted page otherwise). Domain keys mirror
// nav.ts and the backend catalog (modules/role/permissions.go).
function guard(domain: string, el: React.ReactNode) {
  return <RequireDomain domain={domain}>{el}</RequireDomain>
}

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
        <Route path="/" element={guard('dashboard', <DashboardPage />)} />
        <Route path="/products" element={guard('products', <ProductListPage />)} />
        <Route path="/products/new" element={guard('products', <ProductEditPage />)} />
        <Route path="/products/:id/edit" element={guard('products', <ProductEditPage />)} />
        <Route path="/categories" element={guard('categories', <CategoriesPage />)} />
        <Route path="/inventory" element={guard('inventory', <InventoryPage />)} />
        <Route path="/orders" element={guard('orders', <OrderListPage />)} />
        <Route path="/orders/:id" element={guard('orders', <OrderDetailPage />)} />
        <Route path="/bridge" element={guard('bridge', <BridgePage />)} />
        <Route path="/suppliers" element={guard('suppliers', <SuppliersPage />)} />
        <Route path="/users" element={guard('users', <UserListPage />)} />
        <Route path="/users/:id" element={guard('users', <UserDetailPage />)} />
        <Route path="/resellers" element={guard('resellers', <ResellerListPage />)} />
        <Route path="/resellers/:id" element={guard('resellers', <ResellerDetailPage />)} />
        <Route path="/finance" element={guard('finance', <FinancePage />)} />
        <Route path="/topups" element={guard('topups', <TopupsPage />)} />
        <Route path="/payments" element={guard('payments', <PaymentsPage />)} />
        <Route path="/promos" element={guard('promos', <PromoListPage />)} />
        <Route path="/promos/new" element={guard('promos', <PromoEditPage />)} />
        <Route path="/promos/:id/edit" element={guard('promos', <PromoEditPage />)} />
        <Route path="/offers" element={guard('offers', <OffersListPage />)} />
        <Route path="/offers/new" element={guard('offers', <OffersEditPage />)} />
        <Route path="/offers/:id/edit" element={guard('offers', <OffersEditPage />)} />
        <Route path="/expenses" element={guard('expenses', <ExpensesListPage />)} />
        <Route path="/expenses/new" element={guard('expenses', <ExpenseEditPage />)} />
        <Route path="/expenses/:id/edit" element={guard('expenses', <ExpenseEditPage />)} />
        <Route path="/reviews" element={guard('reviews', <ReviewsPage />)} />
        <Route path="/kyc" element={guard('kyc', <KycPage />)} />
        <Route path="/audit" element={guard('audit', <AuditLogPage />)} />
        <Route path="/settings" element={guard('settings', <SettingsPage />)} />
        <Route
          path="/roles"
          element={
            <RequireDomain superAdmin>
              <RolesPage />
            </RequireDomain>
          }
        />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
